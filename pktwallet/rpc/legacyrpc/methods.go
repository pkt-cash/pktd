// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2016 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package legacyrpc

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/pkt-cash/pktd/blockchain"
	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/btcjson"
	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/neutrino/banman"
	"github.com/pkt-cash/pktd/pktlog"
	"github.com/pkt-cash/pktd/pktwallet/chain"
	"github.com/pkt-cash/pktd/pktwallet/waddrmgr"
	"github.com/pkt-cash/pktd/pktwallet/wallet"
	"github.com/pkt-cash/pktd/pktwallet/wallet/txauthor"
	"github.com/pkt-cash/pktd/pktwallet/wallet/txrules"
	"github.com/pkt-cash/pktd/pktwallet/wtxmgr"
	"github.com/pkt-cash/pktd/rpcclient"
	"github.com/pkt-cash/pktd/txscript"
	"github.com/pkt-cash/pktd/txscript/params"
	"github.com/pkt-cash/pktd/wire"
	"github.com/pkt-cash/pktd/wire/ruleerror"
)

// confirms returns the number of confirmations for a transaction in a block at
// height txHeight (or -1 for an unconfirmed tx) given the chain height
// curHeight.
func confirms(txHeight, curHeight int32) int32 {
	switch {
	case txHeight == -1, txHeight > curHeight:
		return 0
	default:
		return curHeight - txHeight + 1
	}
}

// requestHandler is a handler function to handle an unmarshaled and parsed
// request into a marshalable response.  If the error is a *btcjson.RPCError
// or any of the above special error classes, the server will respond with
// the JSON-RPC appropiate error code.  All other errors use the wallet
// catch-all error code, btcjson.ErrRPCWallet.
type requestHandler func(interface{}, *wallet.Wallet) (interface{}, er.R)

// requestHandlerChain is a requestHandler that also takes a parameter for
type handlerChain func(interface{}, *wallet.Wallet, chain.Interface) (interface{}, er.R)

type handlerRPC func(interface{}, *wallet.Wallet, *chain.RPCClient) (interface{}, er.R)

type handlerNeutrino func(interface{}, *wallet.Wallet, *chain.NeutrinoClient) (interface{}, er.R)

var rpcHandlers = map[string]struct {
	handler         requestHandler
	handlerChain    handlerChain
	handlerRPC      handlerRPC
	handlerNeutrino handlerNeutrino

	// Function variables cannot be compared against anything but nil, so
	// use a boolean to record whether help generation is necessary.  This
	// is used by the tests to ensure that help can be generated for every
	// implemented method.
	//
	// A single map and this bool is here is used rather than several maps
	// for the unimplemented handlers so every method has exactly one
	// handler function.
	noHelp bool
}{
	// Reference implementation wallet methods (implemented)
	"addmultisigaddress":     {handler: addMultiSigAddress},
	"createmultisig":         {handler: createMultiSig},
	"dumpprivkey":            {handler: dumpPrivKey},
	"getbalance":             {handler: getBalance},
	"getbestblockhash":       {handler: getBestBlockHash},
	"getblockcount":          {handler: getBlockCount},
	"getinfo":                {handlerChain: getInfo},
	"getnewaddress":          {handler: getNewAddress},
	"getreceivedbyaddress":   {handler: getReceivedByAddress},
	"gettransaction":         {handler: getTransaction},
	"help":                   {handler: helpNoChainRPC, handlerRPC: helpWithChainRPC},
	"importprivkey":          {handler: importPrivKey},
	"listlockunspent":        {handler: listLockUnspent},
	"listreceivedbyaddress":  {handler: listReceivedByAddress},
	"listsinceblock":         {handlerChain: listSinceBlock},
	"listtransactions":       {handler: listTransactions},
	"listunspent":            {handler: listUnspent},
	"lockunspent":            {handler: lockUnspent},
	"sendfrom":               {handler: sendFrom},
	"sendmany":               {handler: sendMany},
	"sendtoaddress":          {handler: sendToAddress},
	"settxfee":               {handler: setTxFee},
	"signmessage":            {handler: signMessage},
	"signrawtransaction":     {handlerChain: signRawTransaction},
	"validateaddress":        {handler: validateAddress},
	"verifymessage":          {handler: verifyMessage},
	"walletlock":             {handler: walletLock},
	"walletpassphrase":       {handler: walletPassphrase},
	"walletpassphrasechange": {handler: walletPassphraseChange},

	// Extensions to the reference client JSON-RPC API
	"getbestblock":          {handler: getBestBlock},
	"setnetworkstewardvote": {handler: setNetworkStewardVote},
	"getnetworkstewardvote": {handler: getNetworkStewardVote},
	"addp2shscript":         {handler: addP2shScript},
	"createtransaction":     {handler: createTransaction},
	"resync":                {handler: resync},
	"stopresync":            {handler: stopResync},
	"getaddressbalances":    {handler: getAddressBalances},
	"getwalletseed":         {handler: getWalletSeed},
	"getsecret":             {handler: getSecret},
	"walletmempool":         {handler: walletMempool},
	// This was an extension but the reference implementation added it as
	// well, but with a different API (no account parameter).  It's listed
	// here because it hasn't been update to use the reference
	// implemenation's API.
	"getunconfirmedbalance":   {handler: getUnconfirmedBalance},
	"listaddresstransactions": {handler: listAddressTransactions},
	"listalltransactions":     {handler: listAllTransactions},
	"walletislocked":          {handler: walletIsLocked},
}

// lazyHandler is a closure over a requestHandler or passthrough request with
// the RPC server's wallet and chain server variables as part of the closure
// context.
type lazyHandler func() (interface{}, er.R)

// lazyApplyHandler looks up the best request handler func for the method,
// returning a closure that will execute it with the (required) wallet and
// (optional) consensus RPC server.  If no handlers are found and the
// chainClient is not nil, the returned handler performs RPC passthrough.
func lazyApplyHandler(request *btcjson.Request, w *wallet.Wallet, chainClient chain.Interface) lazyHandler {
	hndlr, ok := rpcHandlers[request.Method]
	var err er.R
	unm := func(f func(interface{}) (interface{}, er.R)) func() (interface{}, er.R) {
		return func() (interface{}, er.R) {
			if cmd, err := btcjson.UnmarshalCmd(request); err != nil {
				return nil, btcjson.ErrRPCInvalidRequest.Default()
			} else {
				return f(cmd)
			}
		}
	}
	if w == nil {
		err = btcjson.ErrRPCMisc.New("The wallet is not loaded", nil)
	} else if !ok {
		err = btcjson.ErrRPCMisc.New(
			fmt.Sprintf("[%s] does not seem to be a wallet comand", request.Method), nil)
	} else if chainClient == nil {
		// fallthrough
	} else if rpc, ok := chainClient.(*chain.RPCClient); ok && hndlr.handlerRPC != nil {
		return unm(func(cmd interface{}) (interface{}, er.R) { return hndlr.handlerRPC(cmd, w, rpc) })
	} else if neut, ok := chainClient.(*chain.NeutrinoClient); ok && hndlr.handlerNeutrino != nil {
		return unm(func(cmd interface{}) (interface{}, er.R) { return hndlr.handlerNeutrino(cmd, w, neut) })
	} else if hndlr.handlerChain != nil {
		return unm(func(cmd interface{}) (interface{}, er.R) { return hndlr.handlerChain(cmd, w, chainClient) })
	}
	if err != nil {
	} else if hndlr.handler != nil {
		return unm(func(cmd interface{}) (interface{}, er.R) { return hndlr.handler(cmd, w) })
	} else if hndlr.handlerChain != nil {
		err = btcjson.ErrRPCMisc.New("This RPC requires a connection to the blockchain", nil)
	} else if hndlr.handlerNeutrino != nil {
		err = btcjson.ErrRPCMisc.New("This RPC requires neutrino backend (not --userpc mode)", nil)
	} else if hndlr.handlerRPC != nil {
		err = btcjson.ErrRPCMisc.New("This RPC requires RPC backend (--userpc mode)", nil)
	}
	if err == nil {
		err = btcjson.ErrRPCMisc.New("This RPC is has no handlers (internal error)", nil)
	}
	return func() (interface{}, er.R) {
		return nil, err
	}
}

// makeResponse makes the JSON-RPC response struct for the result and error
// returned by a requestHandler.  The returned response is not ready for
// marshaling and sending off to a client, but must be
func makeResponse(id, result interface{}, err er.R) btcjson.Response {
	idPtr := idPointer(id)
	if err != nil {
		return btcjson.Response{
			ID:    idPtr,
			Error: btcjson.SerializeError(jsonError(err)),
		}
	}
	resultBytes, errr := jsoniter.Marshal(result)
	if errr != nil {
		return btcjson.Response{
			ID: idPtr,
			Error: btcjson.SerializeError(
				btcjson.ErrRPCInternal.New("Unexpected error marshalling result", nil)),
		}
	}
	return btcjson.Response{
		ID:     idPtr,
		Result: jsoniter.RawMessage(resultBytes),
	}
}

// jsonError creates a JSON-RPC error from the Go error.
func jsonError(err er.R) er.R {
	if waddrmgr.ErrWrongPassphrase.Is(err) {
		return btcjson.ErrRPCWalletPassphraseIncorrect.New("", err)
	}
	return err
}

// makeMultiSigScript is a helper function to combine common logic for
// AddMultiSig and CreateMultiSig.
func makeMultiSigScript(w *wallet.Wallet, keys []string, nRequired int) ([]byte, er.R) {
	keysesPrecious := make([]*btcutil.AddressPubKey, len(keys))

	// The address list will made up either of addreseses (pubkey hash), for
	// which we need to look up the keys in wallet, straight pubkeys, or a
	// mixture of the two.
	for i, a := range keys {
		// try to parse as pubkey address
		a, err := decodeAddress(a, w.ChainParams())
		if err != nil {
			return nil, err
		}

		switch addr := a.(type) {
		case *btcutil.AddressPubKey:
			keysesPrecious[i] = addr
		default:
			pubKey, err := w.PubKeyForAddress(addr)
			if err != nil {
				return nil, err
			}
			pubKeyAddr, err := btcutil.NewAddressPubKey(
				pubKey.SerializeCompressed(), w.ChainParams())
			if err != nil {
				return nil, err
			}
			keysesPrecious[i] = pubKeyAddr
		}
	}

	return txscript.MultiSigScript(keysesPrecious, nRequired)
}

func addP2shScript(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.AddP2shScriptCmd)
	script, err := decodeHexStr(cmd.Script)
	if err != nil {
		return nil, err
	}
	if cmd.Segwit {
		p2shAddr, err := w.ImportP2WSHRedeemScript(script)
		if err != nil {
			return nil, err
		}
		return p2shAddr.EncodeAddress(), nil
	}
	p2shAddr, err := w.ImportP2SHRedeemScript(script)
	if err != nil {
		return nil, err
	}
	return p2shAddr.EncodeAddress(), nil
}

// addMultiSigAddress handles an addmultisigaddress request by adding a
// multisig address to the given wallet.
func addMultiSigAddress(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.AddMultisigAddressCmd)

	secp256k1Addrs := make([]btcutil.Address, len(cmd.Keys))
	for i, k := range cmd.Keys {
		addr, err := decodeAddress(k, w.ChainParams())
		if err != nil {
			return nil, errParse("unable to decode address", err)
		}
		secp256k1Addrs[i] = addr
	}

	script, err := w.MakeMultiSigScript(secp256k1Addrs, cmd.NRequired)
	if err != nil {
		return nil, err
	}

	p2shAddr, err := w.ImportP2SHRedeemScript(script)
	if err != nil {
		return nil, err
	}

	return p2shAddr.EncodeAddress(), nil
}

// createMultiSig handles an createmultisig request by returning a
// multisig address for the given inputs.
func createMultiSig(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.CreateMultisigCmd)

	script, err := makeMultiSigScript(w, cmd.Keys, cmd.NRequired)
	if err != nil {
		return nil, errParse("unable to parse keys", err)
	}

	address, err := btcutil.NewAddressScriptHash(script, w.ChainParams())
	if err != nil {
		// above is a valid script, shouldn't happen.
		return nil, err
	}

	return btcjson.CreateMultiSigResult{
		Address:      address.EncodeAddress(),
		RedeemScript: hex.EncodeToString(script),
	}, nil
}

// dumpPrivKey handles a dumpprivkey request with the private key
// for a single address, or an appropiate error if the wallet
// is locked.
func dumpPrivKey(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.DumpPrivKeyCmd)

	addr, err := decodeAddress(cmd.Address, w.ChainParams())
	if err != nil {
		return nil, err
	}

	key, err := w.DumpWIFPrivateKey(addr)
	if waddrmgr.ErrLocked.Is(err) {
		// Address was found, but the private key isn't
		// accessible.
		return nil, btcjson.ErrRPCWalletUnlockNeeded.Default()
	}
	return key, err
}

func getAddressBalances(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.GetAddressBalancesCmd)
	szb := cmd.ShowZeroBalance != nil && *cmd.ShowZeroBalance
	if bals, err := w.CalculateAddressBalances(int32(*cmd.MinConf), szb); err != nil {
		return nil, err
	} else {
		results := make([]btcjson.GetAddressBalancesResult, 0, len(bals))
		for addr, bal := range bals {
			results = append(results, btcjson.GetAddressBalancesResult{
				Address: addr.EncodeAddress(),

				Spendable:  bal.Spendable.ToBTC(),
				Sspendable: strconv.FormatInt(int64(bal.Spendable), 10),

				Total:  bal.Total.ToBTC(),
				Stotal: strconv.FormatInt(int64(bal.Total), 10),

				ImmatureReward:  bal.ImmatureReward.ToBTC(),
				SimmatureReward: strconv.FormatInt(int64(bal.ImmatureReward), 10),

				Unconfirmed:  bal.Unconfirmed.ToBTC(),
				Sunconfirmed: strconv.FormatInt(int64(bal.Unconfirmed), 10),

				OutputCount: bal.OutputCount,
			})
		}
		return results, nil
	}
}

func getWalletSeed(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	if w.Manager.IsLocked() {
		return nil, btcjson.ErrRPCWalletUnlockNeeded.Default()
	}
	seed := w.Manager.Seed()
	if seed == nil {
		return nil, er.New("No seed found, this is probably a legacy wallet")
	}
	return seed.Words("english")
}

func getSecret(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.GetSecretCmd)
	return w.GetSecret(cmd.Name)
}

func walletMempool(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	if txs, err := w.WalletMempool(); err != nil {
		return nil, err
	} else {
		out := make([]btcjson.WalletMempoolItem, len(txs))
		for i, h := range txs {
			out[i] = btcjson.WalletMempoolItem{
				Txid:     h.Hash.String(),
				Received: h.Received.String(),
			}
		}
		return out, nil
	}
}

// getBalance handles a getbalance request by returning the balance for an
// account (wallet), or an error if the requested account does not
// exist.
func getBalance(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.GetBalanceCmd)
	if balance, err := w.CalculateBalance(int32(*cmd.MinConf)); err != nil {
		return nil, err
	} else {
		return balance.ToBTC(), nil
	}
}

// getBestBlock handles a getbestblock request by returning a JSON object
// with the height and hash of the most recently processed block.
func getBestBlock(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	blk := w.Manager.SyncedTo()
	result := &btcjson.GetBestBlockResult{
		Hash:   blk.Hash.String(),
		Height: blk.Height,
	}
	return result, nil
}

// getBestBlockHash handles a getbestblockhash request by returning the hash
// of the most recently processed block.
func getBestBlockHash(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	blk := w.Manager.SyncedTo()
	return blk.Hash.String(), nil
}

// getBlockCount handles a getblockcount request by returning the chain height
// of the most recently processed block.
func getBlockCount(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	blk := w.Manager.SyncedTo()
	return blk.Height, nil
}

// getInfo handles a getinfo request by returning the a structure containing
// information about the current state of pktwallet.
// exist.
func getInfo(icmd interface{}, w *wallet.Wallet, chainClient chain.Interface) (interface{}, er.R) {
	bs, err := chainClient.BlockStamp()
	if err != nil {
		return nil, err
	}

	walletStats := btcjson.WalletStats{}
	w.ReadStats(func(ws *btcjson.WalletStats) {
		walletStats = *ws
	})

	mgrStamp := w.Manager.SyncedTo()

	out := btcjson.WalletInfoResult{
		CurrentBlockHash:      mgrStamp.Hash.String(),
		CurrentHeight:         mgrStamp.Height,
		CurrentBlockTimestamp: mgrStamp.Timestamp,
		BackendBlockHash:      bs.Hash.String(),
		BackendHeight:         bs.Height,
		BackendBlockTimestamp: bs.Timestamp,
		IsSyncing:             !chainClient.IsCurrent(),
		Backend:               chainClient.BackEnd(),
		WalletVersion:         int32(waddrmgr.LatestMgrVersion),
		WalletStats:           &walletStats,
	}

	if rpc, ok := chainClient.(*chain.RPCClient); ok {
		// Call down to pktd for all of the information in this command known
		// by them.
		info, err := rpc.GetInfo()
		if err != nil {
			return nil, err
		}
		out.RPCInfo = info
	} else if neut, ok := chainClient.(*chain.NeutrinoClient); ok {
		ni := btcjson.NeutrinoInfo{}
		out.NeutrinoInfo = &ni
		for _, p := range neut.CS.Peers() {
			ni.Peers = append(ni.Peers, p.Describe())
		}
		if err := neut.CS.BanStore().ForEachBannedAddr(func(
			a *net.IPNet,
			r banman.Reason,
			t time.Time,
		) er.R {
			ni.Bans = append(ni.Bans, btcjson.NeutrinoBan{
				Addr:    a.String(),
				Reason:  r.String(),
				EndTime: t.String(),
			})
			return nil
		}); err != nil {
			return nil, err
		}
		for _, q := range neut.CS.GetActiveQueries() {
			peer := "<none>"
			if q.Peer != nil {
				peer = q.Peer.String()
			}
			ni.Queries = append(ni.Queries, btcjson.NeutrinoQuery{
				Peer:             peer,
				Command:          q.Command,
				ReqNum:           q.ReqNum,
				CreateTime:       q.CreateTime,
				LastRequestTime:  q.LastRequestTime,
				LastResponseTime: q.LastResponseTime,
			})
		}
	}

	return out, nil
}

func decodeAddress(s string, params *chaincfg.Params) (btcutil.Address, er.R) {
	addr, err := btcutil.DecodeAddress(s, params)
	if err != nil {
		msg := fmt.Sprintf("Invalid address %q: decode failed", s)
		return nil, btcjson.ErrRPCInvalidAddressOrKey.New(msg, err)
	}
	if !addr.IsForNet(params) {
		msg := fmt.Sprintf("Invalid address %q: not intended for use on %s",
			addr, params.Name)
		return nil, btcjson.ErrRPCInvalidAddressOrKey.New(msg, nil)
	}
	return addr, nil
}

func setNetworkStewardVote(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.SetNetworkStewardVoteCmd)
	vote := waddrmgr.NetworkStewardVote{}
	params := w.ChainParams()
	if cmd.VoteFor == nil {
	} else if vf, err := btcutil.DecodeAddress(*cmd.VoteFor, params); err != nil {
		return nil, err
	} else if vfs, err := txscript.PayToAddrScript(vf); err != nil {
		return nil, err
	} else {
		vote.VoteFor = vfs
	}
	if cmd.VoteAgainst == nil {
	} else if va, err := btcutil.DecodeAddress(*cmd.VoteAgainst, params); err != nil {
		return nil, err
	} else if vas, err := txscript.PayToAddrScript(va); err != nil {
		return nil, err
	} else {
		vote.VoteAgainst = vas
	}
	result := &btcjson.SetNetworkStewardVoteResult{}
	return result, w.PutNetworkStewardVote(waddrmgr.DefaultAccountNum, waddrmgr.KeyScopeBIP0044, &vote)
}

func getNetworkStewardVote(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	vote, err := w.NetworkStewardVote(waddrmgr.DefaultAccountNum, waddrmgr.KeyScopeBIP0044)
	if err != nil {
		return nil, err
	}
	result := &btcjson.GetNetworkStewardVoteResult{}
	if vote == nil {
		return result, nil
	}
	params := w.ChainParams()
	if vote.VoteFor != nil {
		result.VoteFor = txscript.PkScriptToAddress(vote.VoteFor, params).EncodeAddress()
	}
	if vote.VoteAgainst != nil {
		result.VoteAgainst = txscript.PkScriptToAddress(vote.VoteAgainst, params).EncodeAddress()
	}
	return result, nil
}

// getUnconfirmedBalance handles a getunconfirmedbalance extension request
// by returning the current unconfirmed balance of an account.
func getUnconfirmedBalance(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	bals, err := w.CalculateAddressBalances(0, false)
	if err != nil {
		return nil, err
	}
	sum := btcutil.Amount(0)
	for _, b := range bals {
		sum += b.Unconfirmed
	}
	return sum.ToBTC(), nil
}

// importPrivKey handles an importprivkey request by parsing
// a WIF-encoded private key and adding it to an account.
func importPrivKey(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.ImportPrivKeyCmd)

	// Ensure that private keys are only imported to the correct account.
	//
	// Yes, Label is the account name.
	if cmd.Label != nil && *cmd.Label != waddrmgr.ImportedAddrAccountName {
		return nil, errNotImportedAccount()
	}

	wif, err := btcutil.DecodeWIF(cmd.PrivKey)
	if err != nil {
		return nil, btcjson.ErrRPCInvalidAddressOrKey.New("WIF decode failed", err)
	}
	if !wif.IsForNet(w.ChainParams()) {
		return nil, btcjson.ErrRPCInvalidAddressOrKey.New(
			"Key is not intended for "+w.ChainParams().Name, nil)
	}

	// Import the private key, handling any errors.
	_, err = w.ImportPrivateKey(waddrmgr.KeyScopeBIP0044, wif, nil, *cmd.Rescan)
	switch {
	case waddrmgr.ErrDuplicateAddress.Is(err):
		// Do not return duplicate key errors to the client.
		return nil, nil
	case waddrmgr.ErrLocked.Is(err):
		return nil, btcjson.ErrRPCWalletUnlockNeeded.Default()
	}

	return nil, err
}

// getNewAddress handles a getnewaddress request by returning a new
// address for an account.  If the account does not exist an appropiate
// error is returned.
// TODO: Follow BIP 0044 and warn if number of unused addresses exceeds
// the gap limit.
func getNewAddress(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.GetNewAddressCmd)

	scope := waddrmgr.KeyScopeBIP0084
	if cmd.Legacy != nil && *cmd.Legacy {
		scope = waddrmgr.KeyScopeBIP0044
	}
	if addr, err := w.NewAddress(waddrmgr.DefaultAccountNum, scope); err != nil {
		return nil, err
	} else {
		return addr.EncodeAddress(), nil
	}
}

// getReceivedByAddress handles a getreceivedbyaddress request by returning
// the total amount received by a single address.
func getReceivedByAddress(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.GetReceivedByAddressCmd)

	addr, err := decodeAddress(cmd.Address, w.ChainParams())
	if err != nil {
		return nil, err
	}
	total, err := w.TotalReceivedForAddr(addr, int32(*cmd.MinConf))
	if err != nil {
		return nil, err
	}

	return total.ToBTC(), nil
}

// getTransaction handles a gettransaction request by returning details about
// a single transaction saved by wallet.
func getTransaction(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.GetTransactionCmd)

	txHash, err := chainhash.NewHashFromStr(cmd.Txid)
	if err != nil {
		return nil, btcjson.ErrRPCDecodeHexString.New(
			"Transaction hash string decode failed", err)
	}

	details, err := wallet.UnstableAPI(w).TxDetails(txHash)
	if err != nil {
		return nil, err
	}
	if details == nil {
		return nil, btcjson.ErrRPCNoTxInfo.Default()
	}

	syncBlock := w.Manager.SyncedTo()

	// TODO: The serialized transaction is already in the DB, so
	// reserializing can be avoided here.
	var txBuf bytes.Buffer
	txBuf.Grow(details.MsgTx.SerializeSize())
	err = details.MsgTx.Serialize(&txBuf)
	if err != nil {
		return nil, err
	}

	// TODO: Add a "generated" field to this result type.  "generated":true
	// is only added if the transaction is a coinbase.
	ret := btcjson.GetTransactionResult{
		TxID:            cmd.Txid,
		Hex:             hex.EncodeToString(txBuf.Bytes()),
		Time:            details.Received.Unix(),
		TimeReceived:    details.Received.Unix(),
		WalletConflicts: []string{}, // Not saved
		//Generated:     blockchain.IsCoinBaseTx(&details.MsgTx),
	}

	if details.Block.Height != -1 {
		ret.BlockHash = details.Block.Hash.String()
		ret.BlockTime = details.Block.Time.Unix()
		ret.Confirmations = int64(confirms(details.Block.Height, syncBlock.Height))
	}

	var (
		debitTotal  btcutil.Amount
		creditTotal btcutil.Amount // Excludes change
		fee         btcutil.Amount
		feeF64      float64
	)
	for _, deb := range details.Debits {
		debitTotal += deb.Amount
	}
	for _, cred := range details.Credits {
		if !cred.Change {
			creditTotal += cred.Amount
		}
	}
	// Fee can only be determined if every input is a debit.
	if len(details.Debits) == len(details.MsgTx.TxIn) {
		var outputTotal btcutil.Amount
		for _, output := range details.MsgTx.TxOut {
			outputTotal += btcutil.Amount(output.Value)
		}
		fee = debitTotal - outputTotal
		feeF64 = fee.ToBTC()
	}

	if len(details.Debits) == 0 {
		// Credits must be set later, but since we know the full length
		// of the details slice, allocate it with the correct cap.
		ret.Details = make([]btcjson.GetTransactionDetailsResult, 0, len(details.Credits))
	} else {
		ret.Details = make([]btcjson.GetTransactionDetailsResult, 1, len(details.Credits)+1)

		ret.Details[0] = btcjson.GetTransactionDetailsResult{
			// Fields left zeroed:
			//   InvolvesWatchOnly
			//   Account
			//   Address
			//   Vout
			//
			// TODO(jrick): Address and Vout should always be set,
			// but we're doing the wrong thing here by not matching
			// core.  Instead, gettransaction should only be adding
			// details for transaction outputs, just like
			// listtransactions (but using the short result format).
			Category: "send",
			Amount:   (-debitTotal).ToBTC(), // negative since it is a send
			Fee:      &feeF64,
		}
		ret.Fee = feeF64
	}

	credCat := wallet.RecvCategory(details, syncBlock.Height, w.ChainParams()).String()
	for _, cred := range details.Credits {
		// Change is ignored.
		if cred.Change {
			continue
		}

		var address string
		var accountName string
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(
			details.MsgTx.TxOut[cred.Index].PkScript, w.ChainParams())
		if err == nil && len(addrs) == 1 {
			addr := addrs[0]
			address = addr.EncodeAddress()
			account, err := w.AccountOfAddress(addr)
			if err == nil {
				name, err := w.AccountName(waddrmgr.KeyScopeBIP0044, account)
				if err == nil {
					accountName = name
				}
			}
		}

		ret.Details = append(ret.Details, btcjson.GetTransactionDetailsResult{
			// Fields left zeroed:
			//   InvolvesWatchOnly
			//   Fee
			Account:  accountName,
			Address:  address,
			Category: credCat,
			Amount:   cred.Amount.ToBTC(),
			Vout:     cred.Index,
		})
	}

	ret.Amount = creditTotal.ToBTC()
	return ret, nil
}

// These generators create the following global variables in this package:
//
//   var localeHelpDescs map[string]func() map[string]string
//   var requestUsages string
//
// localeHelpDescs maps from locale strings (e.g. "en_US") to a function that
// builds a map of help texts for each RPC server method.  This prevents help
// text maps for every locale map from being rooted and created during init.
// Instead, the appropiate function is looked up when help text is first needed
// using the current locale and saved to the global below for futher reuse.
//
// requestUsages contains single line usages for every supported request,
// separated by newlines.  It is set during init.  These usages are used for all
// locales.
//
//go:generate go run ../../internal/rpchelp/genrpcserverhelp.go legacyrpc
//go:generate gofmt -w rpcserverhelp.go

var helpDescs map[string]string
var helpDescsMu sync.Mutex // Help may execute concurrently, so synchronize access.

// helpWithChainRPC handles the help request when the RPC server has been
// associated with a consensus RPC client.  The additional RPC client is used to
// include help messages for methods implemented by the consensus server via RPC
// passthrough.
func helpWithChainRPC(icmd interface{}, w *wallet.Wallet, chainClient *chain.RPCClient) (interface{}, er.R) {
	return help(icmd, w, chainClient)
}

// helpNoChainRPC handles the help request when the RPC server has not been
// associated with a consensus RPC client.  No help messages are included for
// passthrough requests.
func helpNoChainRPC(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	return help(icmd, w, nil)
}

// help handles the help request by returning one line usage of all available
// methods, or full help for a specific method.  The chainClient is optional,
// and this is simply a helper function for the HelpNoChainRPC and
// HelpWithChainRPC handlers.
func help(icmd interface{}, w *wallet.Wallet, chainClient *chain.RPCClient) (interface{}, er.R) {
	cmd := icmd.(*btcjson.HelpCmd)

	// pktd returns different help messages depending on the kind of
	// connection the client is using.  Only methods availble to HTTP POST
	// clients are available to be used by wallet clients, even though
	// wallet itself is a websocket client to pktd.  Therefore, create a
	// POST client as needed.
	//
	// Returns nil if chainClient is currently nil or there is an error
	// creating the client.
	//
	// This is hacky and is probably better handled by exposing help usage
	// texts in a non-internal pktd package.
	postClient := func() *rpcclient.Client {
		if chainClient == nil {
			return nil
		}
		c, err := chainClient.POSTClient()
		if err != nil {
			return nil
		}
		return c
	}
	if cmd.Command == nil || *cmd.Command == "" {
		// Prepend chain server usage if it is available.
		usages := requestUsages
		client := postClient()
		if client != nil {
			rawChainUsage, err := client.RawRequest("help", nil)
			var chainUsage string
			if err == nil {
				_ = jsoniter.Unmarshal([]byte(rawChainUsage), &chainUsage)
			}
			if chainUsage != "" {
				usages = "Chain server usage:\n\n" + chainUsage + "\n\n" +
					"Wallet server usage (overrides chain requests):\n\n" +
					requestUsages
			}
		}
		return usages, nil
	}

	defer helpDescsMu.Unlock()
	helpDescsMu.Lock()

	if helpDescs == nil {
		// TODO: Allow other locales to be set via config or detemine
		// this from environment variables.  For now, hardcode US
		// English.
		helpDescs = localeHelpDescs["en_US"]()
	}

	helpText, ok := helpDescs[*cmd.Command]
	if ok {
		return helpText, nil
	}

	// Return the chain server's detailed help if possible.
	var chainHelp string
	client := postClient()
	if client != nil {
		param := make([]byte, len(*cmd.Command)+2)
		param[0] = '"'
		copy(param[1:], *cmd.Command)
		param[len(param)-1] = '"'
		rawChainHelp, err := client.RawRequest("help", []jsoniter.RawMessage{param})
		if err == nil {
			_ = jsoniter.Unmarshal([]byte(rawChainHelp), &chainHelp)
		}
	}
	if chainHelp != "" {
		return chainHelp, nil
	}
	return nil, btcjson.ErrRPCInvalidParameter.New(
		fmt.Sprintf("No help for method '%s'", *cmd.Command), nil)
}

// listLockUnspent handles a listlockunspent request by returning an slice of
// all locked outpoints.
func listLockUnspent(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	return w.LockedOutpoints(), nil
}

// listReceivedByAddress handles a listreceivedbyaddress request by returning
// a slice of objects, each one containing:
//  "account": the account of the receiving address;
//  "address": the receiving address;
//  "amount": total amount received by the address;
//  "confirmations": number of confirmations of the most recent transaction.
// It takes two parameters:
//  "minconf": minimum number of confirmations to consider a transaction -
//             default: one;
//  "includeempty": whether or not to include addresses that have no transactions -
//                  default: false.
func listReceivedByAddress(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.ListReceivedByAddressCmd)

	// Intermediate data for each address.
	type AddrData struct {
		// Total amount received.
		amount btcutil.Amount
		// Number of confirmations of the last transaction.
		confirmations int32
		// Hashes of transactions which include an output paying to the address
		tx []string
	}

	syncBlock := w.Manager.SyncedTo()

	// Intermediate data for all addresses.
	allAddrData := make(map[string]AddrData)
	// Create an AddrData entry for each active address in the account.
	// Otherwise we'll just get addresses from transactions later.
	sortedAddrs, err := w.SortedActivePaymentAddresses()
	if err != nil {
		return nil, err
	}
	for _, address := range sortedAddrs {
		// There might be duplicates, just overwrite them.
		allAddrData[address] = AddrData{}
	}

	minConf := *cmd.MinConf
	var endHeight int32
	if minConf == 0 {
		endHeight = -1
	} else {
		endHeight = syncBlock.Height - int32(minConf) + 1
	}
	err = wallet.UnstableAPI(w).RangeTransactions(0, endHeight, func(details []wtxmgr.TxDetails) (bool, er.R) {
		confirmations := confirms(details[0].Block.Height, syncBlock.Height)
		for _, tx := range details {
			for _, cred := range tx.Credits {
				pkScript := tx.MsgTx.TxOut[cred.Index].PkScript
				_, addrs, _, err := txscript.ExtractPkScriptAddrs(
					pkScript, w.ChainParams())
				if err != nil {
					// Non standard script, skip.
					continue
				}
				for _, addr := range addrs {
					addrStr := addr.EncodeAddress()
					addrData, ok := allAddrData[addrStr]
					if ok {
						addrData.amount += cred.Amount
						// Always overwrite confirmations with newer ones.
						addrData.confirmations = confirmations
					} else {
						addrData = AddrData{
							amount:        cred.Amount,
							confirmations: confirmations,
						}
					}
					addrData.tx = append(addrData.tx, tx.Hash.String())
					allAddrData[addrStr] = addrData
				}
			}
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	// Massage address data into output format.
	numAddresses := len(allAddrData)
	ret := make([]btcjson.ListReceivedByAddressResult, numAddresses)
	idx := 0
	for address, addrData := range allAddrData {
		ret[idx] = btcjson.ListReceivedByAddressResult{
			Address:       address,
			Amount:        addrData.amount.ToBTC(),
			Confirmations: uint64(addrData.confirmations),
			TxIDs:         addrData.tx,
		}
		idx++
	}
	return ret, nil
}

// listSinceBlock handles a listsinceblock request by returning an array of maps
// with details of sent and received wallet transactions since the given block.
func listSinceBlock(icmd interface{}, w *wallet.Wallet, chainClient chain.Interface) (interface{}, er.R) {
	cmd := icmd.(*btcjson.ListSinceBlockCmd)

	syncBlock := w.Manager.SyncedTo()
	targetConf := int64(*cmd.TargetConfirmations)

	var start int32
	if cmd.BlockHash != nil {
		hash, err := chainhash.NewHashFromStr(*cmd.BlockHash)
		if err != nil {
			return nil, btcjson.ErrRPCDeserialization.New("unable to parse block hash", err)
		}
		if block, err := chainClient.GetBlock(hash); err != nil {
			return nil, err
		} else if height, err := blockchain.ExtractBlockHeight(block); err != nil {
			return nil, err
		} else {
			start = height + 1
		}
	}

	txInfoList, err := w.ListSinceBlock(start, -1, syncBlock.Height)
	if err != nil {
		return nil, err
	}

	blockHash, err := chainClient.GetBlockHash(int64(syncBlock.Height) + 1 - targetConf)
	if err != nil {
		return nil, err
	}

	res := btcjson.ListSinceBlockResult{
		Transactions: txInfoList,
		LastBlock:    blockHash.String(),
	}
	return res, nil
}

// listTransactions handles a listtransactions request by returning an
// array of maps with details of sent and recevied wallet transactions.
func listTransactions(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.ListTransactionsCmd)
	return w.ListTransactions(*cmd.From, *cmd.Count)
}

// listAddressTransactions handles a listaddresstransactions request by
// returning an array of maps with details of spent and received wallet
// transactions.  The form of the reply is identical to listtransactions,
// but the array elements are limited to transaction details which are
// about the addresess included in the request.
func listAddressTransactions(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.ListAddressTransactionsCmd)

	if cmd.Account != nil && *cmd.Account != "*" {
		return nil, btcjson.ErrRPCInvalidParameter.New(
			"Listing transactions for addresses may only be done for all accounts", nil)

	}

	// Decode addresses.
	hash160Map := make(map[string]struct{})
	for _, addrStr := range cmd.Addresses {
		addr, err := decodeAddress(addrStr, w.ChainParams())
		if err != nil {
			return nil, err
		}
		hash160Map[string(addr.ScriptAddress())] = struct{}{}
	}

	return w.ListAddressTransactions(hash160Map)
}

// listAllTransactions handles a listalltransactions request by returning
// a map with details of sent and recevied wallet transactions.  This is
// similar to ListTransactions, except it takes only a single optional
// argument for the account name and replies with all transactions.
func listAllTransactions(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.ListAllTransactionsCmd)

	if cmd.Account != nil && *cmd.Account != "*" {
		return nil, btcjson.ErrRPCInvalidParameter.New(
			"Listing all transactions may only be done for all accounts", nil)
	}

	return w.ListAllTransactions()
}

// listUnspent handles the listunspent command.
func listUnspent(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.ListUnspentCmd)

	var addresses map[string]struct{}
	if cmd.Addresses != nil {
		addresses = make(map[string]struct{})
		// confirm that all of them are good:
		for _, as := range *cmd.Addresses {
			a, err := decodeAddress(as, w.ChainParams())
			if err != nil {
				return nil, err
			}
			addresses[a.EncodeAddress()] = struct{}{}
		}
	}

	return w.ListUnspent(int32(*cmd.MinConf), int32(*cmd.MaxConf), addresses)
}

// lockUnspent handles the lockunspent command.
func lockUnspent(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.LockUnspentCmd)
	lockName := "none"
	if cmd.LockName != nil {
		lockName = *cmd.LockName
	}

	switch {
	case cmd.Unlock && len(cmd.Transactions) == 0:
		w.ResetLockedOutpoints(cmd.LockName)
	default:
		for _, input := range cmd.Transactions {
			txHash, err := chainhash.NewHashFromStr(input.Txid)
			if err != nil {
				return nil, errParse("unable to parse hash", err)
			}
			op := wire.OutPoint{Hash: *txHash, Index: input.Vout}
			if cmd.Unlock {
				w.UnlockOutpoint(op)
			} else {
				w.LockOutpoint(op, lockName)
			}
		}
	}
	return true, nil
}

// makeOutputs creates a slice of transaction outputs from a pair of address
// strings to amounts.  This is used to create the outputs to include in newly
// created transactions from a JSON object describing the output destinations
// and amounts.
func makeOutputs(pairs map[string]btcutil.Amount, vote *waddrmgr.NetworkStewardVote,
	chainParams *chaincfg.Params) ([]*wire.TxOut, er.R) {
	outputs := make([]*wire.TxOut, 0, len(pairs))
	if vote == nil {
		vote = &waddrmgr.NetworkStewardVote{}
	}
	for addrStr, amt := range pairs {
		addr, err := btcutil.DecodeAddress(addrStr, chainParams)
		if err != nil {
			return nil, er.Errorf("cannot decode address: %s", err)
		}

		pkScript, err := txscript.PayToAddrScriptWithVote(addr, vote.VoteFor, vote.VoteAgainst)
		if err != nil {
			return nil, er.Errorf("cannot create txout script: %s", err)
		}

		outputs = append(outputs, wire.NewTxOut(int64(amt), pkScript))
	}
	return outputs, nil
}

func sendOutputs(
	w *wallet.Wallet,
	amounts map[string]btcutil.Amount,
	vote *waddrmgr.NetworkStewardVote,
	fromAddressses *[]string,
	minconf int32,
	feeSatPerKb btcutil.Amount,
	dryRun bool,
	changeAddress *string,
	inputMinHeight int,
	maxInputs int,
) (*txauthor.AuthoredTx, er.R) {
	req := wallet.CreateTxReq{
		Minconf:        minconf,
		FeeSatPerKB:    feeSatPerKb,
		DryRun:         dryRun,
		InputMinHeight: inputMinHeight,
		MaxInputs:      maxInputs,
	}
	if inputMinHeight > 0 {
		// TODO(cjd): Ideally we would expose the comparator choice to the
		// API consumer, but this is an API break. When we're using inputMinHeight
		// it's normally because we're trying to do multiple createtransaction
		// requests without double-spending, so it's important to prefer oldest
		// in this case.
		req.InputComparator = wallet.PreferOldest
	}
	var err er.R
	req.Outputs, err = makeOutputs(amounts, vote, w.ChainParams())
	if err != nil {
		return nil, err
	}
	if changeAddress != nil {
		addr, err := btcutil.DecodeAddress(*changeAddress, w.ChainParams())
		if err != nil {
			return nil, err
		}
		req.ChangeAddress = &addr
	}
	if fromAddressses != nil {
		addrs := make([]btcutil.Address, 0, len(*fromAddressses))
		for _, addrStr := range *fromAddressses {
			addr, err := btcutil.DecodeAddress(addrStr, w.ChainParams())
			if err != nil {
				return nil, err
			}
			addrs = append(addrs, addr)
		}
		req.InputAddresses = &addrs
	}
	tx, err := w.SendOutputs(req)
	if err != nil {
		if ruleerror.ErrNegativeTxOutValue.Is(err) {
			return nil, errNeedPositiveAmount()
		}
		if waddrmgr.ErrLocked.Is(err) {
			return nil, btcjson.ErrRPCWalletUnlockNeeded.Default()
		}
		if btcjson.Err.Is(err) {
			return nil, err
		}
		return nil, btcjson.ErrRPCInternal.New("SendOutputs failed", err)
	}
	return tx, nil
}

// sendPairs creates and sends payment transactions.
// It returns the transaction hash in string format upon success
// All errors are returned in btcjson.RPCError format
func sendPairs(w *wallet.Wallet, amounts map[string]btcutil.Amount,
	fromAddressses *[]string, minconf int32, feeSatPerKb btcutil.Amount, maxInputs, inputMinHeight int) (string, er.R) {

	vote, err := w.NetworkStewardVote(0, waddrmgr.KeyScopeBIP0044)
	if err != nil {
		return "", err
	}

	tx, err := sendOutputs(w, amounts, vote, fromAddressses, minconf, feeSatPerKb, false, nil, inputMinHeight, maxInputs)
	if err != nil {
		return "", err
	}

	txHashStr := tx.Tx.TxHash().String()
	log.Infof("Successfully sent transaction [%s]", pktlog.Txid(txHashStr))
	return txHashStr, nil
}

func isNilOrEmpty(s *string) bool {
	return s == nil || *s == ""
}

// sendFrom handles a sendfrom RPC request by creating a new transaction
// spending unspent transaction outputs for a wallet to another payment
// address.  Leftover inputs not sent to the payment address or a fee for
// the miner are sent back to a new address in the wallet.  Upon success,
// the TxID for the created transaction is returned.
func sendFrom(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.SendFromCmd)

	// Transaction comments are not yet supported.  Error instead of
	// pretending to save them.
	if !isNilOrEmpty(cmd.Comment) || !isNilOrEmpty(cmd.CommentTo) {
		return nil, errCommentsUnsupported()
	}

	// Check that signed integer parameters are positive.
	if cmd.Amount < 0 {
		return nil, errNeedPositiveAmount()
	}
	minConf := int32(*cmd.MinConf)
	if minConf < 0 {
		return nil, errNeedPositiveMinconf()
	}
	// Create map of address and amount pairs.
	amt, err := btcutil.NewAmount(cmd.Amount)
	if err != nil {
		return nil, err
	}
	pairs := map[string]btcutil.Amount{
		cmd.ToAddress: amt,
	}

	maxInputs := -1
	if cmd.MaxInputs != nil {
		maxInputs = *cmd.MaxInputs
	}

	minHeight := 0
	if cmd.MinHeight != nil {
		minHeight = *cmd.MinHeight
	}

	return sendPairs(w, pairs, cmd.FromAddresses, minConf, txrules.DefaultRelayFeePerKb, maxInputs, minHeight)
}

func createTransaction(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.CreateTransactionCmd)
	feeSatPerKb := txrules.DefaultRelayFeePerKb

	// Check that signed integer parameters are positive.
	if cmd.Amount < 0 {
		return nil, errNeedPositiveAmount()
	}
	minconf := int32(0)
	if cmd.MinConf != nil {
		minconf = int32(*cmd.MinConf)
		if minconf < 0 {
			return nil, errNeedPositiveMinconf()
		}
	}
	inputMinHeight := 0
	if cmd.InputMinHeight != nil && *cmd.InputMinHeight > 0 {
		inputMinHeight = *cmd.InputMinHeight
	}
	// Create map of address and amount pairs.
	amt, err := btcutil.NewAmount(cmd.Amount)
	if err != nil {
		return nil, err
	}
	amounts := map[string]btcutil.Amount{
		cmd.ToAddress: amt,
	}

	var vote *waddrmgr.NetworkStewardVote
	if cmd.Vote != nil && *cmd.Vote {
		vote, err = w.NetworkStewardVote(0, waddrmgr.KeyScopeBIP0044)
		if err != nil {
			return "", err
		}
	}

	maxInputs := -1
	if cmd.MaxInputs != nil {
		maxInputs = *cmd.MaxInputs
	}

	tx, err := sendOutputs(w, amounts, vote, cmd.FromAddresses, minconf,
		feeSatPerKb, true, cmd.ChangeAddress, inputMinHeight, maxInputs)
	if err != nil {
		return "", err
	}

	if cmd.AutoLock != nil {
		for _, in := range tx.Tx.TxIn {
			op := in.PreviousOutPoint
			w.LockOutpoint(op, *cmd.AutoLock)
		}
	}

	if cmd.ElectrumFormat != nil && *cmd.ElectrumFormat {
		b := new(bytes.Buffer)
		if err := tx.Tx.BtcEncode(b, 0, wire.ForceEptfEncoding); err != nil {
			return nil, err
		}
		return hex.EncodeToString(b.Bytes()), nil
	}

	b := bytes.NewBuffer(make([]byte, 0, tx.Tx.SerializeSize()))
	if err := tx.Tx.Serialize(b); err != nil {
		return nil, err
	}
	return hex.EncodeToString(b.Bytes()), nil
}

func stopResync(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	return w.StopResync()
}

func resync(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.ResyncCmd)
	fh := int32(-1)
	th := int32(-1)
	if cmd.FromHeight != nil {
		fh = *cmd.FromHeight
	}
	if cmd.ToHeight != nil {
		th = *cmd.ToHeight
	}
	var a []string
	if cmd.Addresses != nil {
		a = *cmd.Addresses
	}
	return nil, w.ResyncChain(fh, th, a, cmd.DropDb != nil && *cmd.DropDb)
}

// sendMany handles a sendmany RPC request by creating a new transaction
// spending unspent transaction outputs for a wallet to any number of
// payment addresses.  Leftover inputs not sent to the payment address
// or a fee for the miner are sent back to a new address in the wallet.
// Upon success, the TxID for the created transaction is returned.
func sendMany(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.SendManyCmd)

	// Transaction comments are not yet supported.  Error instead of
	// pretending to save them.
	if !isNilOrEmpty(cmd.Comment) {
		return nil, errCommentsUnsupported()
	}

	// Check that minconf is positive.
	minConf := int32(*cmd.MinConf)
	if minConf < 0 {
		return nil, errNeedPositiveMinconf()
	}

	// Recreate address/amount pairs, using dcrutil.Amount.
	pairs := make(map[string]btcutil.Amount, len(cmd.Amounts))
	for k, v := range cmd.Amounts {
		amt, err := btcutil.NewAmount(v)
		if err != nil {
			return nil, err
		}
		pairs[k] = amt
	}

	maxInputs := -1
	if cmd.MaxInputs != nil {
		maxInputs = *cmd.MaxInputs
	}

	return sendPairs(w, pairs, cmd.FromAddresses, minConf, txrules.DefaultRelayFeePerKb, maxInputs, 0)
}

// sendToAddress handles a sendtoaddress RPC request by creating a new
// transaction spending unspent transaction outputs for a wallet to another
// payment address.  Leftover inputs not sent to the payment address or a fee
// for the miner are sent back to a new address in the wallet.  Upon success,
// the TxID for the created transaction is returned.
func sendToAddress(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.SendToAddressCmd)

	// Transaction comments are not yet supported.  Error instead of
	// pretending to save them.
	if !isNilOrEmpty(cmd.Comment) || !isNilOrEmpty(cmd.CommentTo) {
		return nil, errCommentsUnsupported()
	}

	amt, err := btcutil.NewAmount(cmd.Amount)
	if err != nil {
		return nil, err
	}

	// Check that signed integer parameters are positive.
	if amt < 0 {
		return nil, errNeedPositiveAmount()
	}

	// Mock up map of address and amount pairs.
	pairs := map[string]btcutil.Amount{
		cmd.Address: amt,
	}

	// sendtoaddress always spends from the default account, this matches bitcoind
	return sendPairs(w, pairs, nil, 1, txrules.DefaultRelayFeePerKb, -1, 0)
}

// setTxFee sets the transaction fee per kilobyte added to transactions.
func setTxFee(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.SetTxFeeCmd)

	// Check that amount is not negative.
	if cmd.Amount < 0 {
		return nil, errNeedPositiveAmount()
	}

	// A boolean true result is returned upon success.
	return true, nil
}

// signMessage signs the given message with the private key for the given
// address
func signMessage(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.SignMessageCmd)

	addr, err := decodeAddress(cmd.Address, w.ChainParams())
	if err != nil {
		return nil, err
	}

	privKey, err := w.PrivKeyForAddress(addr)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	wire.WriteVarString(&buf, 0, "Bitcoin Signed Message:\n")
	wire.WriteVarString(&buf, 0, cmd.Message)
	messageHash := chainhash.DoubleHashB(buf.Bytes())
	sigbytes, err := btcec.SignCompact(btcec.S256(), privKey,
		messageHash, true)
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.EncodeToString(sigbytes), nil
}

// signRawTransaction handles the signrawtransaction command.
func signRawTransaction(icmd interface{}, w *wallet.Wallet, chainClient chain.Interface) (interface{}, er.R) {
	cmd := icmd.(*btcjson.SignRawTransactionCmd)

	serializedTx, err := decodeHexStr(cmd.RawTx)
	if err != nil {
		return nil, err
	}
	var tx wire.MsgTx
	err = tx.Deserialize(bytes.NewBuffer(serializedTx))
	if err != nil {
		return nil, errDeserialization("TX decode failed", err)
	}

	var hashType params.SigHashType
	switch *cmd.Flags {
	case "ALL":
		hashType = params.SigHashAll
	case "NONE":
		hashType = params.SigHashNone
	case "SINGLE":
		hashType = params.SigHashSingle
	case "ALL|ANYONECANPAY":
		hashType = params.SigHashAll | params.SigHashAnyOneCanPay
	case "NONE|ANYONECANPAY":
		hashType = params.SigHashNone | params.SigHashAnyOneCanPay
	case "SINGLE|ANYONECANPAY":
		hashType = params.SigHashSingle | params.SigHashAnyOneCanPay
	default:
		return nil, btcjson.ErrRPCInvalidParameter.New("Invalid sighash parameter", nil)
	}

	inputs := make(map[wire.OutPoint][]byte)
	scripts := make(map[string][]byte)

	for i, add := range tx.Additional {
		if len(add.PkScript) > 0 {
			inputs[tx.TxIn[i].PreviousOutPoint] = add.PkScript
		}
	}

	var cmdInputs []btcjson.RawTxInput
	if cmd.Inputs != nil {
		cmdInputs = *cmd.Inputs
	}
	for _, rti := range cmdInputs {
		inputHash, err := chainhash.NewHashFromStr(rti.Txid)
		if err != nil {
			return nil, errDeserialization("unable to parse txid", err)
		}

		script, err := decodeHexStr(rti.ScriptPubKey)
		if err != nil {
			return nil, err
		}

		// redeemScript is only actually used iff the user provided
		// private keys. In which case, it is used to get the scripts
		// for signing. If the user did not provide keys then we always
		// get scripts from the wallet.
		// Empty strings are ok for this one and hex.DecodeString will
		// DTRT.
		if cmd.PrivKeys != nil && len(*cmd.PrivKeys) != 0 {
			redeemScript, err := decodeHexStr(rti.RedeemScript)
			if err != nil {
				return nil, err
			}

			addr, err := btcutil.NewAddressScriptHash(redeemScript,
				w.ChainParams())
			if err != nil {
				return nil, errDeserialization("unable to parse redeemScript", err)
			}
			scripts[addr.String()] = redeemScript
		}
		inputs[wire.OutPoint{
			Hash:  *inputHash,
			Index: rti.Vout,
		}] = script
	}

	// Now we go and look for any inputs that we were not provided by
	// querying pktd with getrawtransaction. We queue up a bunch of async
	// requests and will wait for replies after we have checked the rest of
	// the arguments.
	requested := make(map[wire.OutPoint]rpcclient.FutureGetTxOutResult)
	for _, txIn := range tx.TxIn {
		// Did we get this outpoint from the arguments?
		if _, ok := inputs[txIn.PreviousOutPoint]; ok {
			continue
		}

		rpc, ok := chainClient.(*chain.RPCClient)
		if !ok {
			return nil, er.New("You must specify all transaction inputs explicitly, " +
				"or use --userpc to load them from pktd.")
		}

		// Asynchronously request the output script.
		requested[txIn.PreviousOutPoint] = rpc.GetTxOutAsync(
			&txIn.PreviousOutPoint.Hash, txIn.PreviousOutPoint.Index,
			true)
	}

	// Parse list of private keys, if present. If there are any keys here
	// they are the keys that we may use for signing. If empty we will
	// use any keys known to us already.
	var keys map[string]*btcutil.WIF
	if cmd.PrivKeys != nil {
		keys = make(map[string]*btcutil.WIF)

		for _, key := range *cmd.PrivKeys {
			wif, err := btcutil.DecodeWIF(key)
			if err != nil {
				return nil, errDeserialization("unable to parse wif", err)
			}

			if !wif.IsForNet(w.ChainParams()) {
				s := "key network doesn't match wallet's"
				return nil, errDeserialization(s, nil)
			}

			addr, err := btcutil.NewAddressPubKey(wif.SerializePubKey(),
				w.ChainParams())
			if err != nil {
				return nil, errDeserialization("unable to make pubkey", err)
			}
			keys[addr.EncodeAddress()] = wif
		}
	}

	// We have checked the rest of the args. now we can collect the async
	// txs. TODO: If we don't mind the possibility of wasting work we could
	// move waiting to the following loop and be slightly more asynchronous.
	for outPoint, resp := range requested {
		result, err := resp.Receive()
		if err != nil {
			return nil, err
		}
		if a, err := btcutil.DecodeAddress(result.Address, w.ChainParams()); err != nil {
			return nil, err
		} else {
			inputs[outPoint] = a.ScriptAddress()
		}
	}

	// All args collected. Now we can sign all the inputs that we can.
	// `complete' denotes that we successfully signed all outputs and that
	// all scripts will run to completion. This is returned as part of the
	// reply.
	signErrs, err := w.SignTransaction(&tx, hashType, inputs, keys, scripts)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.Grow(tx.SerializeSize())

	// All returned errors (not OOM, which panics) encounted during
	// bytes.Buffer writes are unexpected.
	if err = tx.Serialize(&buf); err != nil {
		panic(err)
	}

	signErrors := make([]btcjson.SignRawTransactionError, 0, len(signErrs))
	for _, e := range signErrs {
		input := tx.TxIn[e.InputIndex]
		signErrors = append(signErrors, btcjson.SignRawTransactionError{
			TxID:      input.PreviousOutPoint.Hash.String(),
			Vout:      input.PreviousOutPoint.Index,
			ScriptSig: hex.EncodeToString(input.SignatureScript),
			Sequence:  input.Sequence,
			Error:     e.Error.String(),
		})
	}

	return btcjson.SignRawTransactionResult{
		Hex:      hex.EncodeToString(buf.Bytes()),
		Complete: len(signErrors) == 0,
		Errors:   signErrors,
	}, nil
}

// validateAddress handles the validateaddress command.
func validateAddress(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.ValidateAddressCmd)

	result := btcjson.ValidateAddressWalletResult{}
	addr, err := decodeAddress(cmd.Address, w.ChainParams())
	if err != nil {
		// Use result zero value (IsValid=false).
		return result, nil
	}

	// We could put whether or not the address is a script here,
	// by checking the type of "addr", however, the reference
	// implementation only puts that information if the script is
	// "ismine", and we follow that behavior.
	result.Address = addr.EncodeAddress()
	result.IsValid = true

	ainfo, err := w.AddressInfo(addr)
	if err != nil {
		if waddrmgr.ErrAddressNotFound.Is(err) {
			// No additional information available about the address.
			return result, nil
		}
		return nil, err
	}

	// The address lookup was successful which means there is further
	// information about it available and it is "mine".
	result.IsMine = true
	acctName, err := w.AccountName(waddrmgr.KeyScopeBIP0044, ainfo.Account())
	if err != nil {
		return nil, errAccountNameNotFound()
	}
	result.Account = acctName

	switch ma := ainfo.(type) {
	case waddrmgr.ManagedPubKeyAddress:
		result.IsCompressed = ma.Compressed()
		result.PubKey = ma.ExportPubKey()

	case waddrmgr.ManagedScriptAddress:
		result.IsScript = true

		// The script is only available if the manager is unlocked, so
		// just break out now if there is an error.
		script, err := ma.Script()
		if err != nil {
			break
		}
		result.Hex = hex.EncodeToString(script)

		// This typically shouldn't fail unless an invalid script was
		// imported.  However, if it fails for any reason, there is no
		// further information available, so just set the script type
		// a non-standard and break out now.
		class, addrs, reqSigs, err := txscript.ExtractPkScriptAddrs(
			script, w.ChainParams())
		if err != nil {
			result.Script = txscript.NonStandardTy.String()
			break
		}

		addrStrings := make([]string, len(addrs))
		for i, a := range addrs {
			addrStrings[i] = a.EncodeAddress()
		}
		result.Addresses = addrStrings

		// Multi-signature scripts also provide the number of required
		// signatures.
		result.Script = class.String()
		if class == txscript.MultiSigTy {
			result.SigsRequired = int32(reqSigs)
		}
	}

	return result, nil
}

// verifyMessage handles the verifymessage command by verifying the provided
// compact signature for the given address and message.
func verifyMessage(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.VerifyMessageCmd)

	addr, err := decodeAddress(cmd.Address, w.ChainParams())
	if err != nil {
		return nil, err
	}

	// decode base64 signature
	sig, errr := base64.StdEncoding.DecodeString(cmd.Signature)
	if errr != nil {
		return nil, er.E(errr)
	}

	// Validate the signature - this just shows that it was valid at all.
	// we will compare it with the key next.
	var buf bytes.Buffer
	wire.WriteVarString(&buf, 0, "Bitcoin Signed Message:\n")
	wire.WriteVarString(&buf, 0, cmd.Message)
	expectedMessageHash := chainhash.DoubleHashB(buf.Bytes())
	pk, wasCompressed, err := btcec.RecoverCompact(btcec.S256(), sig,
		expectedMessageHash)
	if err != nil {
		return nil, err
	}

	var serializedPubKey []byte
	if wasCompressed {
		serializedPubKey = pk.SerializeCompressed()
	} else {
		serializedPubKey = pk.SerializeUncompressed()
	}
	// Verify that the signed-by address matches the given address
	switch checkAddr := addr.(type) {
	case *btcutil.AddressPubKeyHash: // ok
		return bytes.Equal(btcutil.Hash160(serializedPubKey), checkAddr.Hash160()[:]), nil
	case *btcutil.AddressPubKey: // ok
		return string(serializedPubKey) == checkAddr.String(), nil
	default:
		return nil, er.New("address type not supported")
	}
}

// walletIsLocked handles the walletislocked extension request by
// returning the current lock state (false for unlocked, true for locked)
// of an account.
func walletIsLocked(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	return w.Locked(), nil
}

// walletLock handles a walletlock request by locking the all account
// wallets, returning an error if any wallet is not encrypted (for example,
// a watching-only wallet).
func walletLock(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	w.Lock()
	return nil, nil
}

// walletPassphrase responds to the walletpassphrase request by unlocking
// the wallet.  The decryption key is saved in the wallet until timeout
// seconds expires, after which the wallet is locked.
func walletPassphrase(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.WalletPassphraseCmd)

	timeout := time.Second * time.Duration(cmd.Timeout)
	var unlockAfter <-chan time.Time
	if timeout != 0 {
		unlockAfter = time.After(timeout)
	}
	err := w.Unlock([]byte(cmd.Passphrase), unlockAfter)
	return nil, err
}

// walletPassphraseChange responds to the walletpassphrasechange request
// by unlocking all accounts with the provided old passphrase, and
// re-encrypting each private key with an AES key derived from the new
// passphrase.
//
// If the old passphrase is correct and the passphrase is changed, all
// wallets will be immediately locked.
func walletPassphraseChange(icmd interface{}, w *wallet.Wallet) (interface{}, er.R) {
	cmd := icmd.(*btcjson.WalletPassphraseChangeCmd)

	err := w.ChangePrivatePassphrase([]byte(cmd.OldPassphrase),
		[]byte(cmd.NewPassphrase))
	if waddrmgr.ErrWrongPassphrase.Is(err) {
		return nil, btcjson.ErrRPCWalletPassphraseIncorrect.Default()

	}
	return nil, err
}

// decodeHexStr decodes the hex encoding of a string, possibly prepending a
// leading '0' character if there is an odd number of bytes in the hex string.
// This is to prevent an error for an invalid hex string when using an odd
// number of bytes when calling hex.Decode.
func decodeHexStr(hexStr string) ([]byte, er.R) {
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, btcjson.ErrRPCDecodeHexString.New(
			"Hex string decode failed", er.E(err))
	}
	return decoded, nil
}

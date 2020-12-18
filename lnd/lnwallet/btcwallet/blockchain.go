package btcwallet

import (
	"strconv"

	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/wire"

	"github.com/pkt-cash/pktd/lnd/lnwallet"
	"github.com/pkt-cash/pktd/neutrino"
	"github.com/pkt-cash/pktd/pktwallet/chain"
	"github.com/pkt-cash/pktd/pktwallet/waddrmgr"
)

var (
	Err = er.NewErrorType("lnd.btcwallet")
	// ErrOutputSpent is returned by the GetUtxo method if the target output
	// for lookup has already been spent.
	ErrOutputSpent = Err.CodeWithDetail("ErrOutputSpent", "target output has been spent")

	// ErrOutputNotFound signals that the desired output could not be
	// located.
	ErrOutputNotFound = Err.CodeWithDetail("ErrOutputNotFound", "target output was not found")
)

// GetBestBlock returns the current height and hash of the best known block
// within the main chain.
//
// This method is a part of the lnwallet.BlockChainIO interface.
func (b *BtcWallet) GetBestBlock() (*chainhash.Hash, int32, er.R) {
	return b.chain.GetBestBlock()
}

// GetUtxo returns the original output referenced by the passed outpoint that
// creates the target pkScript.
//
// This method is a part of the lnwallet.BlockChainIO interface.
func (b *BtcWallet) GetUtxo(op *wire.OutPoint, pkScript []byte,
	heightHint uint32, cancel <-chan struct{}) (*wire.TxOut, er.R) {
	switch backend := b.chain.(type) {

	case *chain.NeutrinoClient:
		spendReport, err := backend.CS.GetUtxo(
			neutrino.WatchInputs(neutrino.InputWithScript{
				OutPoint: *op,
				PkScript: pkScript,
			}),
			neutrino.StartBlock(&waddrmgr.BlockStamp{
				Height: int32(heightHint),
			}),
			neutrino.QuitChan(cancel),
		)
		if err != nil {
			return nil, err
		}

		// If the spend report is nil, then the output was not found in
		// the rescan.
		if spendReport == nil {
			return nil, ErrOutputNotFound.Default()
		}

		// If the spending transaction is populated in the spend report,
		// this signals that the output has already been spent.
		if spendReport.SpendingTx != nil {
			return nil, ErrOutputSpent.Default()
		}

		// Otherwise, the output is assumed to be in the UTXO.
		return spendReport.Output, nil

	case *chain.RPCClient:
		txout, err := backend.GetTxOut(&op.Hash, op.Index, false)
		if err != nil {
			return nil, err
		} else if txout == nil {
			return nil, ErrOutputSpent.Default()
		}

		addr, err := btcutil.DecodeAddress(txout.Address, b.netParams)
		if err != nil {
			return nil, err
		}

		i, errr := strconv.ParseInt(txout.Svalue, 10, 64)
		if errr != nil {
			return nil, er.E(errr)
		}
		amt := btcutil.Amount(i)

		return &wire.TxOut{
			Value:    int64(amt),
			PkScript: addr.ScriptAddress(),
		}, nil

	default:
		return nil, er.Errorf("unknown backend")
	}
}

// GetBlock returns a raw block from the server given its hash.
//
// This method is a part of the lnwallet.BlockChainIO interface.
func (b *BtcWallet) GetBlock(blockHash *chainhash.Hash) (*wire.MsgBlock, er.R) {
	return b.chain.GetBlock(blockHash)
}

// GetBlockHash returns the hash of the block in the best blockchain at the
// given height.
//
// This method is a part of the lnwallet.BlockChainIO interface.
func (b *BtcWallet) GetBlockHash(blockHeight int64) (*chainhash.Hash, er.R) {
	return b.chain.GetBlockHash(blockHeight)
}

// A compile time check to ensure that BtcWallet implements the BlockChainIO
// interface.
var _ lnwallet.WalletController = (*BtcWallet)(nil)

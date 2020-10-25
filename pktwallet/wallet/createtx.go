// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/pktwallet/waddrmgr"
	"github.com/pkt-cash/pktd/pktwallet/wallet/txauthor"
	"github.com/pkt-cash/pktd/pktwallet/wallet/txrules"
	"github.com/pkt-cash/pktd/pktwallet/walletdb"
	"github.com/pkt-cash/pktd/pktwallet/wtxmgr"
	"github.com/pkt-cash/pktd/txscript"
	"github.com/pkt-cash/pktd/wire"
)

// Maximum number of inputs which will be included in a transaction
const MaxInputsPerTx = 1460

// Maximum number of inputs which can be included in a transaction if there is
// at least one legacy non-segwit input
const MaxInputsPerTxLegacy = 499

var InsufficientFundsError = er.GenericErrorType.CodeWithDetail("InsufficientFundsError",
	"insufficient funds available to construct transaction")

var TooManyInputsError = er.GenericErrorType.CodeWithDetail("TooManyInputsError",
	"unable to construct transaction because there are too many inputs, you may need to fold coins")

var UnconfirmedCoinsError = er.GenericErrorType.CodeWithDetail("UnconfirmedCoinsError",
	"unable to construct transaction, there are coins but they are not yet confirmed")

func makeInputSource(eligible []*wtxmgr.Credit) txauthor.InputSource {
	// Current inputs and their total value.  These are closed over by the
	// returned input source and reused across multiple calls.
	currentTotal := btcutil.Amount(0)
	currentInputs := make([]*wire.TxIn, 0, len(eligible))
	currentAdditonal := make([]wire.TxInAdditional, 0, len(eligible))

	return func(target btcutil.Amount) (btcutil.Amount, []*wire.TxIn, []wire.TxInAdditional, er.R) {
		for currentTotal < target && len(eligible) != 0 {
			nextCredit := eligible[0]
			eligible = eligible[1:]
			nextInput := wire.NewTxIn(&nextCredit.OutPoint, nil, nil)
			currentTotal += nextCredit.Amount
			currentInputs = append(currentInputs, nextInput)
			v := int64(nextCredit.Amount)
			currentAdditonal = append(currentAdditonal, wire.TxInAdditional{
				PkScript: nextCredit.PkScript,
				Value:    &v,
			})
		}
		return currentTotal, currentInputs, currentAdditonal, nil
	}
}

// secretSource is an implementation of txauthor.SecretSource for the wallet's
// address manager.
type secretSource struct {
	*waddrmgr.Manager
	addrmgrNs walletdb.ReadBucket
}

func (s secretSource) GetKey(addr btcutil.Address) (*btcec.PrivateKey, bool, er.R) {
	ma, err := s.Address(s.addrmgrNs, addr)
	if err != nil {
		return nil, false, err
	}

	mpka, ok := ma.(waddrmgr.ManagedPubKeyAddress)
	if !ok {
		e := er.Errorf("managed address type for %v is `%T` but "+
			"want waddrmgr.ManagedPubKeyAddress", addr, ma)
		return nil, false, e
	}
	privKey, err := mpka.PrivKey()
	if err != nil {
		return nil, false, err
	}
	return privKey, ma.Compressed(), nil
}

func (s secretSource) GetScript(addr btcutil.Address) ([]byte, er.R) {
	ma, err := s.Address(s.addrmgrNs, addr)
	if err != nil {
		return nil, err
	}

	msa, ok := ma.(waddrmgr.ManagedScriptAddress)
	if !ok {
		e := er.Errorf("managed address type for %v is `%T` but "+
			"want waddrmgr.ManagedScriptAddress", addr, ma)
		return nil, e
	}
	return msa.Script()
}

// txToOutputs creates a signed transaction which includes each output from
// outputs.  Previous outputs to reedeem are chosen from the passed account's
// UTXO set and minconf policy. An additional output may be added to return
// change to the wallet.  An appropriate fee is included based on the wallet's
// current relay fee.  The wallet must be unlocked to create the transaction.
//
// NOTE: The dryRun argument can be set true to create a tx that doesn't alter
// the database. A tx created with this set to true will intentionally have no
// input scripts added and SHOULD NOT be broadcasted.
func (w *Wallet) txToOutputs(txr CreateTxReq) (tx *txauthor.AuthoredTx, err er.R) {

	chainClient, err := w.requireChainClient()
	if err != nil {
		return nil, err
	}

	dbtx, err := w.db.BeginReadWriteTx()
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback()

	addrmgrNs := dbtx.ReadWriteBucket(waddrmgrNamespaceKey)

	// Get current block's height and hash.
	bs, err := chainClient.BlockStamp()
	if err != nil {
		return nil, err
	}

	var sweepOutput *wire.TxOut
	var needAmount btcutil.Amount
	for _, out := range txr.Outputs {
		needAmount += btcutil.Amount(out.Value)
		if out.Value == 0 {
			sweepOutput = out
		}
	}
	if sweepOutput != nil {
		needAmount = 0
	}
	eligibleOuts, err := w.findEligibleOutputs(
		dbtx, needAmount, txr.InputAddresses, txr.Minconf, bs,
		txr.InputMinHeight, txr.InputComparator, txr.MaxInputs)
	if err != nil {
		return nil, err
	}

	addrStr := "<all>"
	if txr.InputAddresses != nil {
		addrs := make([]string, 0, len(*txr.InputAddresses))
		for _, a := range *txr.InputAddresses {
			addrs = append(addrs, fmt.Sprintf("%s (%s)",
				a.EncodeAddress(), hex.EncodeToString(a.ScriptAddress())))
		}
		addrStr = strings.Join(addrs, ", ")
	}
	log.Debugf("Found [%d] eligible inputs from addresses [%s], excluded [%d] (unconfirmed) "+
		"and [%d] (too many inputs for tx)",
		len(eligibleOuts.credits), addrStr, eligibleOuts.unconfirmedCount, eligibleOuts.unusedCount)
	for _, eo := range eligibleOuts.credits {
		log.Debugf("  %s @ %d - %s", eo.OutPoint.String(), eo.Height, eo.Amount.String())
	}

	inputSource := makeInputSource(eligibleOuts.credits)
	changeSource := func() ([]byte, er.R) {
		// Derive the change output script.  As a hack to allow
		// spending from the imported account, change addresses are
		// created from account 0.
		var changeAddr btcutil.Address
		var err er.R
		if txr.ChangeAddress != nil {
			changeAddr = *txr.ChangeAddress
		} else {
			for _, c := range eligibleOuts.credits {
				_, addrs, _, _ := txscript.ExtractPkScriptAddrs(c.PkScript, w.chainParams)
				if len(addrs) == 1 {
					changeAddr = addrs[0]
				}
			}
			if changeAddr == nil {
				err = er.New("Unable to find qualifying change address")
			}
		}
		if err != nil {
			return nil, err
		}
		return txscript.PayToAddrScript(changeAddr)
	}
	tx, err = txauthor.NewUnsignedTransaction(txr.Outputs, txr.FeeSatPerKB, inputSource, changeSource)
	if err != nil {
		if !txauthor.ImpossibleTxError.Is(err) {
			return nil, err
		} else if eligibleOuts.unusedCount > 0 {
			return nil, TooManyInputsError.New(
				fmt.Sprintf("additional [%d] transactions containing [%f] coins",
					eligibleOuts.unusedCount, eligibleOuts.unusedAmt.ToBTC()), err)
		} else if eligibleOuts.unconfirmedCount > 0 {
			return nil, UnconfirmedCoinsError.New(
				fmt.Sprintf("there are [%f] coins available in [%d] unconfirmed transactions, "+
					"to spend from these you need to specify minconf=0",
					eligibleOuts.unconfirmedAmt.ToBTC(), eligibleOuts.unconfirmedCount), err)
		} else {
			if txr.InputAddresses != nil {
				return nil, InsufficientFundsError.New(
					fmt.Sprintf("address(es) [%s] do not have enough balance", addrStr), err)
			} else {
				return nil, InsufficientFundsError.New("wallet does not have enough balance", err)
			}
		}
	}

	// Randomize change position, if change exists, before signing.  This
	// doesn't affect the serialize size, so the change amount will still
	// be valid.
	if tx.ChangeIndex >= 0 {
		tx.RandomizeChangePosition()
	}

	// If a dry run was requested, we return now before adding the input
	// scripts, and don't commit the database transaction. The DB will be
	// rolled back when this method returns to ensure the dry run didn't
	// alter the DB in any way.
	if txr.DryRun {
		return tx, nil
	}

	err = tx.AddAllInputScripts(secretSource{w.Manager, addrmgrNs})
	if err != nil {
		return nil, err
	}

	err = validateMsgTx(tx.Tx)
	if err != nil {
		return nil, err
	}

	if err := dbtx.Commit(); err != nil {
		return nil, err
	}

	// Finally, we'll request the backend to notify us of the transaction
	// that pays to the change address, if there is one, when it confirms.
	if tx.ChangeIndex >= 0 {
		changePkScript := tx.Tx.TxOut[tx.ChangeIndex].PkScript
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(
			changePkScript, w.chainParams,
		)
		if err != nil {
			return nil, err
		}
		w.watch.WatchAddrs(addrs)
	}

	return tx, nil
}

func addrMatch(
	w *Wallet,
	script []byte,
	fromAddresses *[]btcutil.Address,
) (bool, txscript.ScriptClass) {
	sc, addrs, _, _ := txscript.ExtractPkScriptAddrs(script, w.chainParams)
	if fromAddresses != nil {
		for _, extractedAddr := range addrs {
			for _, addr := range *fromAddresses {
				if bytes.Equal(extractedAddr.ScriptAddress(), addr.ScriptAddress()) {
					return true, sc
				}
			}
		}
	}
	return false, sc
}

type amountCount struct {
	// Amount of coins
	amount btcutil.Amount

	isSegwit bool

	credits *redblacktree.Tree
}

func (a *amountCount) overLimit(maxInputs int) bool {
	count := a.credits.Size()
	if maxInputs > 0 {
		return count > maxInputs
	} else if count < MaxInputsPerTxLegacy {
	} else if a.isSegwit && count < MaxInputsPerTx {
	} else {
		return true
	}
	return false
}

// NilComparator compares by txid/index in order to make the red-black tree functions
func NilComparator(a, b interface{}) int {
	s1 := a.(*wtxmgr.Credit)
	s2 := b.(*wtxmgr.Credit)
	utils.Int64Comparator(int64(s1.Amount), int64(s2.Amount))
	txidCmp := bytes.Compare(s1.Hash[:], s2.Hash[:])
	if txidCmp != 0 {
		return txidCmp
	}
	return utils.UInt32Comparator(s1.Index, s2.Index)
}

// PreferOldest prefers oldest outputs first
func PreferOldest(a, b interface{}) int {
	s1 := a.(*wtxmgr.Credit)
	s2 := b.(*wtxmgr.Credit)
	if s1.Height < s2.Height {
		return -1
	} else if s1.Height > s2.Height {
		return 1
	} else {
		return NilComparator(s1, s2)
	}
}

// PreferNewest prefers newest outputs first
// func PreferNewest(a, b interface{}) int {
// 	return -PreferOldest(a, b)
// }

// PreferBiggest prefers biggest (coin value) outputs first
func PreferBiggest(a, b interface{}) int {
	s1 := a.(*wtxmgr.Credit)
	s2 := b.(*wtxmgr.Credit)
	if s1.Amount < s2.Amount {
		return 1
	} else if s1.Amount > s2.Amount {
		return -1
	} else {
		return NilComparator(s1, s2)
	}
}

// PreferSmallest prefers smallest (coin value) outputs first (spend the dust)
// func PreferSmallest(a, b interface{}) int {
// 	return -PreferBiggest(a, b)
// }

func convertResult(ac *amountCount) []*wtxmgr.Credit {
	ifaces := ac.credits.Keys()
	out := make([]*wtxmgr.Credit, len(ifaces))
	for i := range ifaces {
		out[i] = ifaces[i].(*wtxmgr.Credit)
	}
	return out
}

type eligibleOutputs struct {
	credits          []*wtxmgr.Credit
	unconfirmedCount int
	unconfirmedAmt   btcutil.Amount
	unusedCount      int
	unusedAmt        btcutil.Amount
}

func (w *Wallet) findEligibleOutputs(
	dbtx walletdb.ReadTx,
	needAmount btcutil.Amount,
	fromAddresses *[]btcutil.Address,
	minconf int32,
	bs *waddrmgr.BlockStamp,
	inputMinHeight int,
	inputComparator utils.Comparator,
	maxInputs int,
) (eligibleOutputs, er.R) {
	out := eligibleOutputs{}
	chainClient, err := w.requireChainClient()
	if err != nil {
		return out, err
	}
	txmgrNs := dbtx.ReadBucket(wtxmgrNamespaceKey)

	haveAmounts := make(map[string]*amountCount)
	var winner *amountCount

	if err := w.TxStore.ForEachUnspentOutput(txmgrNs, nil, func(_ []byte, output *wtxmgr.Credit) er.R {

		// Verify that the output is coming from one of the addresses which we accept to spend from
		// This is inherently expensive to filter at this level and ideally it would be moved into
		// the database by storing address->credit mappings directly, but after each transaction
		// is loaded, it's not much more effort to also extract the addresses each time.
		match, sc := addrMatch(w, output.PkScript, fromAddresses)
		if fromAddresses != nil && !match {
			return nil
		}

		if minconf > 0 {
			// Only include this output if it meets the required number of
			// confirmations.  Coinbase transactions must have have reached
			// maturity before their outputs may be spent.
			if !confirmed(minconf, output.Height, bs.Height) {
				out.unconfirmedCount++
				out.unconfirmedAmt += output.Amount
				return nil
			}
		}

		if output.Height >= 0 && output.Height < int32(inputMinHeight) {
			log.Debugf("Skipping output %s at height %d because it is below minimum %d",
				output.String(), output.Height, inputMinHeight)
			return nil
		}

		if output.FromCoinBase {
			if !confirmed(int32(w.chainParams.CoinbaseMaturity), output.Height, bs.Height) {
				return nil
			} else if txrules.IsBurned(output, w.chainParams, bs.Height+1440) {
				log.Debugf("Skipping burned output at height %d", output.Height)
				return nil
			}
		}

		// Locked unspent outputs are skipped.
		if w.LockedOutpoint(output.OutPoint) {
			return nil
		}

		// If there is an unspent which references a block header which doesn't
		// actually exist we've got some trouble. Lets make sure before we try to
		// spend it.
		if output.Height < 0 {
		} else if _, err := chainClient.GetBlockHeader(&output.Block.Hash); err != nil {
			log.Debugf("Input [%s] references block hash [%s] which is not in chain, skipping",
				output.OutPoint.String(), output.Block.Hash)
			return nil
		}

		str := hex.EncodeToString(output.PkScript)
		ha := haveAmounts[str]
		if ha == nil {
			haa := amountCount{}
			if inputComparator == nil {
				// If the user does not specify a comparator, we use the preferBiggest
				// comparator to prefer high value outputs over less valuable outputs.
				//
				// Without this, there would be a risk that the wallet collected a bunch
				// of dust and then - using arbitrary ordering - could not remove the dust
				// inputs to ever make the transaction small enough, despite having large
				// spendable outputs.
				//
				// This does NOT cause the default behavior of the wallet to prefer large
				// outputs over small, because with no explicit comparator, we short circuit
				// as soon as we have enough money to make the transaction.
				haa.credits = redblacktree.NewWith(PreferBiggest)
			} else {
				haa.credits = redblacktree.NewWith(inputComparator)
			}
			haa.isSegwit = sc.IsSegwit()
			ha = &haa
			haveAmounts[str] = ha
		}
		ha.credits.Put(output, nil)
		ha.amount += output.Amount
		if needAmount == 0 {
			// We're sweeping the wallet
		} else if ha.amount < needAmount {
			// We need more coins
		} else {
			worst := ha.credits.Right().Key.(*wtxmgr.Credit)
			if ha.amount-worst.Amount >= needAmount {
				// Our amount is still fine even if we drop the worst credit
				// so we'll drop it and continue traversing to find the best outputs
				ha.credits.Remove(worst)
				ha.amount -= worst.Amount
				out.unusedAmt += worst.Amount
				out.unusedCount++
			}

			// If we have no explicit sorting specified then we can short-circuit
			// and avoid table-scanning the whole db
			if inputComparator == nil {
				winner = ha
				return er.LoopBreak
			}
		}

		if !ha.overLimit(maxInputs) {
			// We don't have too many inputs
		} else if needAmount == 0 && inputComparator == nil {
			// We're sweeping the wallet with no ordering specified
			// This means we should just short-circuit with a winner
			winner = ha
			return er.LoopBreak
		} else {
			// Too many inputs, we will remove the worst
			worst := ha.credits.Right().Key.(*wtxmgr.Credit)
			ha.credits.Remove(worst)
			ha.amount -= worst.Amount
			out.unusedAmt += worst.Amount
			out.unusedCount++
		}
		return nil
	}); err != nil && !er.IsLoopBreak(err) {
		return out, err
	}

	if inputComparator != nil {
		// This is a special consideration because when there is a custom comparator,
		// we don't short circuit early so we might have a winner on our hands but not
		// know it.
		for _, ac := range haveAmounts {
			if ac.amount >= needAmount {
				winner = ac
			}
		}
	}

	if winner != nil {
		// Easy path, we got enough in one address to win, we'll just return those
		for _, ac := range haveAmounts {
			if ac != winner {
				out.unusedAmt += ac.amount
				out.unusedCount += ac.credits.Size()
			}
		}
		out.credits = convertResult(winner)
		return out, nil
	}

	// We don't have an easy answer with just one address, we need to get creative.
	// We will create a new tree using the preferBiggest in order to try to to get
	// a subset of inputs which fits inside of the required count
	outAc := amountCount{
		isSegwit: true,
		credits:  redblacktree.NewWith(PreferBiggest),
	}
	done := false
	for _, ac := range haveAmounts {
		if done {
			out.unusedAmt += ac.amount
			out.unusedCount += ac.credits.Size()
			continue
		}
		it := ac.credits.Iterator()
		for i := 0; it.Next(); i++ {
			outAc.credits.Put(it.Key(), nil)
		}
		outAc.isSegwit = outAc.isSegwit && ac.isSegwit

		wasOver := false
		for outAc.overLimit(maxInputs) {
			// Too many inputs, we will remove the worst
			worst := outAc.credits.Right().Key.(*wtxmgr.Credit)
			outAc.credits.Remove(worst)
			outAc.amount -= worst.Amount
			out.unusedAmt += worst.Amount
			out.unusedCount++
			wasOver = true
		}
		if needAmount == 0 && !wasOver {
			// if we were never over the limit and we're sweeping multiple addresses,
			// lets go around and get another address
		} else if outAc.amount > needAmount {
			// We have enough money to make the tx
			// We'll just iterate over the other entries to make unusedAmt and unusedCount correct
			done = true
		}
	}

	out.credits = convertResult(&outAc)
	return out, nil
}

// validateMsgTx verifies transaction input scripts for tx.  All previous output
// scripts from outputs redeemed by the transaction, in the same order they are
// spent, must be passed in the prevScripts slice.
func validateMsgTx(tx *wire.MsgTx) er.R {
	hashCache := txscript.NewTxSigHashes(tx)
	if len(tx.Additional) != len(tx.TxIn) {
		return er.Errorf("len(tx.Additional) = [%d] but len(tx.TxIn) = [%d], cannot validate tx",
			len(tx.Additional), len(tx.TxIn))
	}
	for i, add := range tx.Additional {
		if len(add.PkScript) == 0 {
			return er.Errorf("Unable to validate transaction, add.PkScript is empty")
		} else if add.Value == nil {
			return er.Errorf("Unable to validate transaction, add.Value is unknown")
		}
		vm, err := txscript.NewEngine(add.PkScript, tx, i,
			txscript.StandardVerifyFlags, nil, hashCache, *add.Value)
		if err != nil {
			err.AddMessage("cannot create script engine")
			return err
		}
		err = vm.Execute()
		if err != nil {
			err.AddMessage("cannot validate transaction")
			return err
		}
	}
	return nil
}

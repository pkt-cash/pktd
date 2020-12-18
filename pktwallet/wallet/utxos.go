// Copyright (c) 2016 The Decred developers
// Copyright (c) 2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/pktwallet/waddrmgr"
	"github.com/pkt-cash/pktd/txscript"
	"github.com/pkt-cash/pktd/wire"
)

// ErrNotMine is an error denoting that a Wallet instance is unable to
// spend a specified output.
var ErrNotMine = er.GenericErrorType.CodeWithDetail("ErrNotMine",
	"the passed output does not belong to the wallet")

// FetchInputInfo queries for the wallet's knowledge of the passed outpoint. If
// the wallet determines this output is under its control, then the original
// full transaction, the target txout and the number of confirmations are
// returned. Otherwise, a non-nil error value of ErrNotMine is returned instead.
func (w *Wallet) FetchInputInfo(prevOut *wire.OutPoint) (*wire.MsgTx,
	*wire.TxOut, int64, er.R) {
	// We manually look up the output within the tx store.
	txid := &prevOut.Hash
	txDetail, err := UnstableAPI(w).TxDetails(txid)
	if err != nil {
		return nil, nil, 0, err
	} else if txDetail == nil {
		return nil, nil, 0, ErrNotMine.Default()
	}

	// With the output retrieved, we'll make an additional check to ensure
	// we actually have control of this output. We do this because the check
	// above only guarantees that the transaction is somehow relevant to us,
	// like in the event of us being the sender of the transaction.
	numOutputs := uint32(len(txDetail.TxRecord.MsgTx.TxOut))
	if prevOut.Index >= numOutputs {
		return nil, nil, 0, er.Errorf("invalid output index %v for "+
			"transaction with %v outputs", prevOut.Index,
			numOutputs)
	}
	pkScript := txDetail.TxRecord.MsgTx.TxOut[prevOut.Index].PkScript
	if _, err := w.fetchOutputAddr(pkScript); err != nil {
		return nil, nil, 0, err
	}

	// Determine the number of confirmations the output currently has.
	_, currentHeight, err := w.chainClient.GetBestBlock()
	if err != nil {
		return nil, nil, 0, er.Errorf("unable to retrieve current "+
			"height: %v", err)
	}
	confs := int64(0)
	if txDetail.Block.Height != -1 {
		confs = int64(currentHeight - txDetail.Block.Height)
	}

	return &txDetail.TxRecord.MsgTx, &wire.TxOut{
		Value:    txDetail.TxRecord.MsgTx.TxOut[prevOut.Index].Value,
		PkScript: pkScript,
	}, confs, nil
}

// fetchOutputAddr attempts to fetch the managed address corresponding to the
// passed output script. This function is used to look up the proper key which
// should be used to sign a specified input.
func (w *Wallet) fetchOutputAddr(script []byte) (waddrmgr.ManagedAddress, er.R) {
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(script, w.chainParams)
	if err != nil {
		return nil, err
	}

	// If the case of a multi-sig output, several address may be extracted.
	// Therefore, we simply select the key for the first address we know
	// of.
	for _, addr := range addrs {
		addr, err := w.AddressInfo(addr)
		if err == nil {
			return addr, nil
		}
	}

	return nil, ErrNotMine.Default()
}

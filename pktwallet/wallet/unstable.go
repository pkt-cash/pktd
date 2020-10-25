// Copyright (c) 2016 The Decred developers
// Copyright (c) 2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/pktwallet/walletdb"
	"github.com/pkt-cash/pktd/pktwallet/wtxmgr"
)

type unstableAPI struct {
	w *Wallet
}

// UnstableAPI exposes additional unstable public APIs for a Wallet.  These APIs
// may be changed or removed at any time.  Currently this type exists to ease
// the transition (particularly for the legacy JSON-RPC server) from using
// exported manager packages to a unified wallet package that exposes all
// functionality by itself.  New code should not be written using this API.
func UnstableAPI(w *Wallet) unstableAPI { return unstableAPI{w} }

// TxDetails calls wtxmgr.Store.TxDetails under a single database view transaction.
func (u unstableAPI) TxDetails(txHash *chainhash.Hash) (*wtxmgr.TxDetails, er.R) {
	var details *wtxmgr.TxDetails
	err := walletdb.View(u.w.db, func(dbtx walletdb.ReadTx) er.R {
		txmgrNs := dbtx.ReadBucket(wtxmgrNamespaceKey)
		var err er.R
		details, err = u.w.TxStore.TxDetails(txmgrNs, txHash)
		return err
	})
	return details, err
}

// RangeTransactions calls wtxmgr.Store.RangeTransactions under a single
// database view tranasction.
func (u unstableAPI) RangeTransactions(begin, end int32, f func([]wtxmgr.TxDetails) (bool, er.R)) er.R {
	return walletdb.View(u.w.db, func(dbtx walletdb.ReadTx) er.R {
		txmgrNs := dbtx.ReadBucket(wtxmgrNamespaceKey)
		return u.w.TxStore.RangeTransactions(txmgrNs, begin, end, f)
	})
}

// Copyright (c) 2015-2020 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/pktlog/log"
	"github.com/pkt-cash/pktd/pktwallet/waddrmgr"
	"github.com/pkt-cash/pktd/pktwallet/walletdb"
	"github.com/pkt-cash/pktd/pktwallet/wtxmgr"
)

// bucketTxLabels is the name of the label sub bucket of the wtxmgr
// top level bucket that stores the mapping between a txid and a
// user-defined transaction label.
var bucketTxLabels = []byte("l")

// DropTransactionHistory completely removes and re-creates the transaction
// manager namespace from the given wallet database. This can be used to force
// a full chain rescan of all wallet transaction and UTXO data. User-defined
// transaction labels can optionally be kept by setting keepLabels to true.
func DropTransactionHistory(db walletdb.DB, keepLabels bool) er.R {
	log.Infof("Dropping btcwallet transaction history")

	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) er.R {
		// If we want to keep our tx labels, we read them out so we
		// can re-add them after we have deleted our wtxmgr.
		var (
			labels map[chainhash.Hash]string
			err    er.R
		)
		if keepLabels {
			labels, err = fetchAllLabels(tx)
			if err != nil {
				return err
			}
		}

		err = tx.DeleteTopLevelBucket(wtxmgrNamespaceKey)
		if err != nil && !walletdb.ErrBucketNotFound.Is(err) {
			return err
		}
		ns, err := tx.CreateTopLevelBucket(wtxmgrNamespaceKey)
		if err != nil {
			return err
		}
		err = wtxmgr.Create(ns)
		if err != nil {
			return err
		}

		// If we want to re-add our labels, we do so now.
		if keepLabels {
			if err := putTxLabels(ns, labels); err != nil {
				return err
			}
		}

		ns = tx.ReadWriteBucket(waddrmgrNamespaceKey)
		birthdayBlock, err := waddrmgr.FetchBirthdayBlock(ns)
		if err != nil {
			log.Warnf("Wallet does not have a birthday block " +
				"set, falling back to rescan from genesis")

			startBlock, err := waddrmgr.FetchStartBlock(ns)
			if err != nil {
				return err
			}
			return waddrmgr.PutSyncedTo(ns, startBlock)
		}

		// We'll need to remove our birthday block first because it
		// serves as a barrier when updating our state to detect reorgs
		// due to the wallet not storing all block hashes of the chain.
		if err := waddrmgr.DeleteBirthdayBlock(ns); err != nil {
			return err
		}

		if err := waddrmgr.PutSyncedTo(ns, &birthdayBlock); err != nil {
			return err
		}
		return waddrmgr.PutBirthdayBlock(ns, birthdayBlock)
	})
	if err != nil {
		return er.Errorf("failed to drop and re-create namespace: %v",
			err)
	}

	return nil
}

// fetchAllLabels returns a map of hex-encoded txid to label.
func fetchAllLabels(tx walletdb.ReadWriteTx) (map[chainhash.Hash]string,
	er.R) {
	// Get our top level bucket, if it does not exist we just exit.
	txBucket := tx.ReadBucket(wtxmgrNamespaceKey)
	if txBucket == nil {
		return nil, nil
	}

	// If we do not have a labels bucket, there are no labels so we exit.
	labelsBucket := txBucket.NestedReadBucket(bucketTxLabels)
	if labelsBucket == nil {
		return nil, nil
	}

	labels := make(map[chainhash.Hash]string)
	if err := labelsBucket.ForEach(func(k, v []byte) er.R {
		txid, err := chainhash.NewHash(k)
		if err != nil {
			return err
		}

		label, err := wtxmgr.DeserializeLabel(v)
		if err != nil {
			return err
		}

		// Add an entry to our map of labels.
		labels[*txid] = label

		return nil
	}); err != nil {
		return nil, err
	}

	return labels, nil
}

// putTxLabels re-adds a nested labels bucket and entries to the bucket provided
// if there are any labels present.
func putTxLabels(ns walletdb.ReadWriteBucket,
	labels map[chainhash.Hash]string) er.R {
	// If there are no labels, exit early.
	if len(labels) == 0 {
		return nil
	}

	// First, we create a labels bucket which we will add all labels to.
	labelBucket, err := ns.CreateBucketIfNotExists(bucketTxLabels)
	if err != nil {
		return err
	}

	// Next, we re-add every label to the bucket.
	for txid, label := range labels {
		err := wtxmgr.PutTxLabel(labelBucket, txid, label)
		if err != nil {
			return err
		}
	}

	return nil
}

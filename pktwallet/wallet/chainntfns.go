// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"fmt"
	"time"

	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/pktlog"

	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/pktwallet/chain"
	"github.com/pkt-cash/pktd/pktwallet/waddrmgr"
	"github.com/pkt-cash/pktd/pktwallet/wallet/watcher"
	"github.com/pkt-cash/pktd/pktwallet/walletdb"
	"github.com/pkt-cash/pktd/pktwallet/wtxmgr"
	"github.com/pkt-cash/pktd/txscript"
	"github.com/pkt-cash/pktd/wire"
)

const (
	// birthdayBlockDelta is the maximum time delta allowed between our
	// birthday timestamp and our birthday block's timestamp when searching
	// for a better birthday block candidate (if possible).
	birthdayBlockDelta = 2 * time.Hour
)

func (w *Wallet) handleChainNotifications() {
	defer w.wg.Done()

	chainClient, err := w.requireChainClient()
	if err != nil {
		log.Errorf("handleChainNotifications called without RPC client")
		return
	}

	for {
		select {
		case n, ok := <-chainClient.Notifications():
			if !ok {
				return
			}

			//log.Infof("Notification %v", reflect.TypeOf(n))
			var notificationName string
			var err er.R
			switch n := n.(type) {
			case chain.ClientConnected:
				// Before attempting to sync with our backend,
				// we'll make sure that our birthday block has
				// been set correctly to potentially prevent
				// missing relevant events.
				birthdayStore := &walletBirthdayStore{
					db:      w.db,
					manager: w.Manager,
				}
				birthdayBlock, err := birthdaySanityCheck(
					chainClient, birthdayStore,
				)
				if err != nil && !waddrmgr.ErrBirthdayBlockNotSet.Is(err) {
					err.AddMessage("Unable to sanity check wallet birthday block")
					panic(err.String())
				}

				err = w.syncWithChain(birthdayBlock)
				if err != nil && !w.ShuttingDown() {
					err.AddMessage("Unable to synchronize wallet to chain")
					panic(err.String())
				}
			case chain.BlockConnected:
				err = w.block(wtxmgr.BlockMeta(n).Block)
				notificationName = "block connected"
			case chain.BlockDisconnected:
				// Do nothing, connect block triggers the operations
				notificationName = "block disconnected"
			case chain.RelevantTx:
				// Neutrino does not notify us of transactions
				notificationName = "relevant transaction"
			case chain.FilteredBlockConnected:
				notificationName = "filtered block connected"

			// The following require some database maintenance, but also
			// need to be reported to the wallet's rescan goroutine.
			case *chain.RescanProgress:
				notificationName = "rescan progress"
			case *chain.RescanFinished:
				notificationName = "rescan finished"
			}
			if err != nil {
				err.AddMessage(fmt.Sprintf("Unable to process chain backend "+
					"%v notification", notificationName))
				log.Errorf(err.String())
			}
		case <-w.quit:
			return
		}
	}
}

func (w *Wallet) storeTxns(
	dbtx walletdb.ReadWriteTx,
	filterResp *chain.FilterBlocksResponse,
) er.R {
	for _, txn := range filterResp.RelevantTxns {
		if txRecord, err := wtxmgr.NewTxRecordFromMsgTx(txn, filterResp.BlockMeta.Time); err != nil {
			return err
		} else if err := w.addRelevantTx(dbtx, txRecord, &filterResp.BlockMeta); err != nil {
			return err
		}
	}
	return nil
}

// // Notify interested clients of the connected block.
// //
// // TODO: move all notifications outside of the database transaction.
// w.NtfnServer.notifyAttachedBlock(dbtx, &wtxmgr.BlockMeta{
// 	Block: wtxmgr.Block{
// 		Hash:   header.BlockHash(),
// 		Height: height,
// 	},
// 	Time: header.Timestamp,
// })
// return nil

func (w *Wallet) _rollbackBlock(dbtx walletdb.ReadWriteTx, bs waddrmgr.BlockStamp) er.R {
	log.Infof("Rollback of block [%v @ %v]", bs.Hash, bs.Height)
	addrmgrNs := dbtx.ReadWriteBucket(waddrmgrNamespaceKey)
	txmgrNs := dbtx.ReadWriteBucket(wtxmgrNamespaceKey)

	bsP := waddrmgr.BlockStamp{
		Height: bs.Height - 1,
	}
	hash, err := w.Manager.BlockHash(addrmgrNs, bsP.Height)
	if err != nil {
		return err
	}
	bsP.Hash = *hash

	// Careful, the preious block (bsP.Hash) might not exist in the chainClient
	// so we're doing the wrong thing and getting the blockStamp by height to get
	// the timestamp so we're sure to get an answer.
	// If this actually ever matters, it means the next block is about to be rolled
	// back as well.
	realBsP, err := getBlockStamp(w.chainClient, bsP.Height)
	if err != nil {
		return err
	}

	bsP.Timestamp = realBsP.Timestamp
	err = w.Manager.SetSyncedTo(addrmgrNs, &bsP)
	if err != nil {
		return err
	}

	err = w.TxStore.RollbackOne(txmgrNs, bs.Height)
	if err != nil {
		return err
	}

	// Notify interested clients of the disconnected block.
	w.NtfnServer.notifyDetachedBlock(&bs.Hash)

	return nil
}

func (w *Wallet) addRelevantTx(dbtx walletdb.ReadWriteTx, rec *wtxmgr.TxRecord, block *wtxmgr.BlockMeta) er.R {
	addrmgrNs := dbtx.ReadWriteBucket(waddrmgrNamespaceKey)
	txmgrNs := dbtx.ReadWriteBucket(wtxmgrNamespaceKey)

	// At the moment all notified transactions are assumed to actually be
	// relevant.  This assumption will not hold true when SPV support is
	// added, but until then, simply insert the transaction because there
	// should either be one or more relevant inputs or outputs.
	err := w.TxStore.InsertTx2(txmgrNs, rec, block)
	if err != nil {
		return err
	}

	// Check every output to determine whether it is controlled by a wallet
	// key.  If so, mark the output as a credit.
	for i, output := range rec.MsgTx.TxOut {
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(output.PkScript,
			w.chainParams)
		if err != nil {
			// Non-standard outputs are skipped.
			continue
		}
		for _, addr := range addrs {
			ma, err := w.Manager.Address(addrmgrNs, addr)
			if err == nil {
				isNew, err := w.TxStore.AddCredit2(txmgrNs, rec, block, uint32(i), ma.Internal())
				if err != nil {
					return err
				}
				err = w.Manager.MarkUsed(addrmgrNs, addr)
				if err != nil {
					return err
				}
				txOutAmt := btcutil.Amount(rec.MsgTx.TxOut[i].Value)
				if !isNew {
					// don't log when we see the same money again
				} else if block != nil {
					log.Infof("💸 "+pktlog.GreenBg("Got paid!")+" [%s] -> [%s] tx [%s] @ [%s]",
						pktlog.Coins(txOutAmt.ToPKT()),
						pktlog.Address(addr.String()),
						pktlog.Txid(rec.Hash.String()),
						pktlog.Height(block.Height))

					// Begin watching so we'll know when it's spent
					w.watch.WatchOutpoints([]watcher.OutPointWatch{
						{
							BeginHeight: block.Height,
							OutPoint:    wire.OutPoint{Hash: rec.Hash, Index: uint32(i)},
							Addr:        addr,
						},
					})
				} else {
					log.Infof("⏱  Detected unconfirmed payment [%s] -> [%s] tx [%s]",
						pktlog.Coins(txOutAmt.ToPKT()),
						pktlog.Address(addr.String()),
						pktlog.Txid(rec.Hash.String()))
				}
				continue
			}

			// Missing addresses are skipped.  Other errors should
			// be propagated.
			if !waddrmgr.ErrAddressNotFound.Is(err) {
				return err
			}
		}
	}

	// Send notification of mined or unmined transaction to any interested
	// clients.
	//
	// TODO: Avoid the extra db hits.
	if block == nil {
		details, err := w.TxStore.UniqueTxDetails(txmgrNs, &rec.Hash, nil)
		if err != nil {
			log.Errorf("Cannot query transaction details for notification: %v", err)
		}

		// It's possible that the transaction was not found within the
		// wallet's set of unconfirmed transactions due to it already
		// being confirmed, so we'll avoid notifying it.
		//
		// TODO(wilmer): ideally we should find the culprit to why we're
		// receiving an additional unconfirmed chain.RelevantTx
		// notification from the chain backend.
		if details != nil {
			w.NtfnServer.notifyUnminedTransaction(dbtx, details)
		}
	} else {
		details, err := w.TxStore.UniqueTxDetails(txmgrNs, &rec.Hash, &block.Block)
		if err != nil {
			log.Errorf("Cannot query transaction details for notification: %v", err)
		}

		// We'll only notify the transaction if it was found within the
		// wallet's set of confirmed transactions.
		if details != nil {
			w.NtfnServer.notifyMinedTransaction(dbtx, details, block)
		}
	}

	return nil
}

// chainConn is an interface that abstracts the chain connection logic required
// to perform a wallet's birthday block sanity check.
type chainConn interface {
	// GetBestBlock returns the hash and height of the best block known to
	// the backend.
	GetBestBlock() (*chainhash.Hash, int32, er.R)

	// GetBlockHash returns the hash of the block with the given height.
	GetBlockHash(int64) (*chainhash.Hash, er.R)

	// GetBlockHeader returns the header for the block with the given hash.
	GetBlockHeader(*chainhash.Hash) (*wire.BlockHeader, er.R)
}

// birthdayStore is an interface that abstracts the wallet's sync-related
// information required to perform a birthday block sanity check.
type birthdayStore interface {
	// Birthday returns the birthday timestamp of the wallet.
	Birthday() time.Time

	// BirthdayBlock returns the birthday block of the wallet. The boolean
	// returned should signal whether the wallet has already verified the
	// correctness of its birthday block.
	BirthdayBlock() (waddrmgr.BlockStamp, bool, er.R)

	// SetBirthdayBlock updates the birthday block of the wallet to the
	// given block. The boolean can be used to signal whether this block
	// should be sanity checked the next time the wallet starts.
	//
	// NOTE: This should also set the wallet's synced tip to reflect the new
	// birthday block. This will allow the wallet to rescan from this point
	// to detect any potentially missed events.
	SetBirthdayBlock(waddrmgr.BlockStamp) er.R
}

// walletBirthdayStore is a wrapper around the wallet's database and address
// manager that satisfies the birthdayStore interface.
type walletBirthdayStore struct {
	db      walletdb.DB
	manager *waddrmgr.Manager
}

var _ birthdayStore = (*walletBirthdayStore)(nil)

// Birthday returns the birthday timestamp of the wallet.
func (s *walletBirthdayStore) Birthday() time.Time {
	return s.manager.Birthday()
}

// BirthdayBlock returns the birthday block of the wallet.
func (s *walletBirthdayStore) BirthdayBlock() (waddrmgr.BlockStamp, bool, er.R) {
	var (
		birthdayBlock         waddrmgr.BlockStamp
		birthdayBlockVerified bool
	)

	err := walletdb.View(s.db, func(tx walletdb.ReadTx) er.R {
		var err er.R
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		birthdayBlock, birthdayBlockVerified, err = s.manager.BirthdayBlock(ns)
		return err
	})

	return birthdayBlock, birthdayBlockVerified, err
}

// SetBirthdayBlock updates the birthday block of the wallet to the
// given block. The boolean can be used to signal whether this block
// should be sanity checked the next time the wallet starts.
//
// NOTE: This should also set the wallet's synced tip to reflect the new
// birthday block. This will allow the wallet to rescan from this point
// to detect any potentially missed events.
func (s *walletBirthdayStore) SetBirthdayBlock(block waddrmgr.BlockStamp) er.R {
	return walletdb.Update(s.db, func(tx walletdb.ReadWriteTx) er.R {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		err := s.manager.SetBirthdayBlock(ns, block, true)
		if err != nil {
			return err
		}
		return s.manager.SetSyncedTo(ns, &block)
	})
}

// birthdaySanityCheck is a helper function that ensures a birthday block
// correctly reflects the birthday timestamp within a reasonable timestamp
// delta. It's intended to be run after the wallet establishes its connection
// with the backend, but before it begins syncing. This is done as the second
// part to the wallet's address manager migration where we populate the birthday
// block to ensure we do not miss any relevant events throughout rescans.
// waddrmgr.ErrBirthdayBlockNotSet is returned if the birthday block has not
// been set yet.
func birthdaySanityCheck(chainConn chainConn,
	birthdayStore birthdayStore) (*waddrmgr.BlockStamp, er.R) {

	// We'll start by fetching our wallet's birthday timestamp and block.
	birthdayTimestamp := birthdayStore.Birthday()
	birthdayBlock, birthdayBlockVerified, err := birthdayStore.BirthdayBlock()
	if err != nil {
		return nil, err
	}

	// If the birthday block has already been verified to be correct, we can
	// exit our sanity check to prevent potentially fetching a better
	// candidate.
	if birthdayBlockVerified {
		log.Debugf("Birthday block has already been verified: "+
			"height=%d, hash=%v", birthdayBlock.Height,
			birthdayBlock.Hash)

		return &birthdayBlock, nil
	}

	// Otherwise, we'll attempt to locate a better one now that we have
	// access to the chain.
	newBirthdayBlock, err := locateBirthdayBlock(chainConn, birthdayTimestamp)
	if err != nil {
		return nil, err
	}

	if err := birthdayStore.SetBirthdayBlock(*newBirthdayBlock); err != nil {
		return nil, err
	}

	return newBirthdayBlock, nil
}

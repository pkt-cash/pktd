// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"bytes"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/emirpasic/gods/utils"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/neutrino/headerfs"
	"github.com/pkt-cash/pktd/neutrino/pushtx"
	"github.com/pkt-cash/pktd/pktlog"
	"github.com/pkt-cash/pktd/txscript/params"

	"github.com/LK4D4/trylock"
	"github.com/pkt-cash/pktd/blockchain"
	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/btcjson"
	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/btcutil/hdkeychain"
	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/chaincfg/genesis"
	"github.com/pkt-cash/pktd/pktwallet/chain"
	"github.com/pkt-cash/pktd/pktwallet/waddrmgr"
	"github.com/pkt-cash/pktd/pktwallet/wallet/seedwords"
	"github.com/pkt-cash/pktd/pktwallet/wallet/txauthor"
	"github.com/pkt-cash/pktd/pktwallet/wallet/txrules"
	"github.com/pkt-cash/pktd/pktwallet/walletdb"
	"github.com/pkt-cash/pktd/pktwallet/walletdb/migration"
	"github.com/pkt-cash/pktd/pktwallet/wtxmgr"
	"github.com/pkt-cash/pktd/txscript"
	"github.com/pkt-cash/pktd/wire"
)

const (
	// InsecurePubPassphrase is the default outer encryption passphrase used
	// for public data (everything but private keys).  Using a non-default
	// public passphrase can prevent an attacker without the public
	// passphrase from discovering all past and future wallet addresses if
	// they gain access to the wallet database.
	//
	// NOTE: at time of writing, public encryption only applies to public
	// data in the waddrmgr namespace.  Transactions are not yet encrypted.
	InsecurePubPassphrase = "public"

	// recoveryBatchSize is the default number of blocks that will be
	// scanned successively by the recovery manager, in the event that the
	// wallet is started in recovery mode.
	recoveryBatchSize = 200
)

var (
	// ErrWalletShuttingDown is an error returned when we attempt to make a
	// request to the wallet but it is in the process of or has already shut
	// down.
	ErrWalletShuttingDown = Err.CodeWithDetail("ErrWalletShuttingDown",
		"wallet shutting down")

	ErrInProgress = Err.CodeWithDetail("ErrInProgress", "Already in progress")

	// Namespace bucket keys.
	waddrmgrNamespaceKey = []byte("waddrmgr")
	wtxmgrNamespaceKey   = []byte("wtxmgr")
)

func (w *Wallet) UpdateStats(f func(ws *btcjson.WalletStats)) {
	w.wsLock.Lock()
	defer w.wsLock.Unlock()
	f(&w.ws)
}

func (w *Wallet) ReadStats(f func(ws *btcjson.WalletStats)) {
	w.wsLock.RLock()
	defer w.wsLock.RUnlock()
	f(&w.ws)
}

// Wallet is a structure containing all the components for a
// complete wallet.  It contains the Armory-style key store
// addresses and keys),
type Wallet struct {
	publicPassphrase []byte

	// Data stores
	db      walletdb.DB
	Manager *waddrmgr.Manager
	TxStore *wtxmgr.Store

	chainClient        chain.Interface
	chainClientLock    sync.Mutex
	chainClientSynced  bool
	chainClientSyncMtx sync.Mutex

	lockedOutpoints map[wire.OutPoint]string

	recoveryWindow uint32

	// Channels for rescan processing.  Requests are added and merged with
	// any waiting requests, before being sent to another goroutine to
	// call the rescan RPC.
	rescanAddJob        chan *RescanJob
	rescanBatch         chan *rescanBatch
	rescanNotifications chan interface{} // From chain server
	rescanProgress      chan *RescanProgressMsg
	rescanFinished      chan *RescanFinishedMsg

	// Channel for transaction creation requests.
	createTxRequests chan createTxRequest

	// Channels for the manager locker.
	unlockRequests     chan unlockRequest
	lockRequests       chan struct{}
	holdUnlockRequests chan chan heldUnlock
	lockState          chan bool
	changePassphrase   chan changePassphraseRequest
	changePassphrases  chan changePassphrasesRequest

	NtfnServer *NotificationServer

	chainParams *chaincfg.Params
	wg          sync.WaitGroup

	started bool
	quit    chan struct{}
	quitMu  sync.Mutex

	rescanMempoolMu trylock.Mutex

	wsLock sync.RWMutex
	ws     btcjson.WalletStats
}

// Start starts the goroutines necessary to manage a wallet.
func (w *Wallet) Start() {
	w.quitMu.Lock()
	select {
	case <-w.quit:
		// Restart the wallet goroutines after shutdown finishes.
		w.WaitForShutdown()
		w.quit = make(chan struct{})
	default:
		// Ignore when the wallet is still running.
		if w.started {
			w.quitMu.Unlock()
			return
		}
		w.started = true
	}
	w.quitMu.Unlock()

	w.wg.Add(2)
	go w.txCreator()
	go w.walletLocker()
}

// SynchronizeRPC associates the wallet with the consensus RPC client,
// synchronizes the wallet with the latest changes to the blockchain, and
// continuously updates the wallet through RPC notifications.
//
// This method is unstable and will be removed when all syncing logic is moved
// outside of the wallet package.
func (w *Wallet) SynchronizeRPC(chainClient chain.Interface) {
	w.quitMu.Lock()
	select {
	case <-w.quit:
		w.quitMu.Unlock()
		return
	default:
	}
	w.quitMu.Unlock()

	// TODO: Ignoring the new client when one is already set breaks callers
	// who are replacing the client, perhaps after a disconnect.
	w.chainClientLock.Lock()
	if w.chainClient != nil {
		w.chainClientLock.Unlock()
		return
	}
	w.chainClient = chainClient

	// If the chain client is a NeutrinoClient instance, set a birthday so
	// we don't download all the filters as we go.
	switch cc := chainClient.(type) {
	case *chain.NeutrinoClient:
		cc.SetStartTime(w.Manager.Birthday())
	}
	w.chainClientLock.Unlock()

	// TODO: It would be preferable to either run these goroutines
	// separately from the wallet (use wallet mutator functions to
	// make changes from the RPC client) and not have to stop and
	// restart them each time the client disconnects and reconnets.
	w.wg.Add(4)
	go w.handleChainNotifications()
	go w.rescanBatchHandler()
	go w.rescanProgressHandler()
	go w.rescanRPCHandler()
}

// requireChainClient marks that a wallet method can only be completed when the
// consensus RPC server is set.  This function and all functions that call it
// are unstable and will need to be moved when the syncing code is moved out of
// the wallet.
func (w *Wallet) requireChainClient() (chain.Interface, er.R) {
	w.chainClientLock.Lock()
	chainClient := w.chainClient
	w.chainClientLock.Unlock()
	if chainClient == nil {
		return nil, er.New("blockchain RPC is inactive")
	}
	return chainClient, nil
}

// ChainClient returns the optional consensus RPC client associated with the
// wallet.
//
// This function is unstable and will be removed once sync logic is moved out of
// the wallet.
func (w *Wallet) ChainClient() chain.Interface {
	w.chainClientLock.Lock()
	chainClient := w.chainClient
	w.chainClientLock.Unlock()
	return chainClient
}

// quitChan atomically reads the quit channel.
func (w *Wallet) quitChan() <-chan struct{} {
	w.quitMu.Lock()
	c := w.quit
	w.quitMu.Unlock()
	return c
}

// Stop signals all wallet goroutines to shutdown.
func (w *Wallet) Stop() {
	w.quitMu.Lock()
	quit := w.quit
	w.quitMu.Unlock()

	select {
	case <-quit:
	default:
		close(quit)
		w.chainClientLock.Lock()
		if w.chainClient != nil {
			w.chainClient.Stop()
			w.chainClient = nil
		}
		w.chainClientLock.Unlock()
	}
}

func (w *Wallet) Db() walletdb.DB {
	return w.db
}

// ShuttingDown returns whether the wallet is currently in the process of
// shutting down or not.
func (w *Wallet) ShuttingDown() bool {
	select {
	case <-w.quitChan():
		return true
	default:
		return false
	}
}

// WaitForShutdown blocks until all wallet goroutines have finished executing.
func (w *Wallet) WaitForShutdown() {
	w.chainClientLock.Lock()
	if w.chainClient != nil {
		w.chainClient.WaitForShutdown()
	}
	w.chainClientLock.Unlock()
	w.wg.Wait()
}

// NetworkStewardVote gets the network steward which this account is voting for
func (w *Wallet) NetworkStewardVote(accountNumber uint32,
	scope waddrmgr.KeyScope) (vote *waddrmgr.NetworkStewardVote, err er.R) {
	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return
	}
	err = walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		var err er.R
		vote, err = manager.NetworkStewardVote(addrmgrNs, accountNumber)
		return err
	})
	return
}

// PutNetworkStewardVote gets the network steward which this account is voting for
func (w *Wallet) PutNetworkStewardVote(accountNumber uint32,
	scope waddrmgr.KeyScope, vote *waddrmgr.NetworkStewardVote) er.R {
	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return err
	}
	return walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) er.R {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return manager.PutNetworkStewardVote(addrmgrNs, accountNumber, vote)
	})
}

// ChainSynced returns whether the wallet has been attached to a chain server
// and synced up to the best block on the main chain.
func (w *Wallet) ChainSynced() bool {
	w.chainClientSyncMtx.Lock()
	synced := w.chainClientSynced
	w.chainClientSyncMtx.Unlock()
	return synced
}

// SetChainSynced marks whether the wallet is connected to and currently in sync
// with the latest block notified by the chain server.
//
// NOTE: Due to an API limitation with rpcclient, this may return true after
// the client disconnected (and is attempting a reconnect).  This will be unknown
// until the reconnect notification is received, at which point the wallet can be
// marked out of sync again until after the next rescan completes.
func (w *Wallet) SetChainSynced(synced bool) {
	w.chainClientSyncMtx.Lock()
	w.chainClientSynced = synced
	w.chainClientSyncMtx.Unlock()
}

// activeData returns the currently-active receiving addresses and all unspent
// outputs.  This is primarely intended to provide the parameters for a
// rescan request.
func (w *Wallet) activeData(dbtx walletdb.ReadTx) ([]btcutil.Address, []wtxmgr.Credit, er.R) {
	addrmgrNs := dbtx.ReadBucket(waddrmgrNamespaceKey)
	txmgrNs := dbtx.ReadBucket(wtxmgrNamespaceKey)

	var addrs []btcutil.Address
	err := w.Manager.ForEachActiveAddress(addrmgrNs, func(addr btcutil.Address) er.R {
		addrs = append(addrs, addr)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	unspent, err := w.TxStore.GetUnspentOutputs(txmgrNs)
	return addrs, unspent, err
}

// onClientConnected starts calls syncWithChain in a new go routine.
// At the moment there is no recourse if the rescan fails for
// some reason, however, the wallet will not be marked synced
// and many methods will error early since the wallet is known
// to be out of date.
func (w *Wallet) onClientConnected(birthdayStamp *waddrmgr.BlockStamp) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		err := w.syncWithChain(birthdayStamp)
		if err != nil && !w.ShuttingDown() {
			log.Warnf("Unable to synchronize wallet to chain: %v", err)
		}
	}()
}

// syncWithChain brings the wallet up to date with the current chain server
// connection. It creates a rescan request and blocks until the rescan has
// finished. The birthday block can be passed in, if set, to ensure we can
// properly detect if it gets rolled back.
func (w *Wallet) syncWithChain(birthdayStamp *waddrmgr.BlockStamp) er.R {
	chainClient, err := w.requireChainClient()
	if err != nil {
		return err
	}

	// We'll wait until the backend is synced to ensure we get the latest
	// MaxReorgDepth blocks to store. We don't do this for development
	// environments as we can't guarantee a lively chain.
	if !w.isDevEnv() {
		log.Debug("Waiting for chain backend to sync to tip")
		if err := w.waitUntilBackendSynced(chainClient); err != nil {
			return err
		}
		log.Info("Chain backend synced to tip! 👍")
	}

	// If we've yet to find our birthday block, we'll do so now.
	if birthdayStamp == nil {
		var err er.R
		birthdayStamp, err = locateBirthdayBlock(
			chainClient, w.Manager.Birthday(),
		)
		if err != nil {
			return er.Errorf("unable to locate birthday block: %v",
				err)
		}

		// We'll also determine our initial sync starting height. This
		// is needed as the wallet can now begin storing blocks from an
		// arbitrary height, rather than all the blocks from genesis, so
		// we persist this height to ensure we don't store any blocks
		// before it.
		startHeight := birthdayStamp.Height

		// With the starting height obtained, get the remaining block
		// details required by the wallet.
		startHash, err := chainClient.GetBlockHash(int64(startHeight))
		if err != nil {
			return err
		}
		startHeader, err := chainClient.GetBlockHeader(startHash)
		if err != nil {
			return err
		}

		err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) er.R {
			ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
			err := w.Manager.SetSyncedTo(ns, &waddrmgr.BlockStamp{
				Hash:      *startHash,
				Height:    startHeight,
				Timestamp: startHeader.Timestamp,
			})
			if err != nil {
				return err
			}
			return w.Manager.SetBirthdayBlock(ns, *birthdayStamp, true)
		})
		if err != nil {
			return er.Errorf("unable to persist initial sync "+
				"data: %v", err)
		}
	}

	w.UpdateStats(func(ws *btcjson.WalletStats) {
		ws.BirthdayBlock = birthdayStamp.Height
	})

	// If the wallet requested an on-chain recovery of its funds, we'll do
	// so now.
	if w.recoveryWindow > 0 {
		if err := w.recovery(chainClient, birthdayStamp); err != nil {
			return er.Errorf("unable to perform wallet recovery: "+
				"%v", err)
		}
	}

	// Compare previously-seen blocks against the current chain. If any of
	// these blocks no longer exist, rollback all of the missing blocks
	// before catching up with the rescan.
	rollback := false
	rollbackStamp := w.Manager.SyncedTo()
	err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) er.R {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		txmgrNs := tx.ReadWriteBucket(wtxmgrNamespaceKey)

		for height := rollbackStamp.Height; true; height-- {
			hash, err := w.Manager.BlockHash(addrmgrNs, height)
			if err != nil {
				return err
			}
			chainHash, err := chainClient.GetBlockHash(int64(height))
			if err != nil {
				return err
			}
			header, err := chainClient.GetBlockHeader(chainHash)
			if err != nil {
				return err
			}

			rollbackStamp.Hash = *chainHash
			rollbackStamp.Height = height
			rollbackStamp.Timestamp = header.Timestamp

			if bytes.Equal(hash[:], chainHash[:]) {
				break
			}
			rollback = true
		}

		// If a rollback did not happen, we can proceed safely.
		if !rollback {
			return nil
		}

		// Otherwise, we'll mark this as our new synced height.
		err := w.Manager.SetSyncedTo(addrmgrNs, &rollbackStamp)
		if err != nil {
			return err
		}

		// If the rollback happened to go beyond our birthday stamp,
		// we'll need to find a new one by syncing with the chain again
		// until finding one.
		if rollbackStamp.Height <= birthdayStamp.Height &&
			rollbackStamp.Hash != birthdayStamp.Hash {

			err := w.Manager.SetBirthdayBlock(
				addrmgrNs, rollbackStamp, true,
			)
			if err != nil {
				return err
			}
		}

		// Finally, we'll roll back our transaction store to reflect the
		// stale state. `Rollback` unconfirms transactions at and beyond
		// the passed height, so add one to the new synced-to height to
		// prevent unconfirming transactions in the synced-to block.
		return w.TxStore.Rollback(txmgrNs, rollbackStamp.Height+1)
	})
	if err != nil {
		return err
	}

	// Request notifications for connected and disconnected blocks.
	//
	// TODO(jrick): Either request this notification only once, or when
	// rpcclient is modified to allow some notification request to not
	// automatically resent on reconnect, include the notifyblocks request
	// as well.  I am leaning towards allowing off all rpcclient
	// notification re-registrations, in which case the code here should be
	// left as is.
	if err := chainClient.NotifyBlocks(); err != nil {
		return err
	}

	// Finally, we'll trigger a wallet rescan and request notifications for
	// transactions sending to all wallet addresses and spending all wallet
	// UTXOs.
	var (
		addrs   []btcutil.Address
		unspent []wtxmgr.Credit
	)
	err = walletdb.View(w.db, func(dbtx walletdb.ReadTx) er.R {
		addrs, unspent, err = w.activeData(dbtx)
		return err
	})
	if err != nil {
		return err
	}

	return w.rescanWithTarget(addrs, unspent, nil)
}

// isDevEnv determines whether the wallet is currently under a local developer
// environment, e.g. simnet or regtest.
func (w *Wallet) isDevEnv() bool {
	switch uint32(w.ChainParams().Net) {
	case uint32(chaincfg.RegressionNetParams.Net):
	case uint32(chaincfg.SimNetParams.Net):
	default:
		return false
	}
	return true
}

// waitUntilBackendSynced blocks until the chain backend considers itself
// "current".
func (w *Wallet) waitUntilBackendSynced(chainClient chain.Interface) er.R {
	// We'll poll every second to determine if our chain considers itself
	// "current".
	t := time.NewTicker(time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			if chainClient.IsCurrent() {
				return nil
			}
		case <-w.quitChan():
			return ErrWalletShuttingDown.Default()
		}
	}
}

// locateBirthdayBlock returns a block that meets the given birthday timestamp
// by a margin of +/-2 hours. This is safe to do as the timestamp is already 2
// days in the past of the actual timestamp.
func locateBirthdayBlock(chainClient chainConn,
	birthday time.Time) (*waddrmgr.BlockStamp, er.R) {

	// Retrieve the lookup range for our block.
	startHeight := int32(0)
	_, bestHeight, err := chainClient.GetBestBlock()
	if err != nil {
		return nil, err
	}

	log.Debugf("Locating suitable block for birthday %v between blocks "+
		"%v-%v", birthday, startHeight, bestHeight)

	var (
		birthdayBlock *waddrmgr.BlockStamp
		left, right   = startHeight, bestHeight
	)

	// Binary search for a block that meets the birthday timestamp by a
	// margin of +/-2 hours.
	for {
		// Retrieve the timestamp for the block halfway through our
		// range.
		mid := left + (right-left)/2
		hash, err := chainClient.GetBlockHash(int64(mid))
		if err != nil {
			return nil, err
		}
		header, err := chainClient.GetBlockHeader(hash)
		if err != nil {
			return nil, err
		}

		log.Debugf("Checking candidate block: height=%v, hash=%v, "+
			"timestamp=%v", mid, hash, header.Timestamp)

		// If the search happened to reach either of our range extremes,
		// then we'll just use that as there's nothing left to search.
		if mid == startHeight || mid == bestHeight || mid == left {
			birthdayBlock = &waddrmgr.BlockStamp{
				Hash:      *hash,
				Height:    int32(mid),
				Timestamp: header.Timestamp,
			}
			break
		}

		// The block's timestamp is more than 2 hours after the
		// birthday, so look for a lower block.
		if header.Timestamp.Sub(birthday) > birthdayBlockDelta {
			right = mid
			continue
		}

		// The birthday is more than 2 hours before the block's
		// timestamp, so look for a higher block.
		if header.Timestamp.Sub(birthday) < -birthdayBlockDelta {
			left = mid
			continue
		}

		birthdayBlock = &waddrmgr.BlockStamp{
			Hash:      *hash,
			Height:    int32(mid),
			Timestamp: header.Timestamp,
		}
		break
	}

	log.Debugf("Found birthday block: height=%d, hash=%v, timestamp=%v",
		birthdayBlock.Height, birthdayBlock.Hash,
		birthdayBlock.Timestamp)

	return birthdayBlock, nil
}

// recovery attempts to recover any unspent outputs that pay to any of our
// addresses starting from our birthday, or the wallet's tip (if higher), which
// would indicate resuming a recovery after a restart.
func (w *Wallet) recovery(chainClient chain.Interface, birthdayBlock *waddrmgr.BlockStamp) er.R {

	// Fetch the best height from the backend to determine when we should
	// stop.
	_, bestHeight, err := chainClient.GetBestBlock()
	if err != nil {
		return err
	}

	// We'll initialize the recovery manager with a default batch size of
	// 2000.
	recoveryMgr := NewRecoveryManager(
		w.recoveryWindow, recoveryBatchSize, w.chainParams,
	)

	// In the event that this recovery is being resumed, we will need to
	// repopulate all found addresses from the database. For basic recovery,
	// we will only do so for the default scopes.
	scopedMgrs, err := w.defaultScopeManagers()
	if err != nil {
		return err
	}
	err = walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		txMgrNS := tx.ReadBucket(wtxmgrNamespaceKey)
		credits, err := w.TxStore.GetUnspentOutputs(txMgrNS)
		if err != nil {
			return err
		}
		addrMgrNS := tx.ReadBucket(waddrmgrNamespaceKey)
		return recoveryMgr.Resurrect(addrMgrNS, scopedMgrs, credits)
	})
	if err != nil {
		return err
	}

	// Now we can begin scanning the chain from the wallet's current tip to
	// ensure we properly handle restarts. Since the recovery process itself
	// acts as rescan, we'll also update our wallet's synced state along the
	// way to reflect the blocks we process and prevent rescanning them
	// later on.
	//
	// NOTE: We purposefully don't update our best height since we assume
	// that a wallet rescan will be performed from the wallet's tip, which
	// will be of bestHeight after completing the recovery process.
	var blocks []*waddrmgr.BlockStamp
	startHeight := w.Manager.SyncedTo().Height + 1

	log.Debugf("Rescanning for used addresses from [%d] to [%d] ([%d] blocks)",
		startHeight, bestHeight, bestHeight-startHeight)

	startTime := time.Now()
	w.UpdateStats(func(ws *btcjson.WalletStats) {
		ws.Syncing = true
		ws.SyncStarted = &startTime
		ws.SyncCurrentBlock = startHeight
		ws.SyncFrom = startHeight
		ws.SyncTo = bestHeight
		ws.SyncRemainingSeconds = -1
	})
	defer w.UpdateStats(func(ws *btcjson.WalletStats) {
		ws.Syncing = false
		ws.SyncStarted = nil
		ws.SyncCurrentBlock = -1
		ws.SyncFrom = -1
		ws.SyncTo = -1
		ws.SyncRemainingSeconds = -1
	})

	for height := startHeight; height <= bestHeight; height++ {
		hash, err := chainClient.GetBlockHash(int64(height))
		if err != nil {
			return err
		}
		header, err := chainClient.GetBlockHeader(hash)
		if err != nil {
			return err
		}
		blocks = append(blocks, &waddrmgr.BlockStamp{
			Hash:      *hash,
			Height:    height,
			Timestamp: header.Timestamp,
		})

		// It's possible for us to run into blocks before our birthday
		// if our birthday is after our reorg safe height, so we'll make
		// sure to not add those to the batch.
		if height >= birthdayBlock.Height {
			recoveryMgr.AddToBlockBatch(
				hash, height, header.Timestamp,
			)
		}

		// We'll perform our recovery in batches of 2000 blocks.  It's
		// possible for us to reach our best height without exceeding
		// the recovery batch size, so we can proceed to commit our
		// state to disk.
		recoveryBatch := recoveryMgr.BlockBatch()
		if len(recoveryBatch) == recoveryBatchSize || height == bestHeight {
			err := walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) er.R {
				ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
				for _, block := range blocks {
					err := w.Manager.SetSyncedTo(ns, block)
					if err != nil {
						return err
					}
				}
				return w.recoverDefaultScopes(
					chainClient, tx, ns, recoveryBatch,
					recoveryMgr.State(),
				)
			})
			if err != nil {
				return err
			}

			timeToGo := time.Duration(0)
			w.UpdateStats(func(ws *btcjson.WalletStats) {
				if ws.SyncStarted != nil {
					startTime := *ws.SyncStarted
					timeSpent := time.Since(startTime)
					blocksSynced := height - ws.SyncFrom
					blocksToGo := ws.SyncTo - height
					timePerBlock := time.Duration(0)
					if blocksSynced > 0 {
						timePerBlock = timeSpent / time.Duration(blocksSynced)
					}
					timeToGo = timePerBlock * time.Duration(blocksToGo)
					ws.SyncRemainingSeconds = int64(timeToGo.Seconds())
				}
				ws.SyncCurrentBlock = height
			})

			if len(recoveryBatch) > 0 {
				log.Debugf("Recovered addresses from blocks [%d-%d] "+
					"Expected remaining time [%v]", recoveryBatch[0].Height,
					recoveryBatch[len(recoveryBatch)-1].Height, timeToGo)
			}

			// Clear the batch of all processed blocks to reuse the
			// same memory for future batches.
			blocks = blocks[:0]
			recoveryMgr.ResetBlockBatch()
		}
	}

	return nil
}

// defaultScopeManagers fetches the ScopedKeyManagers from the wallet using the
// default set of key scopes.
func (w *Wallet) defaultScopeManagers() (
	map[waddrmgr.KeyScope]*waddrmgr.ScopedKeyManager, er.R) {

	scopedMgrs := make(map[waddrmgr.KeyScope]*waddrmgr.ScopedKeyManager)
	for _, scope := range waddrmgr.DefaultKeyScopes {
		scopedMgr, err := w.Manager.FetchScopedKeyManager(scope)
		if err != nil {
			return nil, err
		}

		scopedMgrs[scope] = scopedMgr
	}

	return scopedMgrs, nil
}

// recoverDefaultScopes attempts to recover any addresses belonging to any
// active scoped key managers known to the wallet. Recovery of each scope's
// default account will be done iteratively against the same batch of blocks.
// TODO(conner): parallelize/pipeline/cache intermediate network requests
func (w *Wallet) recoverDefaultScopes(
	chainClient chain.Interface,
	tx walletdb.ReadWriteTx,
	ns walletdb.ReadWriteBucket,
	batch []wtxmgr.BlockMeta,
	recoveryState *RecoveryState) er.R {

	scopedMgrs, err := w.defaultScopeManagers()
	if err != nil {
		return err
	}

	return w.recoverScopedAddresses(
		chainClient, tx, ns, batch, recoveryState, scopedMgrs,
	)
}

// recoverAccountAddresses scans a range of blocks in attempts to recover any
// previously used addresses for a particular account derivation path. At a high
// level, the algorithm works as follows:
//  1) Ensure internal and external branch horizons are fully expanded.
//  2) Filter the entire range of blocks, stopping if a non-zero number of
//       address are contained in a particular block.
//  3) Record all internal and external addresses found in the block.
//  4) Record any outpoints found in the block that should be watched for spends
//  5) Trim the range of blocks up to and including the one reporting the addrs.
//  6) Repeat from (1) if there are still more blocks in the range.
func (w *Wallet) recoverScopedAddresses(
	chainClient chain.Interface,
	tx walletdb.ReadWriteTx,
	ns walletdb.ReadWriteBucket,
	batch []wtxmgr.BlockMeta,
	recoveryState *RecoveryState,
	scopedMgrs map[waddrmgr.KeyScope]*waddrmgr.ScopedKeyManager) er.R {

	// If there are no blocks in the batch, we are done.
	if len(batch) == 0 {
		return nil
	}

	log.Debugf("Scanning %d blocks for recoverable addresses", len(batch))

expandHorizons:
	for scope, scopedMgr := range scopedMgrs {
		scopeState := recoveryState.StateForScope(scope)
		err := expandScopeHorizons(ns, scopedMgr, scopeState)
		if err != nil {
			return err
		}
	}

	// With the internal and external horizons properly expanded, we now
	// construct the filter blocks request. The request includes the range
	// of blocks we intend to scan, in addition to the scope-index -> addr
	// map for all internal and external branches.
	filterReq := newFilterBlocksRequest(batch, scopedMgrs, recoveryState)

	w.Manager.ForEachAccountAddress(ns, waddrmgr.ImportedAddrAccount,
		func(maddr waddrmgr.ManagedAddress) er.R {
			filterReq.ImportedAddrs = append(filterReq.ImportedAddrs, maddr.Address())
			return nil
		})

	// Initiate the filter blocks request using our chain backend. If an
	// error occurs, we are unable to proceed with the recovery.
	filterResp, err := chainClient.FilterBlocks(filterReq)
	if err != nil {
		return err
	}

	// If the filter response is empty, this signals that the rest of the
	// batch was completed, and no other addresses were discovered. As a
	// result, no further modifications to our recovery state are required
	// and we can proceed to the next batch.
	if filterResp == nil {
		return nil
	}

	// Otherwise, retrieve the block info for the block that detected a
	// non-zero number of address matches.
	block := batch[filterResp.BatchIndex]

	// Log any non-trivial findings of addresses or outpoints.
	logFilterBlocksResp(block, filterResp)

	// Report any external or internal addresses found as a result of the
	// appropriate branch recovery state. Adding indexes above the
	// last-found index of either will result in the horizons being expanded
	// upon the next iteration. Any found addresses are also marked used
	// using the scoped key manager.
	err = extendFoundAddresses(ns, filterResp, scopedMgrs, recoveryState)
	if err != nil {
		return err
	}

	// Update the global set of watched outpoints with any that were found
	// in the block.
	for outPoint, addr := range filterResp.FoundOutPoints {
		recoveryState.AddWatchedOutPoint(&outPoint, addr)
	}

	// Finally, record all of the relevant transactions that were returned
	// in the filter blocks response. This ensures that these transactions
	// and their outputs are tracked when the final rescan is performed.
	for _, txn := range filterResp.RelevantTxns {
		txRecord, err := wtxmgr.NewTxRecordFromMsgTx(
			txn, filterResp.BlockMeta.Time,
		)
		if err != nil {
			return err
		}

		err = w.addRelevantTx(tx, txRecord, &filterResp.BlockMeta)
		if err != nil {
			return err
		}
	}

	// Update the batch to indicate that we've processed all block through
	// the one that returned found addresses.
	batch = batch[filterResp.BatchIndex+1:]

	// If this was not the last block in the batch, we will repeat the
	// filtering process again after expanding our horizons.
	if len(batch) > 0 {
		goto expandHorizons
	}

	return nil
}

// expandScopeHorizons ensures that the ScopeRecoveryState has an adequately
// sized look ahead for both its internal and external branches. The keys
// derived here are added to the scope's recovery state, but do not affect the
// persistent state of the wallet. If any invalid child keys are detected, the
// horizon will be properly extended such that our lookahead always includes the
// proper number of valid child keys.
func expandScopeHorizons(ns walletdb.ReadWriteBucket,
	scopedMgr *waddrmgr.ScopedKeyManager,
	scopeState *ScopeRecoveryState) er.R {

	// Compute the current external horizon and the number of addresses we
	// must derive to ensure we maintain a sufficient recovery window for
	// the external branch.
	exHorizon, exWindow := scopeState.ExternalBranch.ExtendHorizon()
	count, childIndex := uint32(0), exHorizon
	for count < exWindow {
		keyPath := externalKeyPath(childIndex)
		addr, err := scopedMgr.DeriveFromKeyPath(ns, keyPath)
		switch {
		case hdkeychain.ErrInvalidChild.Is(err):
			// Record the existence of an invalid child with the
			// external branch's recovery state. This also
			// increments the branch's horizon so that it accounts
			// for this skipped child index.
			scopeState.ExternalBranch.MarkInvalidChild(childIndex)
			childIndex++
			continue

		case err != nil:
			return err
		}

		// Register the newly generated external address and child index
		// with the external branch recovery state.
		scopeState.ExternalBranch.AddAddr(childIndex, addr.Address())

		childIndex++
		count++
	}

	// Compute the current internal horizon and the number of addresses we
	// must derive to ensure we maintain a sufficient recovery window for
	// the internal branch.
	inHorizon, inWindow := scopeState.InternalBranch.ExtendHorizon()
	count, childIndex = 0, inHorizon
	for count < inWindow {
		keyPath := internalKeyPath(childIndex)
		addr, err := scopedMgr.DeriveFromKeyPath(ns, keyPath)
		switch {
		case hdkeychain.ErrInvalidChild.Is(err):
			// Record the existence of an invalid child with the
			// internal branch's recovery state. This also
			// increments the branch's horizon so that it accounts
			// for this skipped child index.
			scopeState.InternalBranch.MarkInvalidChild(childIndex)
			childIndex++
			continue

		case err != nil:
			return err
		}

		// Register the newly generated internal address and child index
		// with the internal branch recovery state.
		scopeState.InternalBranch.AddAddr(childIndex, addr.Address())

		childIndex++
		count++
	}

	return nil
}

// externalKeyPath returns the relative external derivation path /0/0/index.
func externalKeyPath(index uint32) waddrmgr.DerivationPath {
	return waddrmgr.DerivationPath{
		Account: waddrmgr.DefaultAccountNum,
		Branch:  waddrmgr.ExternalBranch,
		Index:   index,
	}
}

// internalKeyPath returns the relative internal derivation path /0/1/index.
func internalKeyPath(index uint32) waddrmgr.DerivationPath {
	return waddrmgr.DerivationPath{
		Account: waddrmgr.DefaultAccountNum,
		Branch:  waddrmgr.InternalBranch,
		Index:   index,
	}
}

// newFilterBlocksRequest constructs FilterBlocksRequests using our current
// block range, scoped managers, and recovery state.
func newFilterBlocksRequest(batch []wtxmgr.BlockMeta,
	scopedMgrs map[waddrmgr.KeyScope]*waddrmgr.ScopedKeyManager,
	recoveryState *RecoveryState) *chain.FilterBlocksRequest {

	filterReq := &chain.FilterBlocksRequest{
		Blocks:           batch,
		ExternalAddrs:    make(map[waddrmgr.ScopedIndex]btcutil.Address),
		InternalAddrs:    make(map[waddrmgr.ScopedIndex]btcutil.Address),
		ImportedAddrs:    make([]btcutil.Address, 0),
		WatchedOutPoints: recoveryState.WatchedOutPoints(),
	}

	// Populate the external and internal addresses by merging the addresses
	// sets belong to all currently tracked scopes.
	for scope := range scopedMgrs {
		scopeState := recoveryState.StateForScope(scope)
		for index, addr := range scopeState.ExternalBranch.Addrs() {
			scopedIndex := waddrmgr.ScopedIndex{
				Scope: scope,
				Index: index,
			}
			filterReq.ExternalAddrs[scopedIndex] = addr
		}
		for index, addr := range scopeState.InternalBranch.Addrs() {
			scopedIndex := waddrmgr.ScopedIndex{
				Scope: scope,
				Index: index,
			}
			filterReq.InternalAddrs[scopedIndex] = addr
		}
	}

	return filterReq
}

// extendFoundAddresses accepts a filter blocks response that contains addresses
// found on chain, and advances the state of all relevant derivation paths to
// match the highest found child index for each branch.
func extendFoundAddresses(ns walletdb.ReadWriteBucket,
	filterResp *chain.FilterBlocksResponse,
	scopedMgrs map[waddrmgr.KeyScope]*waddrmgr.ScopedKeyManager,
	recoveryState *RecoveryState) er.R {

	// Mark all recovered external addresses as used. This will be done only
	// for scopes that reported a non-zero number of external addresses in
	// this block.
	for scope, indexes := range filterResp.FoundExternalAddrs {
		// First, report all external child indexes found for this
		// scope. This ensures that the external last-found index will
		// be updated to include the maximum child index seen thus far.
		scopeState := recoveryState.StateForScope(scope)
		for index := range indexes {
			scopeState.ExternalBranch.ReportFound(index)
		}

		scopedMgr := scopedMgrs[scope]

		// Now, with all found addresses reported, derive and extend all
		// external addresses up to and including the current last found
		// index for this scope.
		exNextUnfound := scopeState.ExternalBranch.NextUnfound()

		exLastFound := exNextUnfound
		if exLastFound > 0 {
			exLastFound--
		}

		err := scopedMgr.ExtendExternalAddresses(
			ns, waddrmgr.DefaultAccountNum, exLastFound,
		)
		if err != nil {
			return err
		}

		// Finally, with the scope's addresses extended, we mark used
		// the external addresses that were found in the block and
		// belong to this scope.
		for index := range indexes {
			addr := scopeState.ExternalBranch.GetAddr(index)
			err := scopedMgr.MarkUsed(ns, addr)
			if err != nil {
				return err
			}
		}
	}

	// Mark all recovered internal addresses as used. This will be done only
	// for scopes that reported a non-zero number of internal addresses in
	// this block.
	for scope, indexes := range filterResp.FoundInternalAddrs {
		// First, report all internal child indexes found for this
		// scope. This ensures that the internal last-found index will
		// be updated to include the maximum child index seen thus far.
		scopeState := recoveryState.StateForScope(scope)
		for index := range indexes {
			scopeState.InternalBranch.ReportFound(index)
		}

		scopedMgr := scopedMgrs[scope]

		// Now, with all found addresses reported, derive and extend all
		// internal addresses up to and including the current last found
		// index for this scope.
		inNextUnfound := scopeState.InternalBranch.NextUnfound()

		inLastFound := inNextUnfound
		if inLastFound > 0 {
			inLastFound--
		}
		err := scopedMgr.ExtendInternalAddresses(
			ns, waddrmgr.DefaultAccountNum, inLastFound,
		)
		if err != nil {
			return err
		}

		// Finally, with the scope's addresses extended, we mark used
		// the internal addresses that were found in the blockand belong
		// to this scope.
		for index := range indexes {
			addr := scopeState.InternalBranch.GetAddr(index)
			err := scopedMgr.MarkUsed(ns, addr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// logFilterBlocksResp provides useful logging information when filtering
// succeeded in finding relevant transactions.
func logFilterBlocksResp(block wtxmgr.BlockMeta,
	resp *chain.FilterBlocksResponse) {

	if log.Level() < pktlog.LevelDebug {
		// Nothing here runs unless debug level logging
		return
	}

	// Log the number of external addresses found in this block.
	var nFoundExternal int
	for _, indexes := range resp.FoundExternalAddrs {
		nFoundExternal += len(indexes)
	}
	if nFoundExternal > 0 {
		log.Debugf("Recovered %d external addrs at height=%d hash=%v",
			nFoundExternal, block.Height, block.Hash)
	}

	// Log the number of internal addresses found in this block.
	var nFoundInternal int
	for _, indexes := range resp.FoundInternalAddrs {
		nFoundInternal += len(indexes)
	}
	if nFoundInternal > 0 {
		log.Debugf("Recovered %d internal addrs at height=%d hash=%v",
			nFoundInternal, block.Height, block.Hash)
	}

	// Log the number of outpoints found in this block.
	nFoundOutPoints := len(resp.FoundOutPoints)
	if nFoundOutPoints > 0 {
		log.Debugf("Found %d spends from watched outpoints at "+
			"height=%d hash=%v",
			nFoundOutPoints, block.Height, block.Hash)
	}
}

type (
	CreateTxReq struct {
		InputAddresses  *[]btcutil.Address
		Outputs         []*wire.TxOut
		Minconf         int32
		FeeSatPerKB     btcutil.Amount
		DryRun          bool
		ChangeAddress   *btcutil.Address
		InputMinHeight  int
		InputComparator utils.Comparator
		MaxInputs       int
	}
	createTxRequest struct {
		req  CreateTxReq
		resp chan createTxResponse
	}
	createTxResponse struct {
		tx  *txauthor.AuthoredTx
		err er.R
	}
)

// txCreator is responsible for the input selection and creation of
// transactions.  These functions are the responsibility of this method
// (designed to be run as its own goroutine) since input selection must be
// serialized, or else it is possible to create double spends by choosing the
// same inputs for multiple transactions.  Along with input selection, this
// method is also responsible for the signing of transactions, since we don't
// want to end up in a situation where we run out of inputs as multiple
// transactions are being created.  In this situation, it would then be possible
// for both requests, rather than just one, to fail due to not enough available
// inputs.
func (w *Wallet) txCreator() {
	quit := w.quitChan()
out:
	for {
		select {
		case txr := <-w.createTxRequests:
			heldUnlock, err := w.holdUnlock()
			if err != nil {
				txr.resp <- createTxResponse{nil, err}
				continue
			}
			tx, err := w.txToOutputs(txr.req)
			heldUnlock.release()
			txr.resp <- createTxResponse{tx, err}
		case <-quit:
			break out
		}
	}
	w.wg.Done()
}

// CreateSimpleTx creates a new signed transaction spending unspent P2PKH
// outputs with at least minconf confirmations spending to any number of
// address/amount pairs.  Change and an appropriate transaction fee are
// automatically included, if necessary.  All transaction creation through this
// function is serialized to prevent the creation of many transactions which
// spend the same outputs.
//
// NOTE: The dryRun argument can be set true to create a tx that doesn't alter
// the database. A tx created with this set to true SHOULD NOT be broadcasted.
func (w *Wallet) CreateSimpleTx(r CreateTxReq) (*txauthor.AuthoredTx, er.R) {
	req := createTxRequest{
		req:  r,
		resp: make(chan createTxResponse),
	}
	w.createTxRequests <- req
	resp := <-req.resp
	return resp.tx, resp.err
}

type (
	unlockRequest struct {
		passphrase []byte
		lockAfter  <-chan time.Time // nil prevents the timeout.
		err        chan er.R
	}

	changePassphraseRequest struct {
		old, new []byte
		private  bool
		err      chan er.R
	}

	changePassphrasesRequest struct {
		publicOld, publicNew   []byte
		privateOld, privateNew []byte
		err                    chan er.R
	}

	// heldUnlock is a tool to prevent the wallet from automatically
	// locking after some timeout before an operation which needed
	// the unlocked wallet has finished.  Any aquired heldUnlock
	// *must* be released (preferably with a defer) or the wallet
	// will forever remain unlocked.
	heldUnlock chan struct{}
)

// walletLocker manages the locked/unlocked state of a wallet.
func (w *Wallet) walletLocker() {
	var timeout <-chan time.Time
	holdChan := make(heldUnlock)
	quit := w.quitChan()
out:
	for {
		select {
		case req := <-w.unlockRequests:
			err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
				addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
				return w.Manager.Unlock(addrmgrNs, req.passphrase)
			})
			if err != nil {
				req.err <- err
				continue
			}
			timeout = req.lockAfter
			if timeout == nil {
				log.Info("The wallet has been unlocked without a time limit")
			} else {
				log.Info("🔓 Wallet unlocked for use")
			}
			req.err <- nil
			continue

		case req := <-w.changePassphrase:
			err := walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) er.R {
				addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
				return w.Manager.ChangePassphrase(
					addrmgrNs, req.old, req.new, req.private,
					&waddrmgr.DefaultScryptOptions,
				)
			})
			req.err <- err
			continue

		case req := <-w.changePassphrases:
			err := walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) er.R {
				addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
				err := w.Manager.ChangePassphrase(
					addrmgrNs, req.publicOld, req.publicNew,
					false, &waddrmgr.DefaultScryptOptions,
				)
				if err != nil {
					return err
				}

				return w.Manager.ChangePassphrase(
					addrmgrNs, req.privateOld, req.privateNew,
					true, &waddrmgr.DefaultScryptOptions,
				)
			})
			req.err <- err
			continue

		case req := <-w.holdUnlockRequests:
			if w.Manager.IsLocked() {
				close(req)
				continue
			}

			req <- holdChan
			<-holdChan // Block until the lock is released.

			// If, after holding onto the unlocked wallet for some
			// time, the timeout has expired, lock it now instead
			// of hoping it gets unlocked next time the top level
			// select runs.
			select {
			case <-timeout:
				// Let the top level select fallthrough so the
				// wallet is locked.
			default:
				continue
			}

		case w.lockState <- w.Manager.IsLocked():
			continue

		case <-quit:
			break out

		case <-w.lockRequests:
		case <-timeout:
		}

		// Select statement fell through by an explicit lock or the
		// timer expiring.  Lock the manager here.
		timeout = nil
		err := w.Manager.Lock()
		if err != nil && !waddrmgr.ErrLocked.Is(err) {
			log.Errorf("Could not lock wallet: %v", err)
		} else {
			log.Info("The wallet has been locked")
		}
	}
	w.wg.Done()
}

// Unlock unlocks the wallet's address manager and relocks it after timeout has
// expired.  If the wallet is already unlocked and the new passphrase is
// correct, the current timeout is replaced with the new one.  The wallet will
// be locked if the passphrase is incorrect or any other error occurs during the
// unlock.
func (w *Wallet) Unlock(passphrase []byte, lock <-chan time.Time) er.R {
	err := make(chan er.R, 1)
	w.unlockRequests <- unlockRequest{
		passphrase: passphrase,
		lockAfter:  lock,
		err:        err,
	}
	return <-err
}

// Lock locks the wallet's address manager.
func (w *Wallet) Lock() {
	w.lockRequests <- struct{}{}
}

// Locked returns whether the account manager for a wallet is locked.
func (w *Wallet) Locked() bool {
	return <-w.lockState
}

// holdUnlock prevents the wallet from being locked.  The heldUnlock object
// *must* be released, or the wallet will forever remain unlocked.
//
// TODO: To prevent the above scenario, perhaps closures should be passed
// to the walletLocker goroutine and disallow callers from explicitly
// handling the locking mechanism.
func (w *Wallet) holdUnlock() (heldUnlock, er.R) {
	req := make(chan heldUnlock)
	w.holdUnlockRequests <- req
	hl, ok := <-req
	if !ok {
		// TODO(davec): This should be defined and exported from
		// waddrmgr.
		return nil, waddrmgr.ErrLocked.New("address manager is locked", nil)
	}
	return hl, nil
}

// release releases the hold on the unlocked-state of the wallet and allows the
// wallet to be locked again.  If a lock timeout has already expired, the
// wallet is locked again as soon as release is called.
func (c heldUnlock) release() {
	c <- struct{}{}
}

// ChangePrivatePassphrase attempts to change the passphrase for a wallet from
// old to new.  Changing the passphrase is synchronized with all other address
// manager locking and unlocking.  The lock state will be the same as it was
// before the password change.
func (w *Wallet) ChangePrivatePassphrase(old, new []byte) er.R {
	err := make(chan er.R, 1)
	w.changePassphrase <- changePassphraseRequest{
		old:     old,
		new:     new,
		private: true,
		err:     err,
	}
	return <-err
}

// AccountAddresses returns the addresses for every created address for an
// account.
func (w *Wallet) AccountAddresses(account uint32) (addrs []btcutil.Address, err er.R) {
	err = walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		return w.Manager.ForEachAccountAddress(addrmgrNs, account, func(maddr waddrmgr.ManagedAddress) er.R {
			addrs = append(addrs, maddr.Address())
			return nil
		})
	})
	return
}

func (w *Wallet) GetSecret(name string) (*string, er.R) {
	if w.Manager.IsLocked() {
		return nil, btcjson.ErrRPCWalletUnlockNeeded.Default()
	}
	// It's going to be easiest to use KeyScopeBIP0044 because that's what is
	// used by everything else to generate addresses, so if we need to migrate
	// to a different system, we'll be able to continue.
	manager, err := w.Manager.FetchScopedKeyManager(waddrmgr.KeyScopeBIP0044)
	if err != nil {
		return nil, err
	}
	var out *string
	err = walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		b, err := manager.GetSecret(addrmgrNs, waddrmgr.DefaultAccountNum, []byte(name))
		if err == nil {
			s := hex.EncodeToString(b)
			out = &s
		}
		return err
	})
	return out, err
}

// CalculateBalance sums the amounts of all unspent transaction
// outputs to addresses of a wallet and returns the balance.
//
// If confirmations is 0, all UTXOs, even those not present in a
// block (height -1), will be used to get the balance.  Otherwise,
// a UTXO must be in a block.  If confirmations is 1 or greater,
// the balance will be calculated based on how many how many blocks
// include a UTXO.
func (w *Wallet) CalculateBalance(confirms int32) (btcutil.Amount, er.R) {
	var balance btcutil.Amount
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)
		var err er.R
		blk := w.Manager.SyncedTo()
		balance, err = w.TxStore.Balance(txmgrNs, confirms, blk.Height)
		return err
	})
	return balance, err
}

// Balances records total, spendable (by policy), and immature coinbase
// reward balance amounts.
type Balances struct {
	Total          btcutil.Amount
	Spendable      btcutil.Amount
	ImmatureReward btcutil.Amount
	Unconfirmed    btcutil.Amount
	OutputCount    int32
}

func (w *Wallet) CalculateAddressBalances(
	confirms int32,
	showZeroBalances bool,
) (map[btcutil.Address]*Balances, er.R) {
	bals := make(map[btcutil.Address]*Balances)
	bals0 := make(map[string]*Balances)
	return bals, walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)
		// Get current block.  The block height used for calculating
		// the number of tx confirmations.
		syncBlock := w.Manager.SyncedTo()
		if showZeroBalances {
			addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
			if err := w.Manager.ForEachActiveAddress(addrmgrNs, func(addr btcutil.Address) er.R {
				_bal := Balances{}
				bal := &_bal
				bals0[string(addr.ScriptAddress())] = bal
				bals[addr] = bal
				return nil
			}); err != nil {
				return err
			}
		}
		return w.TxStore.ForEachUnspentOutput(txmgrNs, nil, func(_ []byte, output *wtxmgr.Credit) er.R {
			if _, addrs, _, err := txscript.ExtractPkScriptAddrs(output.PkScript, w.chainParams); err != nil {
				return err
			} else if len(addrs) > 0 {
				bal := bals0[string(addrs[0].ScriptAddress())]
				if bal == nil {
					_bal := Balances{}
					bal = &_bal
					bals0[string(addrs[0].ScriptAddress())] = bal
					bals[addrs[0]] = bal
				}
				bal.Total += output.Amount
				bal.OutputCount++
				if output.FromCoinBase && !confirmed(int32(w.chainParams.CoinbaseMaturity),
					output.Height, syncBlock.Height) {
					bal.ImmatureReward += output.Amount
				} else if confirmed(confirms, output.Height, syncBlock.Height) {
					bal.Spendable += output.Amount
				} else {
					bal.Unconfirmed += output.Amount
				}
			}
			return nil
		})
	})
}

// CalculateAccountBalances sums the amounts of all unspent transaction
// outputs to the given account of a wallet and returns the balance.
//
// This function is much slower than it needs to be since transactions outputs
// are not indexed by the accounts they credit to, and all unspent transaction
// outputs must be iterated.
func (w *Wallet) CalculateAccountBalances(account uint32, confirms int32) (Balances, er.R) {
	var bals Balances
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

		// Get current block.  The block height used for calculating
		// the number of tx confirmations.
		syncBlock := w.Manager.SyncedTo()

		return w.TxStore.ForEachUnspentOutput(txmgrNs, nil, func(_ []byte, output *wtxmgr.Credit) er.R {
			var outputAcct uint32
			_, addrs, _, err := txscript.ExtractPkScriptAddrs(
				output.PkScript, w.chainParams)
			if err == nil && len(addrs) > 0 {
				_, outputAcct, err = w.Manager.AddrAccount(addrmgrNs, addrs[0])
			}
			if err != nil || outputAcct != account {
				// disregard the error and keep searching
				return nil
			}

			bals.Total += output.Amount
			if output.FromCoinBase && !confirmed(int32(w.chainParams.CoinbaseMaturity),
				output.Height, syncBlock.Height) {
				bals.ImmatureReward += output.Amount
			} else if confirmed(confirms, output.Height, syncBlock.Height) {
				bals.Spendable += output.Amount
			}
			return nil
		})
	})
	return bals, err
}

// CurrentAddress gets the most recently requested Bitcoin payment address
// from a wallet for a particular key-chain scope.  If the address has already
// been used (there is at least one transaction spending to it in the
// blockchain or pktd mempool), the next chained address is returned.
func (w *Wallet) CurrentAddress(account uint32, scope waddrmgr.KeyScope) (btcutil.Address, er.R) {
	chainClient, err := w.requireChainClient()
	if err != nil {
		return nil, err
	}

	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return nil, err
	}

	var (
		addr  btcutil.Address
		props *waddrmgr.AccountProperties
	)
	err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) er.R {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		maddr, err := manager.LastExternalAddress(addrmgrNs, account)
		if err != nil {
			// If no address exists yet, create the first external
			// address.
			if waddrmgr.ErrAddressNotFound.Is(err) {
				addr, props, err = w.newAddress(
					addrmgrNs, account, scope,
				)
			}
			return err
		}

		// Get next chained address if the last one has already been
		// used.
		if maddr.Used(addrmgrNs) {
			addr, props, err = w.newAddress(
				addrmgrNs, account, scope,
			)
			return err
		}

		addr = maddr.Address()
		return nil
	})
	if err != nil {
		return nil, err
	}

	// If the props have been initially, then we had to create a new address
	// to satisfy the query. Notify the rpc server about the new address.
	if props != nil {
		err = chainClient.NotifyReceived([]btcutil.Address{addr})
		if err != nil {
			return nil, err
		}

		w.NtfnServer.notifyAccountProperties(props)
	}

	return addr, nil
}

// PubKeyForAddress looks up the associated public key for a P2PKH address.
func (w *Wallet) PubKeyForAddress(a btcutil.Address) (*btcec.PublicKey, er.R) {
	var pubKey *btcec.PublicKey
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		managedAddr, err := w.Manager.Address(addrmgrNs, a)
		if err != nil {
			return err
		}
		managedPubKeyAddr, ok := managedAddr.(waddrmgr.ManagedPubKeyAddress)
		if !ok {
			return er.New("address does not have an associated public key")
		}
		pubKey = managedPubKeyAddr.PubKey()
		return nil
	})
	return pubKey, err
}

// PrivKeyForAddress looks up the associated private key for a P2PKH or P2PK
// address.
func (w *Wallet) PrivKeyForAddress(a btcutil.Address) (*btcec.PrivateKey, er.R) {
	var privKey *btcec.PrivateKey
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		managedAddr, err := w.Manager.Address(addrmgrNs, a)
		if err != nil {
			return err
		}
		managedPubKeyAddr, ok := managedAddr.(waddrmgr.ManagedPubKeyAddress)
		if !ok {
			return er.New("address does not have an associated private key")
		}
		privKey, err = managedPubKeyAddr.PrivKey()
		return err
	})
	return privKey, err
}

// AccountOfAddress finds the account that an address is associated with.
func (w *Wallet) AccountOfAddress(a btcutil.Address) (uint32, er.R) {
	var account uint32
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		var err er.R
		_, account, err = w.Manager.AddrAccount(addrmgrNs, a)
		return err
	})
	return account, err
}

// AddressInfo returns detailed information regarding a wallet address.
func (w *Wallet) AddressInfo(a btcutil.Address) (waddrmgr.ManagedAddress, er.R) {
	var managedAddress waddrmgr.ManagedAddress
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		var err er.R
		managedAddress, err = w.Manager.Address(addrmgrNs, a)
		return err
	})
	return managedAddress, err
}

// AccountNumber returns the account number for an account name under a
// particular key scope.
func (w *Wallet) AccountNumber(scope waddrmgr.KeyScope, accountName string) (uint32, er.R) {
	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return 0, err
	}

	var account uint32
	err = walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		var err er.R
		account, err = manager.LookupAccount(addrmgrNs, accountName)
		return err
	})
	return account, err
}

// AccountName returns the name of an account.
func (w *Wallet) AccountName(scope waddrmgr.KeyScope, accountNumber uint32) (string, er.R) {
	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return "", err
	}

	var accountName string
	err = walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		var err er.R
		accountName, err = manager.AccountName(addrmgrNs, accountNumber)
		return err
	})
	return accountName, err
}

// CreditCategory describes the type of wallet transaction output.  The category
// of "sent transactions" (debits) is always "send", and is not expressed by
// this type.
//
// TODO: This is a requirement of the RPC server and should be moved.
type CreditCategory byte

// These constants define the possible credit categories.
const (
	CreditReceive CreditCategory = iota
	CreditGenerate
	CreditImmature
)

// String returns the category as a string.  This string may be used as the
// JSON string for categories as part of listtransactions and gettransaction
// RPC responses.
func (c CreditCategory) String() string {
	switch c {
	case CreditReceive:
		return "receive"
	case CreditGenerate:
		return "generate"
	case CreditImmature:
		return "immature"
	default:
		return "unknown"
	}
}

// RecvCategory returns the category of received credit outputs from a
// transaction record.  The passed block chain height is used to distinguish
// immature from mature coinbase outputs.
//
// TODO: This is intended for use by the RPC server and should be moved out of
// this package at a later time.
func RecvCategory(details *wtxmgr.TxDetails, syncHeight int32, net *chaincfg.Params) CreditCategory {
	if blockchain.IsCoinBaseTx(&details.MsgTx) {
		if confirmed(int32(net.CoinbaseMaturity), details.Block.Height,
			syncHeight) {
			return CreditGenerate
		}
		return CreditImmature
	}
	return CreditReceive
}

// listTransactions creates a object that may be marshalled to a response result
// for a listtransactions RPC.
//
// TODO: This should be moved to the legacyrpc package.
func listTransactions(tx walletdb.ReadTx, details *wtxmgr.TxDetails, addrMgr *waddrmgr.Manager,
	syncHeight int32, net *chaincfg.Params) []btcjson.ListTransactionsResult {

	addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)

	var (
		blockHashStr  string
		blockTime     int64
		confirmations int64
	)
	if details.Block.Height != -1 {
		blockHashStr = details.Block.Hash.String()
		blockTime = details.Block.Time.Unix()
		confirmations = int64(confirms(details.Block.Height, syncHeight))
	}

	results := []btcjson.ListTransactionsResult{}
	txHashStr := details.Hash.String()
	received := details.Received.Unix()
	generated := blockchain.IsCoinBaseTx(&details.MsgTx)
	recvCat := RecvCategory(details, syncHeight, net).String()

	send := len(details.Debits) != 0

	// Fee can only be determined if every input is a debit.
	var feeF64 float64
	if len(details.Debits) == len(details.MsgTx.TxIn) {
		var debitTotal btcutil.Amount
		for _, deb := range details.Debits {
			debitTotal += deb.Amount
		}
		var outputTotal btcutil.Amount
		for _, output := range details.MsgTx.TxOut {
			outputTotal += btcutil.Amount(output.Value)
		}
		// Note: The actual fee is debitTotal - outputTotal.  However,
		// this RPC reports negative numbers for fees, so the inverse
		// is calculated.
		feeF64 = (outputTotal - debitTotal).ToBTC()
	}

outputs:
	for i, output := range details.MsgTx.TxOut {
		// Determine if this output is a credit, and if so, determine
		// its spentness.
		var isCredit bool
		var spentCredit bool
		for _, cred := range details.Credits {
			if cred.Index == uint32(i) {
				// Change outputs are ignored.
				if cred.Change {
					continue outputs
				}

				isCredit = true
				spentCredit = cred.Spent
				break
			}
		}

		var address string
		var accountName string
		_, addrs, _, _ := txscript.ExtractPkScriptAddrs(output.PkScript, net)
		if len(addrs) == 1 {
			addr := addrs[0]
			address = addr.EncodeAddress()
			mgr, account, err := addrMgr.AddrAccount(addrmgrNs, addrs[0])
			if err == nil {
				accountName, err = mgr.AccountName(addrmgrNs, account)
				if err != nil {
					accountName = ""
				}
			}
		}

		amountF64 := btcutil.Amount(output.Value).ToBTC()
		result := btcjson.ListTransactionsResult{
			// Fields left zeroed:
			//   InvolvesWatchOnly
			//   BlockIndex
			//
			// Fields set below:
			//   Account (only for non-"send" categories)
			//   Category
			//   Amount
			//   Fee
			Address:         address,
			Vout:            uint32(i),
			Confirmations:   confirmations,
			Generated:       generated,
			BlockHash:       blockHashStr,
			BlockTime:       blockTime,
			TxID:            txHashStr,
			WalletConflicts: []string{},
			Time:            received,
			TimeReceived:    received,
		}

		// Add a received/generated/immature result if this is a credit.
		// If the output was spent, create a second result under the
		// send category with the inverse of the output amount.  It is
		// therefore possible that a single output may be included in
		// the results set zero, one, or two times.
		//
		// Since credits are not saved for outputs that are not
		// controlled by this wallet, all non-credits from transactions
		// with debits are grouped under the send category.

		if send || spentCredit {
			result.Category = "send"
			result.Amount = -amountF64
			result.Fee = &feeF64
			results = append(results, result)
		}
		if isCredit {
			result.Account = accountName
			result.Category = recvCat
			result.Amount = amountF64
			result.Fee = nil
			results = append(results, result)
		}
	}
	return results
}

// ListSinceBlock returns a slice of objects with details about transactions
// since the given block. If the block is -1 then all transactions are included.
// This is intended to be used for listsinceblock RPC replies.
func (w *Wallet) ListSinceBlock(start, end, syncHeight int32) ([]btcjson.ListTransactionsResult, er.R) {
	txList := []btcjson.ListTransactionsResult{}
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

		rangeFn := func(details []wtxmgr.TxDetails) (bool, er.R) {
			for _, detail := range details {
				jsonResults := listTransactions(tx, &detail,
					w.Manager, syncHeight, w.chainParams)
				txList = append(txList, jsonResults...)
			}
			return false, nil
		}

		return w.TxStore.RangeTransactions(txmgrNs, start, end, rangeFn)
	})
	return txList, err
}

// ListTransactions returns a slice of objects with details about a recorded
// transaction.  This is intended to be used for listtransactions RPC
// replies.
func (w *Wallet) ListTransactions(from, count int) ([]btcjson.ListTransactionsResult, er.R) {
	txList := []btcjson.ListTransactionsResult{}

	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

		// Get current block.  The block height used for calculating
		// the number of tx confirmations.
		syncBlock := w.Manager.SyncedTo()

		// Need to skip the first from transactions, and after those, only
		// include the next count transactions.
		skipped := 0
		n := 0

		rangeFn := func(details []wtxmgr.TxDetails) (bool, er.R) {
			// Iterate over transactions at this height in reverse order.
			// This does nothing for unmined transactions, which are
			// unsorted, but it will process mined transactions in the
			// reverse order they were marked mined.
			for i := len(details) - 1; i >= 0; i-- {
				if from > skipped {
					skipped++
					continue
				}

				n++
				if n > count {
					return true, nil
				}

				jsonResults := listTransactions(tx, &details[i],
					w.Manager, syncBlock.Height, w.chainParams)
				txList = append(txList, jsonResults...)

				if len(jsonResults) > 0 {
					n++
				}
			}

			return false, nil
		}

		// Return newer results first by starting at mempool height and working
		// down to the genesis block.
		return w.TxStore.RangeTransactions(txmgrNs, -1, 0, rangeFn)
	})
	return txList, err
}

// ListAddressTransactions returns a slice of objects with details about
// recorded transactions to or from any address belonging to a set.  This is
// intended to be used for listaddresstransactions RPC replies.
func (w *Wallet) ListAddressTransactions(pkHashes map[string]struct{}) ([]btcjson.ListTransactionsResult, er.R) {
	txList := []btcjson.ListTransactionsResult{}
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

		// Get current block.  The block height used for calculating
		// the number of tx confirmations.
		syncBlock := w.Manager.SyncedTo()
		rangeFn := func(details []wtxmgr.TxDetails) (bool, er.R) {
		loopDetails:
			for i := range details {
				detail := &details[i]

				for _, cred := range detail.Credits {
					pkScript := detail.MsgTx.TxOut[cred.Index].PkScript
					_, addrs, _, err := txscript.ExtractPkScriptAddrs(
						pkScript, w.chainParams)
					if err != nil || len(addrs) != 1 {
						continue
					}
					apkh, ok := addrs[0].(*btcutil.AddressPubKeyHash)
					if !ok {
						continue
					}
					_, ok = pkHashes[string(apkh.ScriptAddress())]
					if !ok {
						continue
					}

					jsonResults := listTransactions(tx, detail,
						w.Manager, syncBlock.Height, w.chainParams)
					if err != nil {
						return false, err
					}
					txList = append(txList, jsonResults...)
					continue loopDetails
				}
			}
			return false, nil
		}

		return w.TxStore.RangeTransactions(txmgrNs, 0, -1, rangeFn)
	})
	return txList, err
}

// ListAllTransactions returns a slice of objects with details about a recorded
// transaction.  This is intended to be used for listalltransactions RPC
// replies.
func (w *Wallet) ListAllTransactions() ([]btcjson.ListTransactionsResult, er.R) {
	txList := []btcjson.ListTransactionsResult{}
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

		// Get current block.  The block height used for calculating
		// the number of tx confirmations.
		syncBlock := w.Manager.SyncedTo()

		rangeFn := func(details []wtxmgr.TxDetails) (bool, er.R) {
			// Iterate over transactions at this height in reverse order.
			// This does nothing for unmined transactions, which are
			// unsorted, but it will process mined transactions in the
			// reverse order they were marked mined.
			for i := len(details) - 1; i >= 0; i-- {
				jsonResults := listTransactions(tx, &details[i], w.Manager,
					syncBlock.Height, w.chainParams)
				txList = append(txList, jsonResults...)
			}
			return false, nil
		}

		// Return newer results first by starting at mempool height and
		// working down to the genesis block.
		return w.TxStore.RangeTransactions(txmgrNs, -1, 0, rangeFn)
	})
	return txList, err
}

// creditSlice satisifies the sort.Interface interface to provide sorting
// transaction credits from oldest to newest.  Credits with the same receive
// time and mined in the same block are not guaranteed to be sorted by the order
// they appear in the block.  Credits from the same transaction are sorted by
// output index.
type creditSlice []wtxmgr.Credit

func (s creditSlice) Len() int {
	return len(s)
}

func (s creditSlice) Less(i, j int) bool {
	switch {
	// If both credits are from the same tx, sort by output index.
	case s[i].OutPoint.Hash == s[j].OutPoint.Hash:
		return s[i].OutPoint.Index < s[j].OutPoint.Index

	// If both transactions are unmined, sort by their received date.
	case s[i].Height == -1 && s[j].Height == -1:
		return s[i].Received.Before(s[j].Received)

	// Unmined (newer) txs always come last.
	case s[i].Height == -1:
		return false
	case s[j].Height == -1:
		return true

	// If both txs are mined in different blocks, sort by block height.
	default:
		return s[i].Height < s[j].Height
	}
}

func (s creditSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// ListUnspent returns a slice of objects representing the unspent wallet
// transactions fitting the given criteria. The confirmations will be more than
// minconf, less than maxconf and if addresses is populated only the addresses
// contained within it will be considered.  If we know nothing about a
// transaction an empty array will be returned.
func (w *Wallet) ListUnspent(minconf, maxconf int32,
	addresses map[string]struct{}) ([]*btcjson.ListUnspentResult, er.R) {

	var results []*btcjson.ListUnspentResult
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

		syncBlock := w.Manager.SyncedTo()

		filter := len(addresses) != 0
		unspent, err := w.TxStore.GetUnspentOutputs(txmgrNs)
		if err != nil {
			return err
		}
		sort.Sort(sort.Reverse(creditSlice(unspent)))

		defaultAccountName := "default"

		results = make([]*btcjson.ListUnspentResult, 0, len(unspent))
		for i := range unspent {
			output := unspent[i]

			// Outputs with fewer confirmations than the minimum or more
			// confs than the maximum are excluded.
			confs := confirms(output.Height, syncBlock.Height)
			if confs < minconf || confs > maxconf {
				continue
			}

			// Only mature coinbase outputs are included.
			if output.FromCoinBase {
				target := int32(w.ChainParams().CoinbaseMaturity)
				if !confirmed(target, output.Height, syncBlock.Height) {
					continue
				}
			}

			// Exclude locked outputs from the result set.
			if w.LockedOutpoint(output.OutPoint) {
				continue
			}

			// Lookup the associated account for the output.  Use the
			// default account name in case there is no associated account
			// for some reason, although this should never happen.
			//
			// This will be unnecessary once transactions and outputs are
			// grouped under the associated account in the db.
			acctName := defaultAccountName
			sc, addrs, _, err := txscript.ExtractPkScriptAddrs(
				output.PkScript, w.chainParams)
			if err != nil {
				continue
			}
			if len(addrs) > 0 {
				smgr, acct, err := w.Manager.AddrAccount(addrmgrNs, addrs[0])
				if err == nil {
					s, err := smgr.AccountName(addrmgrNs, acct)
					if err == nil {
						acctName = s
					}
				}
			}

			if filter {
				for _, addr := range addrs {
					_, ok := addresses[addr.EncodeAddress()]
					if ok {
						goto include
					}
				}
				continue
			}

		include:
			// At the moment watch-only addresses are not supported, so all
			// recorded outputs that are not multisig are "spendable".
			// Multisig outputs are only "spendable" if all keys are
			// controlled by this wallet.
			//
			// TODO: Each case will need updates when watch-only addrs
			// is added.  For P2PK, P2PKH, and P2SH, the address must be
			// looked up and not be watching-only.  For multisig, all
			// pubkeys must belong to the manager with the associated
			// private key (currently it only checks whether the pubkey
			// exists, since the private key is required at the moment).
			var spendable bool
		scSwitch:
			switch sc {
			case txscript.PubKeyHashTy:
				spendable = true
			case txscript.PubKeyTy:
				spendable = true
			case txscript.WitnessV0ScriptHashTy:
				spendable = true
			case txscript.WitnessV0PubKeyHashTy:
				spendable = true
			case txscript.MultiSigTy:
				for _, a := range addrs {
					_, err := w.Manager.Address(addrmgrNs, a)
					if err == nil {
						continue
					}
					if waddrmgr.ErrAddressNotFound.Is(err) {
						break scSwitch
					}
					return err
				}
				spendable = true
			}

			result := &btcjson.ListUnspentResult{
				TxID:          output.OutPoint.Hash.String(),
				Vout:          output.OutPoint.Index,
				Account:       acctName,
				ScriptPubKey:  hex.EncodeToString(output.PkScript),
				Amount:        output.Amount.ToBTC(),
				Confirmations: int64(confs),
				Spendable:     spendable,
			}

			// BUG: this should be a JSON array so that all
			// addresses can be included, or removed (and the
			// caller extracts addresses from the pkScript).
			if len(addrs) > 0 {
				result.Address = addrs[0].EncodeAddress()
			}

			results = append(results, result)
		}
		return nil
	})
	return results, err
}

// DumpWIFPrivateKey returns the WIF encoded private key for a
// single wallet address.
func (w *Wallet) DumpWIFPrivateKey(addr btcutil.Address) (string, er.R) {
	var maddr waddrmgr.ManagedAddress
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		waddrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		// Get private key from wallet if it exists.
		var err er.R
		maddr, err = w.Manager.Address(waddrmgrNs, addr)
		return err
	})
	if err != nil {
		return "", err
	}

	pka, ok := maddr.(waddrmgr.ManagedPubKeyAddress)
	if !ok {
		return "", er.Errorf("address %s is not a key type", addr)
	}

	wif, err := pka.ExportPrivKey()
	if err != nil {
		return "", err
	}
	return wif.String(), nil
}

// ImportPrivateKey imports a private key to the wallet and writes the new
// wallet to disk.
//
// NOTE: If a block stamp is not provided, then the wallet's birthday will be
// set to the genesis block of the corresponding chain.
func (w *Wallet) ImportPrivateKey(scope waddrmgr.KeyScope, wif *btcutil.WIF,
	bs *waddrmgr.BlockStamp, rescan bool) (string, er.R) {

	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return "", err
	}

	// The starting block for the key is the genesis block unless otherwise
	// specified.
	if bs == nil {
		bs = &waddrmgr.BlockStamp{
			Hash:      *w.chainParams.GenesisHash,
			Height:    0,
			Timestamp: genesis.Block(w.chainParams.GenesisHash).Header.Timestamp,
		}
	} else if bs.Timestamp.IsZero() {
		// Only update the new birthday time from default value if we
		// actually have timestamp info in the header.
		header, err := w.chainClient.GetBlockHeader(&bs.Hash)
		if err == nil {
			bs.Timestamp = header.Timestamp
		}
	}

	// Attempt to import private key into wallet.
	var addr btcutil.Address
	var props *waddrmgr.AccountProperties
	err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) er.R {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		maddr, err := manager.ImportPrivateKey(addrmgrNs, wif, bs)
		if err != nil {
			return err
		}
		addr = maddr.Address()
		props, err = manager.AccountProperties(
			addrmgrNs, waddrmgr.ImportedAddrAccount,
		)
		if err != nil {
			return err
		}

		// We'll only update our birthday with the new one if it is
		// before our current one. Otherwise, if we do, we can
		// potentially miss detecting relevant chain events that
		// occurred between them while rescanning.
		birthdayBlock, _, err := w.Manager.BirthdayBlock(addrmgrNs)
		if err != nil {
			return err
		}
		if bs.Height >= birthdayBlock.Height {
			return nil
		}

		err = w.Manager.SetBirthday(addrmgrNs, bs.Timestamp)
		if err != nil {
			return err
		}

		// To ensure this birthday block is correct, we'll mark it as
		// unverified to prompt a sanity check at the next restart to
		// ensure it is correct as it was provided by the caller.
		return w.Manager.SetBirthdayBlock(addrmgrNs, *bs, false)
	})
	if err != nil {
		return "", err
	}

	// Rescan blockchain for transactions with txout scripts paying to the
	// imported address.
	if rescan {
		job := &RescanJob{
			Addrs:      []btcutil.Address{addr},
			OutPoints:  nil,
			BlockStamp: *bs,
		}

		// Submit rescan job and log when the import has completed.
		// Do not block on finishing the rescan.  The rescan success
		// or failure is logged elsewhere, and the channel is not
		// required to be read, so discard the return value.
		_ = w.SubmitRescan(job)
	} else {
		err := w.chainClient.NotifyReceived([]btcutil.Address{addr})
		if err != nil {
			return "", er.Errorf("Failed to subscribe for address ntfns for "+
				"address %s: %s", addr.EncodeAddress(), err)
		}
	}

	addrStr := addr.EncodeAddress()
	log.Infof("Imported payment address %s", addrStr)

	w.NtfnServer.notifyAccountProperties(props)

	// Return the payment address string of the imported private key.
	return addrStr, nil
}

// LockedOutpoint returns whether an outpoint has been marked as locked and
// should not be used as an input for created transactions.
func (w *Wallet) LockedOutpoint(op wire.OutPoint) bool {
	_, locked := w.lockedOutpoints[op]
	return locked
}

// LockOutpoint marks an outpoint as locked, that is, it should not be used as
// an input for newly created transactions.
func (w *Wallet) LockOutpoint(op wire.OutPoint, name string) {
	w.lockedOutpoints[op] = name
}

// UnlockOutpoint marks an outpoint as unlocked, that is, it may be used as an
// input for newly created transactions.
func (w *Wallet) UnlockOutpoint(op wire.OutPoint) {
	delete(w.lockedOutpoints, op)
}

// ResetLockedOutpoints resets the set of locked outpoints so all may be used
// as inputs for new transactions.
func (w *Wallet) ResetLockedOutpoints(lockName *string) {
	if lockName != nil {
		for op, ln := range w.lockedOutpoints {
			if ln == *lockName {
				delete(w.lockedOutpoints, op)
			}
		}
	} else {
		w.lockedOutpoints = map[wire.OutPoint]string{}
	}
}

// LockedOutpoints returns a slice of currently locked outpoints.  This is
// intended to be used by marshaling the result as a JSON array for
// listlockunspent RPC results.
func (w *Wallet) LockedOutpoints() []btcjson.LockedUnspent {
	locked := make([]btcjson.LockedUnspent, len(w.lockedOutpoints))
	i := 0
	for op, ln := range w.lockedOutpoints {
		locked[i] = btcjson.LockedUnspent{
			Txid:     op.Hash.String(),
			Vout:     op.Index,
			LockName: ln,
		}
		i++
	}
	return locked
}

// resendUnminedTxs iterates through all transactions that spend from wallet
// credits that are not known to have been mined into a block, and attempts
// to send each to the chain server for relay.
func (w *Wallet) resendUnminedTxs() {
	var txs []*wire.MsgTx
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)
		var err er.R
		txs, err = w.TxStore.UnminedTxs(txmgrNs)
		return err
	})
	if err != nil {
		log.Errorf("Unable to retrieve unconfirmed transactions to "+
			"resend: %v", err)
		return
	}

	for _, tx := range txs {
		txHash, err := w.publishTransaction(tx)
		if err != nil {
			log.Debugf("Unable to rebroadcast transaction %v: %v",
				tx.TxHash(), err)
			continue
		}

		log.Debugf("Successfully rebroadcast unconfirmed transaction %v",
			txHash)
	}
}

// SortedActivePaymentAddresses returns a slice of all active payment
// addresses in a wallet.
func (w *Wallet) SortedActivePaymentAddresses() ([]string, er.R) {
	var addrStrs []string
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		return w.Manager.ForEachActiveAddress(addrmgrNs, func(addr btcutil.Address) er.R {
			addrStrs = append(addrStrs, addr.EncodeAddress())
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(addrStrs)
	return addrStrs, nil
}

// NewAddress returns the next external chained address for a wallet.
func (w *Wallet) NewAddress(account uint32,
	scope waddrmgr.KeyScope) (btcutil.Address, er.R) {

	chainClient, err := w.requireChainClient()
	if err != nil {
		return nil, err
	}

	var (
		addr  btcutil.Address
		props *waddrmgr.AccountProperties
	)
	err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) er.R {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		var err er.R
		addr, props, err = w.newAddress(addrmgrNs, account, scope)
		return err
	})
	if err != nil {
		return nil, err
	}

	// Notify the rpc server about the newly created address.
	err = chainClient.NotifyReceived([]btcutil.Address{addr})
	if err != nil {
		return nil, err
	}

	w.NtfnServer.notifyAccountProperties(props)

	return addr, nil
}

func (w *Wallet) newAddress(addrmgrNs walletdb.ReadWriteBucket, account uint32,
	scope waddrmgr.KeyScope) (btcutil.Address, *waddrmgr.AccountProperties, er.R) {

	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return nil, nil, err
	}

	// Get next address from wallet.
	addrs, err := manager.NextExternalAddresses(addrmgrNs, account, 1)
	if err != nil {
		return nil, nil, err
	}

	props, err := manager.AccountProperties(addrmgrNs, account)
	if err != nil {
		log.Errorf("Cannot fetch account properties for notification "+
			"after deriving next external address: %v", err)
		return nil, nil, err
	}

	return addrs[0].Address(), props, nil
}

// confirmed checks whether a transaction at height txHeight has met minconf
// confirmations for a blockchain at height curHeight.
func confirmed(minconf, txHeight, curHeight int32) bool {
	return confirms(txHeight, curHeight) >= minconf
}

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

// TotalReceivedForAddr iterates through a wallet's transaction history,
// returning the total amount of bitcoins received for a single wallet
// address.
func (w *Wallet) TotalReceivedForAddr(addr btcutil.Address, minConf int32) (btcutil.Amount, er.R) {
	var amount btcutil.Amount
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

		syncBlock := w.Manager.SyncedTo()

		var (
			addrStr    = addr.EncodeAddress()
			stopHeight int32
		)

		if minConf > 0 {
			stopHeight = syncBlock.Height - minConf + 1
		} else {
			stopHeight = -1
		}
		rangeFn := func(details []wtxmgr.TxDetails) (bool, er.R) {
			for i := range details {
				detail := &details[i]
				for _, cred := range detail.Credits {
					pkScript := detail.MsgTx.TxOut[cred.Index].PkScript
					_, addrs, _, err := txscript.ExtractPkScriptAddrs(pkScript,
						w.chainParams)
					// An error creating addresses from the output script only
					// indicates a non-standard script, so ignore this credit.
					if err != nil {
						continue
					}
					for _, a := range addrs {
						if addrStr == a.EncodeAddress() {
							amount += cred.Amount
							break
						}
					}
				}
			}
			return false, nil
		}
		return w.TxStore.RangeTransactions(txmgrNs, 0, stopHeight, rangeFn)
	})
	return amount, err
}

// SendOutputs creates and sends payment transactions. It returns the
// transaction upon success.
func (w *Wallet) SendOutputs(txr CreateTxReq) (*txauthor.AuthoredTx, er.R) {

	// Ensure the outputs to be created adhere to the network's consensus
	// rules.
	hasSweep := false
	for _, output := range txr.Outputs {
		if output.Value == 0 {
			if hasSweep {
				return nil, er.New("Multiple outputs with zero value, a single output with zero value " +
					"will sweep the address(es) to this output, multiple zero value outputs are ambiguous")
			}
			hasSweep = true
			continue
		}
		err := txrules.CheckOutput(
			output, txrules.DefaultRelayFeePerKb,
		)
		if err != nil {
			return nil, err
		}
	}

	// Create the transaction and broadcast it to the network. The
	// transaction will be added to the database in order to ensure that we
	// continue to re-broadcast the transaction upon restarts until it has
	// been confirmed.
	createdTx, err := w.CreateSimpleTx(txr)
	if err != nil {
		return nil, err
	}
	if txr.DryRun {
		return createdTx, nil
	}

	txHash, err := w.reliablyPublishTransaction(createdTx.Tx)
	if err != nil {
		return nil, err
	}

	// Sanity check on the returned tx hash.
	if *txHash != createdTx.Tx.TxHash() {
		return nil, er.New("tx hash mismatch")
	}

	return createdTx, nil
}

// SignatureError records the underlying error when validating a transaction
// input signature.
type SignatureError struct {
	InputIndex uint32
	Error      er.R
}

// SignTransaction uses secrets of the wallet, as well as additional secrets
// passed in by the caller, to create and add input signatures to a transaction.
//
// Transaction input script validation is used to confirm that all signatures
// are valid.  For any invalid input, a SignatureError is added to the returns.
// The final error return is reserved for unexpected or fatal errors, such as
// being unable to determine a previous output script to redeem.
//
// The transaction pointed to by tx is modified by this function.
func (w *Wallet) SignTransaction(tx *wire.MsgTx, hashType params.SigHashType,
	additionalPrevScripts map[wire.OutPoint][]byte,
	additionalKeysByAddress map[string]*btcutil.WIF,
	p2shRedeemScriptsByAddress map[string][]byte,
) ([]SignatureError, er.R) {

	hashCache := txscript.NewTxSigHashes(tx)
	if len(tx.Additional) == 0 {
		tx.Additional = make([]wire.TxInAdditional, len(tx.TxIn))
	} else if len(tx.Additional) != len(tx.TxIn) {
		return nil, er.New("tx contains Additional field of unexpected length")
	}

	var signErrors []SignatureError
	err := walletdb.View(w.db, func(dbtx walletdb.ReadTx) er.R {
		addrmgrNs := dbtx.ReadBucket(waddrmgrNamespaceKey)
		txmgrNs := dbtx.ReadBucket(wtxmgrNamespaceKey)

		for i, txIn := range tx.TxIn {
			prevPkScript, ok := additionalPrevScripts[txIn.PreviousOutPoint]
			if ok {
				tx.Additional[i].PkScript = prevPkScript
			}
			if len(tx.Additional[i].PkScript) > 0 && tx.Additional[i].Value != nil {
				// PkScript is included in the tx already
			} else {
				prevHash := &txIn.PreviousOutPoint.Hash
				prevIndex := txIn.PreviousOutPoint.Index
				txDetails, err := w.TxStore.TxDetails(txmgrNs, prevHash)
				if err != nil {
					return er.Errorf("cannot query previous transaction "+
						"details for %v: %v", txIn.PreviousOutPoint, err)
				}
				if txDetails == nil {
					return er.Errorf("%v not found",
						txIn.PreviousOutPoint)
				}
				if len(tx.Additional[i].PkScript) == 0 {
					tx.Additional[i].PkScript = txDetails.MsgTx.TxOut[prevIndex].PkScript
				}
				if tx.Additional[i].Value == nil {
					v := txDetails.MsgTx.TxOut[prevIndex].Value
					tx.Additional[i].Value = &v
				}
			}

			// Set up our callbacks that we pass to txscript so it can
			// look up the appropriate keys and scripts by address.
			getKey := txscript.KeyClosure(func(addr btcutil.Address) (*btcec.PrivateKey, bool, er.R) {
				if len(additionalKeysByAddress) != 0 {
					addrStr := addr.EncodeAddress()
					wif, ok := additionalKeysByAddress[addrStr]
					if !ok {
						return nil, false,
							er.New("no key for address")
					}
					return wif.PrivKey, wif.CompressPubKey, nil
				}
				address, err := w.Manager.Address(addrmgrNs, addr)
				if err != nil {
					return nil, false, err
				}

				pka, ok := address.(waddrmgr.ManagedPubKeyAddress)
				if !ok {
					return nil, false, er.Errorf("address %v is not "+
						"a pubkey address", address.Address().EncodeAddress())
				}

				key, err := pka.PrivKey()
				if err != nil {
					return nil, false, err
				}

				return key, pka.Compressed(), nil
			})
			getScript := txscript.ScriptClosure(func(addr btcutil.Address) ([]byte, er.R) {
				// If keys were provided then we can only use the
				// redeem scripts provided with our inputs, too.
				if len(additionalKeysByAddress) != 0 {
					addrStr := addr.EncodeAddress()
					script, ok := p2shRedeemScriptsByAddress[addrStr]
					if !ok {
						return nil, er.New("no script for address")
					}
					return script, nil
				}
				address, err := w.Manager.Address(addrmgrNs, addr)
				if err != nil {
					return nil, err
				}
				sa, ok := address.(waddrmgr.ManagedScriptAddress)
				if !ok {
					return nil, er.New("address is not a script" +
						" address")
				}

				return sa.Script()
			})

			// SigHashSingle inputs can only be signed if there's a
			// corresponding output. However this could be already signed,
			// so we always verify the output.
			if (hashType&params.SigHashSingle) !=
				params.SigHashSingle || i < len(tx.TxOut) {
				if err := txauthor.SignInputScript(
					tx,
					i,
					hashType,
					hashCache,
					getKey,
					getScript,
					w.ChainParams(),
				); err != nil {
					// Failure to sign isn't an error, it just means that
					// the tx isn't complete.
					signErrors = append(signErrors, SignatureError{
						InputIndex: uint32(i),
						Error:      err,
					})
					continue
				}
			}

			// Either it was already signed or we just signed it.
			// Find out if it is completely satisfied or still needs more.
			var v int64
			if tx.Additional[i].Value != nil {
				v = *tx.Additional[i].Value
			}
			vm, err := txscript.NewEngine(tx.Additional[i].PkScript, tx, i,
				txscript.StandardVerifyFlags, nil, hashCache, v)
			if err == nil {
				err = vm.Execute()
			}
			if err != nil {
				signErrors = append(signErrors, SignatureError{
					InputIndex: uint32(i),
					Error:      err,
				})
			}
		}
		return nil
	})
	return signErrors, err
}

// reliablyPublishTransaction is a superset of publishTransaction which contains
// the primary logic required for publishing a transaction, updating the
// relevant database state, and finally possible removing the transaction from
// the database (along with cleaning up all inputs used, and outputs created) if
// the transaction is rejected by the backend.
func (w *Wallet) reliablyPublishTransaction(tx *wire.MsgTx) (*chainhash.Hash, er.R) {
	chainClient, err := w.requireChainClient()
	if err != nil {
		return nil, err
	}

	// As we aim for this to be general reliable transaction broadcast API,
	// we'll write this tx to disk as an unconfirmed transaction. This way,
	// upon restarts, we'll always rebroadcast it, and also add it to our
	// set of records.
	txRec, err := wtxmgr.NewTxRecordFromMsgTx(tx, time.Now())
	if err != nil {
		return nil, err
	}
	err = walletdb.Update(w.db, func(dbTx walletdb.ReadWriteTx) er.R {
		return w.addRelevantTx(dbTx, txRec, nil)
	})
	if err != nil {
		return nil, err
	}

	// We'll also ask to be notified of the transaction once it confirms
	// on-chain. This is done outside of the database transaction to prevent
	// backend interaction within it.
	//
	// NOTE: In some cases, it's possible that the transaction to be
	// broadcast is not directly relevant to the user's wallet, e.g.,
	// multisig. In either case, we'll still ask to be notified of when it
	// confirms to maintain consistency.
	//
	// TODO(wilmer): import script as external if the address does not
	// belong to the wallet to handle confs during restarts?
	for _, txOut := range tx.TxOut {
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(
			txOut.PkScript, w.chainParams,
		)
		if err != nil {
			// Non-standard outputs can safely be skipped because
			// they're not supported by the wallet.
			continue
		}

		if err := chainClient.NotifyReceived(addrs); err != nil {
			return nil, err
		}
	}

	return w.publishTransaction(tx)
}

// publishTransaction attempts to send an unconfirmed transaction to the
// wallet's current backend. In the event that sending the transaction fails for
// whatever reason, it will be removed from the wallet's unconfirmed transaction
// store.
func (w *Wallet) publishTransaction(tx *wire.MsgTx) (*chainhash.Hash, er.R) {
	chainClient, err := w.requireChainClient()
	if err != nil {
		return nil, err
	}

	_, err = chainClient.SendRawTransaction(tx, false)

	// If we're in neutrino land here then we're going to get a pushtx.Err
	// but if we're using the RPC then we're going to get a

	// Determine if this was an RPC error thrown due to the transaction
	// already confirming.
	txid := tx.TxHash()
	switch {
	case err == nil:
		return &txid, nil

	// This error is returned when broadcasting a transaction to a bitcoind
	// node that already has it in their mempool.
	// TODO(cjd): We should not be parsing errors here looking for "txn-already-in-mempool"
	// 						but if we're using the JSON RPC then we don't have a specific error
	//            code for "already in mempool".
	case pushtx.RejMempool.Is(err):
		fallthrough
	case strings.Contains(
		strings.ToLower(err.Message()), "txn-already-in-mempool",
	):
		return &txid, nil

	// If the transaction has already confirmed, we can safely remove it
	// from the unconfirmed store as it should already exist within the
	// confirmed store. We'll avoid returning an error as the broadcast was
	// in a sense successful.
	//
	// This error is returned when sending a transaction that has already
	// confirmed to a pktd/bitcoind node over RPC.
	case pushtx.RejConfirmed.Is(err):
		fallthrough
	case btcjson.ErrRPCTxAlreadyInChain.Is(err):
		dbErr := walletdb.Update(w.db, func(dbTx walletdb.ReadWriteTx) er.R {
			txmgrNs := dbTx.ReadWriteBucket(wtxmgrNamespaceKey)
			txRec, err := wtxmgr.NewTxRecordFromMsgTx(tx, time.Now())
			if err != nil {
				return err
			}
			return w.TxStore.RemoveUnminedTx(txmgrNs, txRec)
		})
		if dbErr != nil {
			log.Warnf("Unable to remove confirmed transaction %v "+
				"from unconfirmed store: %v", tx.TxHash(), dbErr)
		}

		return &txid, nil

	// If the transaction was rejected for whatever other reason, then we'll
	// remove it from the transaction store, as otherwise, we'll attempt to
	// continually re-broadcast it, and the UTXO state of the wallet won't
	// be accurate.
	default:
		dbErr := walletdb.Update(w.db, func(dbTx walletdb.ReadWriteTx) er.R {
			txmgrNs := dbTx.ReadWriteBucket(wtxmgrNamespaceKey)
			txRec, err := wtxmgr.NewTxRecordFromMsgTx(tx, time.Now())
			if err != nil {
				return err
			}
			return w.TxStore.RemoveUnminedTx(txmgrNs, txRec)
		})
		if dbErr != nil {
			log.Warnf("Unable to remove invalid transaction %v: %v",
				tx.TxHash(), dbErr)
		} else {
			// This thing creates enormous noise
			//log.Infof("Removed invalid transaction: %v", spew.Sdump(tx))
		}

		return nil, err
	}
}

// ChainParams returns the network parameters for the blockchain the wallet
// belongs to.
func (w *Wallet) ChainParams() *chaincfg.Params {
	return w.chainParams
}

// Create creates an new wallet, writing it to an empty database.  If the passed
// seed is non-nil, it is used.  Otherwise, a secure random seed of the
// recommended length is generated.
func Create(db walletdb.DB, pubPass, privPass []byte, seedInput []byte,
	seedx *seedwords.Seed, params *chaincfg.Params) er.R {

	// If a seed was provided, ensure that it is of valid length. Otherwise,
	// we generate a random seed for the wallet with the recommended seed
	// length.
	var legacySeed []byte
	if seedx != nil {
	} else if seedbin, err := hex.DecodeString(string(seedInput)); err == nil {
		// it's a legacy seed, we need to just support it
		if len(seedbin) < hdkeychain.MinSeedBytes ||
			len(seedbin) > hdkeychain.MaxSeedBytes {
			return hdkeychain.ErrInvalidSeedLen.Default()
		}
		legacySeed = seedbin
	} else {
		return er.New("No seed provided")
	}

	var birthday time.Time
	if seedx != nil {
		birthday = seedx.Birthday()
	} else {
		// If we don't know the bday, put it before all of this began
		birthday = time.Unix(1231006505, 0)
	}

	return walletdb.Update(db, func(tx walletdb.ReadWriteTx) er.R {
		addrmgrNs, err := tx.CreateTopLevelBucket(waddrmgrNamespaceKey)
		if err != nil {
			return err
		}
		txmgrNs, err := tx.CreateTopLevelBucket(wtxmgrNamespaceKey)
		if err != nil {
			return err
		}

		err = waddrmgr.Create(
			addrmgrNs, legacySeed, seedx, pubPass, privPass, params, nil,
			birthday,
		)
		if err != nil {
			return err
		}
		return wtxmgr.Create(txmgrNs)
	})
}

// ResyncChain re-synchronizes the wallet from the very first block
func (w *Wallet) ResyncChain(dropDb bool) er.R {
	chainClient, err := w.requireChainClient()
	if err != nil {
		return err
	}
	genesis := waddrmgr.BlockStamp{
		Hash:   *w.ChainParams().GenesisHash,
		Height: 0,
	}
	if err := walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) er.R {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		if err := w.Manager.SetSyncedTo(ns, &genesis); err != nil {
			return err
		}

		if dropDb {
			txNs := tx.ReadWriteBucket(wtxmgrNamespaceKey)
			if err := wtxmgr.DropTransactionHistory(txNs); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}
	return w.recovery(chainClient, &genesis)
}

func (w *Wallet) WalletMempool() ([]wtxmgr.TxDetails, er.R) {
	var unminedTxDetails []wtxmgr.TxDetails
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) er.R {
		ns := tx.ReadBucket([]byte("wtxmgr"))
		return w.TxStore.RangeTransactions(ns, -1, -1, func(utxd []wtxmgr.TxDetails) (bool, er.R) {
			unminedTxDetails = utxd
			return true, nil
		})
	})
	return unminedTxDetails, err
}

func (w *Wallet) RescanMempoolTxns() er.R {
	if !w.rescanMempoolMu.TryLock() {
		return ErrInProgress.Default()
	}
	defer func() {
		log.Debugf("RescanMempoolTxns() done")
		w.rescanMempoolMu.Unlock()
	}()

	var (
		addrs   []btcutil.Address
		unspent []wtxmgr.Credit
	)
	if txDetails, err := w.WalletMempool(); err != nil {
		return err
	} else if incl := (func() (incl []wtxmgr.TxDetails) {
		for _, td := range txDetails {
			if td.Received.Unix() < time.Now().Unix()-3600 {
				incl = append(incl, td)
			}
		}
		return
	})(); len(incl) == 0 {
		return nil
	} else if err := walletdb.View(w.db, func(dbtx walletdb.ReadTx) er.R {
		addrs, unspent, err = w.activeData(dbtx)
		return err
	}); err != nil {
		return err
	} else {
		earliest := incl[0].Received
		log.Debugf("RescanMempoolTxns() [%d] mempool transactions, scanning to see if they're confirmed",
			len(incl))
		for _, c := range incl {
			for _, out := range c.MsgTx.TxOut {
				addrs = append(addrs, txscript.PkScriptToAddress(out.PkScript, w.chainParams))
				break
			}
			if c.Received.Before(earliest) {
				earliest = c.Received
			}
		}
		if bs, err := locateBirthdayBlock(w.chainClient, earliest); err != nil {
			return err
		} else {
			saddrs := make([]string, 0, len(addrs))
			for _, a := range addrs {
				saddrs = append(saddrs, a.String())
			}
			log.Debugf("RescanMempoolTxns() Begin scan from height [%d] for addresses [%s]",
				bs.Height, strings.Join(saddrs, ", "))
			return w.rescanWithTarget(addrs, unspent, bs)
		}
	}
}

// zero duration means it will continue vacuuming until it is done, no matter how long.
func (w *Wallet) VacuumDb(startKey string, maxTime time.Duration) (*btcjson.VacuumDbRes, er.R) {
	deadline := time.Now().Add(maxTime)
	stats := btcjson.VacuumDbRes{}
	if sk, errr := hex.DecodeString(startKey); errr != nil {
		return nil, er.E(errr)
	} else if chainClient, err := w.requireChainClient(); err != nil {
		return nil, err
	} else if bs, err := chainClient.BlockStamp(); err != nil {
		return nil, err
	} else if err := walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) er.R {
		txNs := tx.ReadWriteBucket(wtxmgrNamespaceKey)
		var badOutputs []wtxmgr.Credit
		i := 0
		if err := w.TxStore.ForEachUnspentOutput(txNs, sk, func(k []byte, op *wtxmgr.Credit) er.R {
			if maxTime > 0 && time.Now().After(deadline) {
				stats.EndKey = hex.EncodeToString(k)
				return er.LoopBreak
			}
			i++
			if txrules.IsBurned(op, w.chainParams, bs.Height) {
				log.Debugf("Removing tx [%s] which has burned",
					op.OutPoint.Hash.String())
				badOutputs = append(badOutputs, *op)
				stats.Burned++
				return nil
			}
			txD, err := w.TxStore.TxDetails(txNs, &op.Hash)
			if err != nil {
				return err
			}
			if op.Height < 0 {
				if txD == nil {
					log.Errorf("Unmined UTXO %s "+
						"has no accompanying transaction in the db", op.String())
					// TODO(cjd): Need to somehow fix this
					return nil
				}
				if txD.Block.Height != -1 {
					log.Errorf("Unmined UTXO %s "+
						"is mined according to the tx db", op.String())
					// TODO(cjd): Need to somehow fix this
					return nil
				}
				return nil
			}

			if txD == nil {
				log.Errorf("Mined UTXO %s "+
					"has no accompanying record in tx db", op.String())
				return nil
			}
			if txD.Block.Height == -1 {
				log.Errorf("Mined UTXO %s "+
					"is unmined according to the tx db", op.String())
				return nil
			}

			if _, err := chainClient.GetBlockHeader(&op.Block.Hash); err != nil {
				if !headerfs.ErrHashNotFound.Is(err) {
					// Don't confuse with a real error
					return err
				}
				// The block containing the previous transaction which paid this one
				// has gone missing, if it's a coinbase then we need to kill it because
				// it's never coming back. If it's a regular transaction we're going to
				// revert it and put it back in the mempool.
				log.Debugf("Removing tx [%s] because it references orphan block [%s]",
					op.OutPoint.Hash.String(), op.Block.Hash)
				badOutputs = append(badOutputs, *op)
				stats.Orphaned++
				return nil
			}

			return nil
		}); err != nil && !er.IsLoopBreak(err) {
			return err
		} else {
			stats.VisitedUtxos = i
			badHashes := map[chainhash.Hash]struct{}{}
			for _, op := range badOutputs {
				if _, ok := badHashes[op.OutPoint.Hash]; ok {
				} else if err := wtxmgr.RollbackTransaction(txNs, &op.OutPoint.Hash, &op.Block, w.chainParams); err != nil {
					return err
				} else {
					badHashes[op.OutPoint.Hash] = struct{}{}
				}
			}

			if err == nil {
				// it's not a loopbreak, we're at the end of the line, lets scan for unconfirmed txns
				return w.RescanMempoolTxns()
			}
		}
		return nil
	}); err != nil {
		return &stats, err
	}
	return &stats, nil
}

// Open loads an already-created wallet from the passed database and namespaces.
func Open(db walletdb.DB, pubPass []byte, cbs *waddrmgr.OpenCallbacks,
	params *chaincfg.Params, recoveryWindow uint32) (*Wallet, er.R) {

	var (
		addrMgr *waddrmgr.Manager
		txMgr   *wtxmgr.Store
	)

	// Before attempting to open the wallet, we'll check if there are any
	// database upgrades for us to proceed. We'll also create our references
	// to the address and transaction managers, as they are backed by the
	// database.
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) er.R {
		addrMgrBucket := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		if addrMgrBucket == nil {
			return er.New("missing address manager namespace")
		}
		txMgrBucket := tx.ReadWriteBucket(wtxmgrNamespaceKey)
		if txMgrBucket == nil {
			return er.New("missing transaction manager namespace")
		}

		addrMgrUpgrader := waddrmgr.NewMigrationManager(addrMgrBucket)
		txMgrUpgrader := wtxmgr.NewMigrationManager(txMgrBucket)
		err := migration.Upgrade(txMgrUpgrader, addrMgrUpgrader)
		if err != nil {
			return err
		}

		addrMgr, err = waddrmgr.Open(addrMgrBucket, pubPass, params)
		if err != nil {
			return err
		}
		txMgr, err = wtxmgr.Open(txMgrBucket, params)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Infof("Opened wallet") // TODO: log balance? last sync height?

	w := &Wallet{
		publicPassphrase:    pubPass,
		db:                  db,
		Manager:             addrMgr,
		TxStore:             txMgr,
		lockedOutpoints:     map[wire.OutPoint]string{},
		recoveryWindow:      recoveryWindow,
		rescanAddJob:        make(chan *RescanJob),
		rescanBatch:         make(chan *rescanBatch),
		rescanNotifications: make(chan interface{}),
		rescanProgress:      make(chan *RescanProgressMsg),
		rescanFinished:      make(chan *RescanFinishedMsg),
		createTxRequests:    make(chan createTxRequest),
		unlockRequests:      make(chan unlockRequest),
		lockRequests:        make(chan struct{}),
		holdUnlockRequests:  make(chan chan heldUnlock),
		lockState:           make(chan bool),
		changePassphrase:    make(chan changePassphraseRequest),
		changePassphrases:   make(chan changePassphrasesRequest),
		chainParams:         params,
		quit:                make(chan struct{}),
	}

	w.NtfnServer = newNotificationServer(w)
	w.TxStore.NotifyUnspent = func(hash *chainhash.Hash, index uint32) {
		w.NtfnServer.notifyUnspentOutput(0, hash, index)
	}

	return w, nil
}

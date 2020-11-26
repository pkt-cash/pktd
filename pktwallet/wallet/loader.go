// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/pktwallet/internal/prompt"
	"github.com/pkt-cash/pktd/pktwallet/waddrmgr"
	"github.com/pkt-cash/pktd/pktwallet/wallet/seedwords"
	"github.com/pkt-cash/pktd/pktwallet/walletdb"
	"github.com/pkt-cash/pktd/pktwallet/walletdb/bdb"
	"go.etcd.io/bbolt"
)

var Err er.ErrorType = er.NewErrorType("wallet.Err")

var (
	// ErrLoaded describes the error condition of attempting to load or
	// create a wallet when the loader has already done so.
	ErrLoaded = Err.CodeWithDetail("ErrLoaded",
		"wallet already loaded")

	// ErrNotLoaded describes the error condition of attempting to close a
	// loaded wallet when a wallet has not been loaded.
	ErrNotLoaded = Err.CodeWithDetail("ErrNotLoaded",
		"wallet is not loaded")

	// ErrExists describes the error condition of attempting to create a new
	// wallet when one exists already.
	ErrExists = Err.CodeWithDetail("ErrExists",
		"wallet already exists")
)

// Loader implements the creating of new and opening of existing wallets, while
// providing a callback system for other subsystems to handle the loading of a
// wallet.  This is primarily intended for use by the RPC servers, to enable
// methods and services which require the wallet when the wallet is loaded by
// another subsystem.
//
// Loader is safe for concurrent access.
type Loader struct {
	callbacks      []func(*Wallet)
	chainParams    *chaincfg.Params
	dbDirPath      string
	noFreelistSync bool
	walletName     string
	recoveryWindow uint32
	wallet         *Wallet
	db             walletdb.DB
	mu             sync.Mutex
}

// NewLoader constructs a Loader with an optional recovery window. If the
// recovery window is non-zero, the wallet will attempt to recovery addresses
// starting from the last SyncedTo height.
func NewLoader(chainParams *chaincfg.Params, dbDirPath,
	walletName string, noFreelistSync bool, recoveryWindow uint32) *Loader {

	return &Loader{
		chainParams:    chainParams,
		walletName:     walletName,
		dbDirPath:      dbDirPath,
		noFreelistSync: noFreelistSync,
		recoveryWindow: recoveryWindow,
	}
}

// onLoaded executes each added callback and prevents loader from loading any
// additional wallets.  Requires mutex to be locked.
func (l *Loader) onLoaded(w *Wallet, db walletdb.DB) {
	for _, fn := range l.callbacks {
		fn(w)
	}

	l.wallet = w
	l.db = db
	l.callbacks = nil // not needed anymore
}

// RunAfterLoad adds a function to be executed when the loader creates or opens
// a wallet.  Functions are executed in a single goroutine in the order they are
// added.
func (l *Loader) RunAfterLoad(fn func(*Wallet)) {
	l.mu.Lock()
	if l.wallet != nil {
		w := l.wallet
		l.mu.Unlock()
		fn(w)
	} else {
		l.callbacks = append(l.callbacks, fn)
		l.mu.Unlock()
	}
}

func WalletDbPath(netDir, walletName string) string {
	if strings.HasSuffix(walletName, ".db") {
		if strings.HasPrefix(walletName, "/") {
			// absolute path
			return walletName
		} else {
			return filepath.Join(netDir, walletName)
		}
	} else {
		return filepath.Join(netDir, fmt.Sprintf("wallet_%s.db", walletName))
	}
}

// CreateNewWallet creates a new wallet using the provided public and private
// passphrases.  The seed is optional.  If non-nil, addresses are derived from
// this seed.  If nil, a secure random seed is generated.
func (l *Loader) CreateNewWallet(pubPassphrase, privPassphrase []byte,
	seedInput []byte, seed *seedwords.Seed) (*Wallet, er.R) {

	defer l.mu.Unlock()
	l.mu.Lock()

	if l.wallet != nil {
		return nil, ErrLoaded.Default()
	}

	dbPath := WalletDbPath(l.dbDirPath, l.walletName)
	exists, err := fileExists(dbPath)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrExists.Default()
	}

	// Create the wallet database backed by bolt db.
	err = er.E(os.MkdirAll(l.dbDirPath, 0700))
	if err != nil {
		return nil, err
	}
	opts := &bbolt.Options{
		NoFreelistSync:  true,
		FreelistType:    bbolt.FreelistMapType,
	}
	db, err := bdb.OpenDB(dbPath, true, opts)
	if err != nil {
		return nil, err
	}

	// Initialize the newly created database for the wallet before opening.
	err = Create(db, pubPassphrase, privPassphrase, seedInput, seed, l.chainParams)
	if err != nil {
		return nil, err
	}

	// Open the newly-created wallet.
	w, err := Open(db, pubPassphrase, nil, l.chainParams, l.recoveryWindow)
	if err != nil {
		return nil, err
	}
	w.Start()

	l.onLoaded(w, db)
	return w, nil
}

func noConsole() ([]byte, er.R) {
	return nil, er.New("db upgrade requires console access for additional input")
}

// OpenExistingWallet opens the wallet from the loader's wallet database path
// and the public passphrase.  If the loader is being called by a context where
// standard input prompts may be used during wallet upgrades, setting
// canConsolePrompt will enables these prompts.
func (l *Loader) OpenExistingWallet(pubPassphrase []byte, canConsolePrompt bool) (*Wallet, er.R) {
	defer l.mu.Unlock()
	l.mu.Lock()

	if l.wallet != nil {
		return nil, ErrLoaded.Default()
	}

	// Ensure that the network directory exists.
	if err := checkCreateDir(l.dbDirPath); err != nil {
		return nil, err
	}

		var dbFileSize int64
		var opts *bbolt.Options

	// Open the database using the boltdb backend.
	dbPath := WalletDbPath(l.dbDirPath, l.walletName)
    exists, err := fileExists(dbPath)
    if err != nil {
        return nil, err
    }
    if exists {
		dbFileInfo, _ := os.Stat(dbPath)
		dbFileSize = int64(dbFileInfo.Size())
		opts = &bbolt.Options{
			NoFreelistSync:  true,
			InitialMmapSize: int(math.Ceil(float64(dbFileSize) * 2.5)),
			FreelistType:    bbolt.FreelistMapType,
		}
	} else {
		return nil, err
	}
	db, err := bdb.OpenDB(dbPath, false, opts)
	if err != nil {
		log.Errorf("Failed to open database: %v", err)
		return nil, err
	}

	var cbs *waddrmgr.OpenCallbacks
	if canConsolePrompt {
		cbs = &waddrmgr.OpenCallbacks{
			ObtainSeed:        prompt.ProvideSeed,
			ObtainPrivatePass: prompt.ProvidePrivPassphrase,
		}
	} else {
		cbs = &waddrmgr.OpenCallbacks{
			ObtainSeed:        noConsole,
			ObtainPrivatePass: noConsole,
		}
	}
	w, err := Open(db, pubPassphrase, cbs, l.chainParams, l.recoveryWindow)
	if err != nil {
		// If opening the wallet fails (e.g. because of wrong
		// passphrase), we must close the backing database to
		// allow future calls to walletdb.Open().
		e := db.Close()
		if e != nil {
			log.Warnf("Error closing database: %v", e)
		}
		return nil, err
	}
	w.Start()

	l.onLoaded(w, db)
	return w, nil
}

// LoadedWallet returns the loaded wallet, if any, and a bool for whether the
// wallet has been loaded or not.  If true, the wallet pointer should be safe to
// dereference.
func (l *Loader) LoadedWallet() (*Wallet, bool) {
	l.mu.Lock()
	w := l.wallet
	l.mu.Unlock()
	return w, w != nil
}

// UnloadWallet stops the loaded wallet, if any, and closes the wallet database.
// This returns ErrNotLoaded if the wallet has not been loaded with
// CreateNewWallet or LoadExistingWallet.  The Loader may be reused if this
// function returns without error.
func (l *Loader) UnloadWallet() er.R {
	defer l.mu.Unlock()
	l.mu.Lock()

	if l.wallet == nil {
		return ErrNotLoaded.Default()
	}

	l.wallet.Stop()
	l.wallet.WaitForShutdown()
	err := l.db.Close()
	if err != nil {
		return err
	}

	l.wallet = nil
	l.db = nil
	return nil
}

func fileExists(filePath string) (bool, er.R) {
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, er.E(err)
	}
	return true, nil
}

// Copyright (c) 2014-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"encoding/hex"
	"github.com/json-iterator/go"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/wire/protocol"

	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/pktwallet/internal/legacy/keystore"
	"github.com/pkt-cash/pktd/pktwallet/internal/prompt"
	"github.com/pkt-cash/pktd/pktwallet/internal/zero"
	"github.com/pkt-cash/pktd/pktwallet/waddrmgr"
	"github.com/pkt-cash/pktd/pktwallet/wallet"
	"github.com/pkt-cash/pktd/pktwallet/wallet/seedwords"
	"github.com/pkt-cash/pktd/pktwallet/walletdb/bdb"
	"go.etcd.io/bbolt"
)

// networkDir returns the directory name of a network directory to hold wallet
// files.
func networkDir(dataDir string, chainParams *chaincfg.Params) string {
	netname := chainParams.Name

	// For now, we must always name the testnet data directory as "testnet"
	// and not "testnet3" or any other version, as the chaincfg testnet3
	// paramaters will likely be switched to being named "testnet3" in the
	// future.  This is done to future proof that change, and an upgrade
	// plan to move the testnet3 data directory can be worked out later.
	if chainParams.Net == protocol.TestNet3 {
		netname = "testnet"
	}

	return filepath.Join(dataDir, netname)
}

// convertLegacyKeystore converts all of the addresses in the passed legacy
// key store to the new waddrmgr.Manager format.  Both the legacy keystore and
// the new manager must be unlocked.
func convertLegacyKeystore(legacyKeyStore *keystore.Store, w *wallet.Wallet) er.R {
	netParams := legacyKeyStore.Net()
	blockStamp := waddrmgr.BlockStamp{
		Height: 0,
		Hash:   *netParams.GenesisHash,
	}
	for _, walletAddr := range legacyKeyStore.ActiveAddresses() {
		switch addr := walletAddr.(type) {
		case keystore.PubKeyAddress:
			privKey, err := addr.PrivKey()
			if err != nil {
				fmt.Printf("WARN: Failed to obtain private key "+
					"for address %v: %v\n", addr.Address(),
					err)
				continue
			}

			wif, err := btcutil.NewWIF((*btcec.PrivateKey)(privKey),
				netParams, addr.Compressed())
			if err != nil {
				fmt.Printf("WARN: Failed to create wallet "+
					"import format for address %v: %v\n",
					addr.Address(), err)
				continue
			}

			_, err = w.ImportPrivateKey(waddrmgr.KeyScopeBIP0044,
				wif, &blockStamp, false)
			if err != nil {
				fmt.Printf("WARN: Failed to import private "+
					"key for address %v: %v\n",
					addr.Address(), err)
				continue
			}

		case keystore.ScriptAddress:
			_, err := w.ImportP2SHRedeemScript(addr.Script())
			if err != nil {
				fmt.Printf("WARN: Failed to import "+
					"pay-to-script-hash script for "+
					"address %v: %v\n", addr.Address(), err)
				continue
			}

		default:
			fmt.Printf("WARN: Skipping unrecognized legacy "+
				"keystore type: %T\n", addr)
			continue
		}
	}

	return nil
}

type WalletSetupCfg struct {
	Passphrase       *string `json:"passphrase"`
	PublicPassphrase *string `json:"viewpassphrase"`
	Seed             *string `json:"seed"`
	SeedPassphrase   *string `json:"seedpassphrase"`
}

// createWallet prompts the user for information needed to generate a new wallet
// and generates the wallet accordingly.  The new wallet will reside at the
// provided path.
func createWallet(cfg *config) er.R {
	dbDir := networkDir(cfg.AppDataDir.Value, activeNet.Params)
	loader := wallet.NewLoader(activeNet.Params, dbDir, cfg.Wallet, true, 250)

	// When there is a legacy keystore, open it now to ensure any errors
	// don't end up exiting the process after the user has spent time
	// entering a bunch of information.
	netDir := networkDir(cfg.AppDataDir.Value, activeNet.Params)
	keystorePath := filepath.Join(netDir, keystore.Filename)
	var legacyKeyStore *keystore.Store
	_, errr := os.Stat(keystorePath)
	if errr != nil && !os.IsNotExist(errr) {
		// A stat error not due to a non-existant file should be
		// returned to the caller.
		return er.E(errr)
	} else if errr == nil {
		// Keystore file exists.
		var err er.R
		legacyKeyStore, err = keystore.OpenDir(netDir)
		if err != nil {
			return err
		}
	}

	fi, err := os.Stdin.Stat()
	if err != nil {
		panic("createWallet: os.Stdin.Stat failure.")
	}
	tty := false
	var privPass []byte
	pubPass := []byte(wallet.InsecurePubPassphrase)
	var seedInput []byte
	var seed *seedwords.Seed
	setupCfg := WalletSetupCfg{}
	if (fi.Mode() & os.ModeCharDevice) != 0 {
		tty = true
	} else if bytes, err := ioutil.ReadAll(os.Stdin); err != nil {
		return er.E(err)
	} else if err := jsoniter.Unmarshal(bytes, &setupCfg); err != nil {
		return er.E(err)
	} else {
		if setupCfg.Passphrase != nil {
			privPass = []byte(*setupCfg.Passphrase)
		}
		if setupCfg.PublicPassphrase != nil {
			pubPass = []byte(*setupCfg.PublicPassphrase)
		}
		if setupCfg.Seed != nil {
			if decoded, err := hex.DecodeString(*setupCfg.Seed); err == nil {
				zero.Bytes(decoded)
				seedInput = []byte(*setupCfg.Seed)
			} else {
				seedEnc, err := seedwords.SeedFromWords(*setupCfg.Seed)
				if err != nil {
					return err
				}
				var bs []byte
				if setupCfg.SeedPassphrase != nil {
					bs = []byte(*setupCfg.SeedPassphrase)
				}
				if setupCfg.SeedPassphrase != nil || !seedEnc.NeedsPassphrase() {
					s, err := seedEnc.Decrypt(bs, false)
					if err != nil {
						return err
					}
					seed = s
				} else {
					return er.New("The provided seed requires a passphrase")
				}
			}
		} else {
			if s, err := seedwords.RandomSeed(); err != nil {
				return err
			} else {
				seed = s
			}
		}
	}

	// Start by prompting for the private passphrase.  When there is an
	// existing keystore, the user will be promped for that passphrase,
	// otherwise they will be prompted for a new one.
	var reader *bufio.Reader
	if tty {
		reader = bufio.NewReader(os.Stdin)
		pvt, err := prompt.PrivatePass(reader, legacyKeyStore)
		if err != nil {
			return err
		}
		privPass = pvt
	}

	// When there exists a legacy keystore, unlock it now and set up a
	// callback to import all keystore keys into the new walletdb
	// wallet
	if legacyKeyStore != nil {
		err := legacyKeyStore.Unlock(privPass)
		if err != nil {
			return err
		}

		// Import the addresses in the legacy keystore to the new wallet if
		// any exist, locking each wallet again when finished.
		loader.RunAfterLoad(func(w *wallet.Wallet) {
			defer legacyKeyStore.Lock()

			fmt.Println("Importing addresses from existing wallet...")

			lockChan := make(chan time.Time, 1)
			defer func() {
				lockChan <- time.Time{}
			}()
			err := w.Unlock(privPass, lockChan)
			if err != nil {
				fmt.Printf("ERR: Failed to unlock new wallet "+
					"during old wallet key import: %v", err)
				return
			}

			err = convertLegacyKeystore(legacyKeyStore, w)
			if err != nil {
				fmt.Printf("ERR: Failed to import keys from old "+
					"wallet format: %v", err)
				return
			}

			// Remove the legacy key store.
			errr = os.Remove(keystorePath)
			if errr != nil {
				fmt.Printf("WARN: Failed to remove legacy wallet "+
					"from'%s'\n", keystorePath)
			}
		})
	}

	// Ascertain the public passphrase.  This will either be a value
	// specified by the user or the default hard-coded public passphrase if
	// the user does not want the additional public data encryption.
	if tty {
		pub, err := prompt.PublicPass(reader, privPass,
			[]byte(wallet.InsecurePubPassphrase), []byte(cfg.WalletPass))
		if err != nil {
			return err
		}
		pubPass = pub
	}

	// Ascertain the wallet generation seed.  This will either be an
	// automatically generated value the user has already confirmed or a
	// value the user has entered which has already been validated.
	if tty {
		si, sd, err := prompt.Seed(reader, privPass)
		if err != nil {
			return err
		}
		seedInput = si
		seed = sd
	}

	if tty {
		fmt.Println("Creating the wallet...")
	}
	w, werr := loader.CreateNewWallet(pubPass, privPass, seedInput, seed)
	if werr != nil {
		return werr
	}

	w.Manager.Close()
	if tty {
		fmt.Println("The wallet has been created successfully.")
	} else if seed != nil {
		seedEnc := seed.Encrypt(privPass)
		if words, err := seedEnc.Words("english"); err != nil {
			return err
		} else {
			fmt.Printf(`{"seed":"%s"}`+"\n", words)
		}
		seedEnc.Zero()
	} else {
		fmt.Printf(`{"seed":"%s"}`+"\n", seedInput)
	}
	if seed != nil {
		seed.Zero()
	}
	return nil
}

// createSimulationWallet is intended to be called from the rpcclient
// and used to create a wallet for actors involved in simulations.
func createSimulationWallet(cfg *config) er.R {
	// Simulation wallet password is 'password'.
	privPass := []byte("password")

	// Public passphrase is the default.
	pubPass := []byte(wallet.InsecurePubPassphrase)

	netDir := networkDir(cfg.AppDataDir.Value, activeNet.Params)

	// Create the wallet.
	dbPath := wallet.WalletDbPath(netDir, cfg.Wallet)
	fmt.Println("Creating the wallet...")

	// Create the wallet database backed by bolt db.
	opts := &bbolt.Options{
		NoFreelistSync:  true,
		InitialMmapSize: 1024 * 1024 * 1024 * 2,
		FreelistType:    bbolt.FreelistMapType,
	}
	db, err := bdb.OpenDB(dbPath, true, opts)
	if err != nil {
		return err
	}
	defer db.Close()

	seed, err := seedwords.RandomSeed()
	if err != nil {
		return err
	}

	// Create the wallet.
	err = wallet.Create(db, pubPass, privPass, nil, seed, activeNet.Params)
	if err != nil {
		return err
	}

	fmt.Println("The wallet has been created successfully.")
	return nil
}

// checkCreateDir checks that the path exists and is a directory.
// If path does not exist, it is created.
func checkCreateDir(path string) er.R {
	if fi, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			// Attempt data directory creation
			if err = os.MkdirAll(path, 0700); err != nil {
				return er.Errorf("cannot create directory: %s", err)
			}
		} else {
			return er.Errorf("error checking directory: %s", err)
		}
	} else {
		if !fi.IsDir() {
			return er.Errorf("path '%s' is not a directory", path)
		}
	}

	return nil
}

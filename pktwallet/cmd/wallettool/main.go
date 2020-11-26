// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jessevdk/go-flags"
	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/btcutil/util"
	"github.com/pkt-cash/pktd/pktconfig/version"
	"github.com/pkt-cash/pktd/pktwallet/walletdb"
	"github.com/pkt-cash/pktd/pktwallet/walletdb/bdb"
	"go.etcd.io/bbolt"
)

const defaultNet = "pkt"

var datadir = btcutil.AppDataDir("pktwallet", false)

// Flags.
var opts = struct {
	DbPath string `long:"db" description:"Path to wallet database"`
}{
	DbPath: filepath.Join(datadir, defaultNet, "wallet.db"),
}

func main() {
	version.SetUserAgentName("wallettool")
	os.Exit(mainInt())
}

const maxCount = 100

func print0(depth int, b walletdb.ReadBucket) er.R {
	binaryKey := false
	binaryVal := false
	count := 0
	err := b.ForEach(func(k, v []byte) er.R {
		count++
		if !utf8.Valid(k) {
			binaryKey = true
		}
		if v != nil && !utf8.Valid(v) {
			binaryVal = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	pad := strings.Repeat(" ", depth)
	if binaryKey && count > maxCount {
		fmt.Printf("%s \"_\": \"(%d entries)\",\n", pad, count)
		return nil
	}
	return b.ForEach(func(k, v []byte) er.R {
		cb := b.NestedReadBucket(k)
		if binaryKey && count > maxCount {
		} else {
			ks := ""
			if binaryKey {
				ks = fmt.Sprintf("hex(\"%s\")", hex.EncodeToString(k))
			} else {
				ks = strconv.QuoteToASCII(string(k))
			}
			if cb != nil && len(v) == 0 {
				fmt.Printf("%s%v:", pad, ks)
			} else if binaryVal {
				fmt.Printf("%s%v: hex(\"%s\")", pad, ks, hex.EncodeToString(v))
			} else {
				vs := strconv.QuoteToASCII(string(v))
				fmt.Printf("%s%v: %v", pad, ks, vs)
			}
		}
		if cb != nil {
			fmt.Printf(" {\n")
			ret := print0(depth+2, cb)
			fmt.Printf("%s},\n", pad)
			return ret
		} else {
			fmt.Printf(",\n")
		}
		return nil
	})
}

func wprint(db walletdb.DB) er.R {
	return walletdb.View(db, func(tx walletdb.ReadTx) er.R {
		return print0(0, tx.ReadBucket(nil))
	})
}

func repair1(to walletdb.ReadWriteBucket, from walletdb.ReadBucket) er.R {
	return from.ForEach(func(k, v []byte) er.R {
		fromB := from.NestedReadBucket(k)
		if fromB != nil {
			if len(v) > 0 {
				panic("Value and nested bucket both")
			}
			toB, err := to.CreateBucket(k)
			if err != nil {
				return err
			}
			return repair1(toB, fromB)
		}
		return to.Put(k, v)
	})
}

func repair0(temppath string, db walletdb.DB) er.R {
	backupPath := fmt.Sprintf("%s.repair_backup", opts.DbPath)
	if util.Exists(backupPath) {
		return er.Errorf("%s exists so no place to put the backup", backupPath)
	}
    bboltopts := &bbolt.Options{
		NoFreelistSync:  true,
		FreelistType:    bbolt.FreelistMapType,
    }
	toDb, err := bdb.OpenDB(temppath, true, bboltopts)
	if err != nil {
		return err
	}
	defer toDb.Close()
	err = walletdb.View(db, func(fromTx walletdb.ReadTx) er.R {
		return walletdb.Update(toDb, func(toTx walletdb.ReadWriteTx) er.R {
			return repair1(toTx.ReadWriteBucket(nil), fromTx.ReadBucket(nil))
		})
	})
	if err != nil {
		return err
	}
	err = er.E(os.Rename(opts.DbPath, backupPath))
	if err != nil {
		return err
	}
	err = er.E(os.Rename(temppath, opts.DbPath))
	if err != nil {
		return err
	}
	fmt.Println("Ok")
	return nil
}

func repair(db walletdb.DB) er.R {
	temppath := fmt.Sprintf("%s.repaired_%d", opts.DbPath, time.Now().UnixNano())
	err := repair0(temppath, db)
	if err != nil {
		// if we fail to remove, ignore since there was already an error
		os.Remove(temppath)
	}
	return err
}

var ops = map[string]func(db walletdb.DB) er.R{
	"print":  wprint,
	"repair": repair,
}

func mainInt() int {
	args, errr := flags.Parse(&opts)
	if errr != nil {
		return 1
	}
	if len(args) != 1 || ops[args[0]] == nil {
		fmt.Println("Usage: wallettool [--db <path_to_wallet.db>] COMMAND")
		fmt.Println("    print             # print some of the decodable keys from the wallet")
		fmt.Println("    repair            # attempt to repair the wallet")
		return 1
	}

	if !util.Exists(opts.DbPath) {
		fmt.Println("Database file does not exist")
		return 1
	}
	bboltopts := &bbolt.Options{
		NoFreelistSync:  true,
		FreelistType:    bbolt.FreelistMapType,
	}
	db, err := bdb.OpenDB(opts.DbPath, false, bboltopts)
	if err != nil {
		fmt.Println("Failed to open database:", err)
		return 1
	}
	defer db.Close()

	err = ops[args[0]](db)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	return 0
}

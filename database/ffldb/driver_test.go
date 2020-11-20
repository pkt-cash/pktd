// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ffldb_test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/chaincfg/genesis"
	"github.com/pkt-cash/pktd/database"
	"github.com/pkt-cash/pktd/database/ffldb"
)

// TestPersistence ensures that values stored are still valid after closing and
// reopening the database.
func TestPersistence(t *testing.T) {

	// Create a new database to run tests against.
	dbPath := filepath.Join(os.TempDir(), "ffldb-persistencetest")
	_ = os.RemoveAll(dbPath)
	db, err := ffldb.OpenDB(dbPath, blockDataNet, true)
	if err != nil {
		t.Errorf("Failed to create test database: %v", err)
		return
	}
	defer os.RemoveAll(dbPath)
	defer db.Close()

	// Create a bucket, put some values into it, and store a block so they
	// can be tested for existence on re-open.
	bucket1Key := []byte("bucket1")
	storeValues := map[string]string{
		"b1key1": "foo1",
		"b1key2": "foo2",
		"b1key3": "foo3",
	}
	genesisBlock := btcutil.NewBlock(genesis.Block(chaincfg.MainNetParams.GenesisHash))
	genesisHash := chaincfg.MainNetParams.GenesisHash
	err = db.Update(func(tx database.Tx) er.R {
		metadataBucket := tx.Metadata()
		if metadataBucket == nil {
			return er.Errorf("Metadata: unexpected nil bucket")
		}

		bucket1, err := metadataBucket.CreateBucket(bucket1Key)
		if err != nil {
			return er.Errorf("CreateBucket: unexpected error: %v",
				err)
		}

		for k, v := range storeValues {
			err := bucket1.Put([]byte(k), []byte(v))
			if err != nil {
				return er.Errorf("Put: unexpected error: %v",
					err)
			}
		}

		if err := tx.StoreBlock(genesisBlock); err != nil {
			return er.Errorf("StoreBlock: unexpected error: %v",
				err)
		}

		return nil
	})
	if err != nil {
		t.Errorf("Update: unexpected error: %v", err)
		return
	}

	// Close and reopen the database to ensure the values persist.
	db.Close()
	db, err = ffldb.OpenDB(dbPath, blockDataNet, false)
	if err != nil {
		t.Errorf("Failed to open test database: %v", err)
		return
	}
	defer db.Close()

	// Ensure the values previously stored in the 3rd namespace still exist
	// and are correct.
	err = db.View(func(tx database.Tx) er.R {
		metadataBucket := tx.Metadata()
		if metadataBucket == nil {
			return er.Errorf("Metadata: unexpected nil bucket")
		}

		bucket1 := metadataBucket.Bucket(bucket1Key)
		if bucket1 == nil {
			return er.Errorf("Bucket1: unexpected nil bucket")
		}

		for k, v := range storeValues {
			gotVal := bucket1.Get([]byte(k))
			if !reflect.DeepEqual(gotVal, []byte(v)) {
				return er.Errorf("Get: key '%s' does not "+
					"match expected value - got %s, want %s",
					k, gotVal, v)
			}
		}

		genesisBlockBytes, _ := genesisBlock.Bytes()
		gotBytes, err := tx.FetchBlock(genesisHash)
		if err != nil {
			return er.Errorf("FetchBlock: unexpected error: %v",
				err)
		}
		if !reflect.DeepEqual(gotBytes, genesisBlockBytes) {
			return er.Errorf("FetchBlock: stored block mismatch")
		}

		return nil
	})
	if err != nil {
		t.Errorf("View: unexpected error: %v", err)
		return
	}
}

// TestInterface performs all interfaces tests for this database driver.
func TestInterface(t *testing.T) {

	// Create a new database to run tests against.
	dbPath := filepath.Join(os.TempDir(), "ffldb-interfacetest")
	_ = os.RemoveAll(dbPath)
	db, err := ffldb.OpenDB(dbPath, blockDataNet, true)
	if err != nil {
		t.Errorf("Failed to create test database %v", err)
		return
	}
	defer os.RemoveAll(dbPath)
	defer db.Close()

	// Run all of the interface tests against the database.
	runtime.GOMAXPROCS(runtime.NumCPU()*6)

	// Change the maximum file size to a small value to force multiple flat
	// files with the test data set.
	ffldb.TstRunWithMaxBlockFileSize(db, 2048, func() {
		testInterface(t, db)
	})
}

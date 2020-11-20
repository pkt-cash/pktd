// Copyright (c) 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// This interface was inspired heavily by the excellent boltdb
// project at https://github.com/boltdb/bolt by Ben B. Johnson.

package walletdb

import (
	//"go.etcd.io/bbolt"

	"io"

	"github.com/pkt-cash/pktd/btcutil/er"
)

// ReadTx represents a database transaction that can only be used for reads.  If
// a database update must occur, use a ReadWriteTx.
type ReadTx interface {
	// ReadBucket opens the root bucket for read only access.  If the bucket
	// described by the key does not exist, nil is returned.
	ReadBucket(key []byte) ReadBucket

	// Rollback closes the transaction, discarding changes (if any) if the
	// database was modified by a write transaction.
	Rollback() er.R
}

// ReadWriteTx represents a database transaction that can be used for both reads
// and writes.  When only reads are necessary, consider using a ReadTx instead.
type ReadWriteTx interface {
	ReadTx

	// ReadWriteBucket opens the root bucket for read/write access.  If the
	// bucket described by the key does not exist, nil is returned.
	ReadWriteBucket(key []byte) ReadWriteBucket

	// CreateTopLevelBucket creates the top level bucket for a key if it
	// does not exist.  The newly-created bucket it returned.
	CreateTopLevelBucket(key []byte) (ReadWriteBucket, er.R)

	// DeleteTopLevelBucket deletes the top level bucket for a key.  This
	// errors if the bucket can not be found or the key keys a single value
	// instead of a bucket.
	DeleteTopLevelBucket(key []byte) er.R

	// Commit commits all changes that have been on the transaction's root
	// buckets and all of their sub-buckets to persistent storage.
	Commit() er.R

	// OnCommit takes a function closure that will be executed when the
	// transaction successfully gets committed.
	OnCommit(func())
}

// ReadBucket represents a bucket (a hierarchical structure within the database)
// that is only allowed to perform read operations.
type ReadBucket interface {
	// NestedReadBucket retrieves a nested bucket with the given key.
	// Returns nil if the bucket does not exist.
	NestedReadBucket(key []byte) ReadBucket

	// ForEach invokes the passed function with every key/value pair in
	// the bucket.  This includes nested buckets, in which case the value
	// is nil, but it does not include the key/value pairs within those
	// nested buckets.
	//
	// NOTE: The values returned by this function are only valid during a
	// transaction.  Attempting to access them after a transaction has ended
	// results in undefined behavior.  This constraint prevents additional
	// data copies and allows support for memory-mapped database
	// implementations.
	ForEachBeginningWith([]byte, func(k, v []byte) er.R) er.R
	ForEach(func(k, v []byte) er.R) er.R

	// Get returns the value for the given key.  Returns nil if the key does
	// not exist in this bucket (or nested buckets).
	//
	// NOTE: The value returned by this function is only valid during a
	// transaction.  Attempting to access it after a transaction has ended
	// results in undefined behavior.  This constraint prevents additional
	// data copies and allows support for memory-mapped database
	// implementations.
	Get(key []byte) []byte

	ReadCursor() ReadCursor
}

// ReadWriteBucket represents a bucket (a hierarchical structure within the
// database) that is allowed to perform both read and write operations.
type ReadWriteBucket interface {
	ReadBucket

	// NestedReadWriteBucket retrieves a nested bucket with the given key.
	// Returns nil if the bucket does not exist.
	NestedReadWriteBucket(key []byte) ReadWriteBucket

	// CreateBucket creates and returns a new nested bucket with the given
	// key.  Returns ErrBucketExists if the bucket already exists,
	// ErrBucketNameRequired if the key is empty, or ErrIncompatibleValue
	// if the key value is otherwise invalid for the particular database
	// implementation.  Other errors are possible depending on the
	// implementation.
	CreateBucket(key []byte) (ReadWriteBucket, er.R)

	// CreateBucketIfNotExists creates and returns a new nested bucket with
	// the given key if it does not already exist.  Returns
	// ErrBucketNameRequired if the key is empty or ErrIncompatibleValue
	// if the key value is otherwise invalid for the particular database
	// backend.  Other errors are possible depending on the implementation.
	CreateBucketIfNotExists(key []byte) (ReadWriteBucket, er.R)

	// DeleteNestedBucket removes a nested bucket with the given key.
	// Returns ErrTxNotWritable if attempted against a read-only transaction
	// and ErrBucketNotFound if the specified bucket does not exist.
	DeleteNestedBucket(key []byte) er.R

	// Put saves the specified key/value pair to the bucket.  Keys that do
	// not already exist are added and keys that already exist are
	// overwritten.  Returns ErrTxNotWritable if attempted against a
	// read-only transaction.
	Put(key, value []byte) er.R

	// Delete removes the specified key from the bucket.  Deleting a key
	// that does not exist does not return an error.  Returns
	// ErrTxNotWritable if attempted against a read-only transaction.
	Delete(key []byte) er.R

	// Cursor returns a new cursor, allowing for iteration over the bucket's
	// key/value pairs and nested buckets in forward or backward order.
	ReadWriteCursor() ReadWriteCursor

	// Tx returns the bucket's transaction.
	Tx() ReadWriteTx
}

// ReadCursor represents a bucket cursor that can be positioned at the start or
// end of the bucket's key/value pairs and iterate over pairs in the bucket.
// This type is only allowed to perform database read operations.
type ReadCursor interface {
	// First positions the cursor at the first key/value pair and returns
	// the pair.
	First() (key, value []byte)

	// Last positions the cursor at the last key/value pair and returns the
	// pair.
	Last() (key, value []byte)

	// Next moves the cursor one key/value pair forward and returns the new
	// pair.
	Next() (key, value []byte)

	// Prev moves the cursor one key/value pair backward and returns the new
	// pair.
	Prev() (key, value []byte)

	// Seek positions the cursor at the passed seek key.  If the key does
	// not exist, the cursor is moved to the next key after seek.  Returns
	// the new pair.
	Seek(seek []byte) (key, value []byte)
}

// ReadWriteCursor represents a bucket cursor that can be positioned at the
// start or end of the bucket's key/value pairs and iterate over pairs in the
// bucket.  This abstraction is allowed to perform both database read and write
// operations.
type ReadWriteCursor interface {
	ReadCursor

	// Delete removes the current key/value pair the cursor is at without
	// invalidating the cursor.  Returns ErrIncompatibleValue if attempted
	// when the cursor points to a nested bucket.
	Delete() er.R
}

// DB represents an ACID database.  All database access is performed through
// read or read+write transactions.
type DB interface {
	// BeginReadTx opens a database read transaction.
	BeginReadTx() (ReadTx, er.R)

	// BeginReadWriteTx opens a database read+write transaction.
	BeginReadWriteTx() (ReadWriteTx, er.R)

	// Copy writes a copy of the database to the provided writer.  This
	// call will start a read-only transaction to perform all operations.
	Copy(w io.Writer) er.R

	// Close cleanly shuts down the database and syncs all data.
	Close() er.R
}

// View opens a database read transaction and executes the function f with the
// transaction passed as a parameter.  After f exits, the transaction is rolled
// back.  If f errors, its error is returned, not a rollback error (if any
// occur).
func View(db DB, f func(tx ReadTx) er.R) er.R {
	tx, err := db.BeginReadTx()
	if err != nil {
		return err
	}
	// Make sure the transaction rolls back in the event of a panic.
	defer func() {
        if tx != nil {
                tx.Rollback()
        }
	}()
	err = f(tx)
	rollbackErr := tx.Rollback()
	if err != nil {
		return err
	}
	if rollbackErr != nil {
		return rollbackErr
	}
	return nil
}

// Update opens a database read/write transaction and executes the function f
// with the transaction passed as a parameter.  After f exits, if f did not
// error, the transaction is committed.  Otherwise, if f did error, the
// transaction is rolled back.  If the rollback fails, the original error
// returned by f is still returned.  If the commit fails, the commit error is
// returned.
func Update(db DB, f func(tx ReadWriteTx) er.R) er.R {
	tx, err := db.BeginReadWriteTx()
	if err != nil {
		return err
	}
	// Make sure the transaction rolls back in the event of a panic.
	defer func() {
        if tx != nil {
                tx.Rollback()
        }
	}()
	err = f(tx)
	if err != nil {
		// Want to return the original error, not a rollback error if
		// any occur.
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Create intializes and opens a database for the specified type.  The arguments
// are specific to the database type driver.  See the documentation for the
// database driver for further details.
//func Create(dbPath string, options *bbolt.Options) (DB, er.R) {
//	return OpenDB(dbPath, true, options)
//}

// Open opens an existing database for the specified type.  The arguments are
// specific to the database type driver.  See the documentation for the database
// driver for further details.
//func Open(dbPath string, options *bbolt.Options) (DB, er.R) {
//	return OpenDB(dbPath, false, options)
//}

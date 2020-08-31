// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package database

import (
	"github.com/pkt-cash/pktd/btcutil/er"
)

// Err identifies a kind of error for the database
var Err er.ErrorType = er.NewErrorType("database.Err")

// These constants are used to identify a specific database Error.
var (
	// **************************************
	// Errors related to driver registration.
	// **************************************

	// ErrDbTypeRegistered indicates two different database drivers
	// attempt to register with the name database type.
	ErrDbTypeRegistered = Err.Code("ErrDbTypeRegistered")

	// *************************************
	// Errors related to database functions.
	// *************************************

	// ErrDbUnknownType indicates there is no driver registered for
	// the specified database type.
	ErrDbUnknownType = Err.Code("ErrDbUnknownType")

	// ErrDbDoesNotExist indicates open is called for a database that
	// does not exist.
	ErrDbDoesNotExist = Err.Code("ErrDbDoesNotExist")

	// ErrDbNotOpen indicates a database instance is accessed before
	// it is opened or after it is closed.
	ErrDbNotOpen = Err.Code("ErrDbNotOpen")

	// ErrCorruption indicates a checksum failure occurred which invariably
	// means the database is corrupt.
	ErrCorruption = Err.Code("ErrCorruption")

	// ****************************************
	// Errors related to database transactions.
	// ****************************************

	// ErrTxClosed indicates an attempt was made to commit or rollback a
	// transaction that has already had one of those operations performed.
	ErrTxClosed = Err.Code("ErrTxClosed")

	// ErrTxNotWritable indicates an operation that requires write access to
	// the database was attempted against a read-only transaction.
	ErrTxNotWritable = Err.Code("ErrTxNotWritable")

	// ErrAvailableDiskSpace indicates that the user is running out of
	// disk space.  The database preventively decided to not allow the
	// transaction to prevent causing hard-to-detect problems.
	ErrAvailableDiskSpace = Err.Code("ErrAvailableDiskSpace")

	// **************************************
	// Errors related to metadata operations.
	// **************************************

	// ErrBucketNotFound indicates an attempt to access a bucket that has
	// not been created yet.
	ErrBucketNotFound = Err.Code("ErrBucketNotFound")

	// ErrBucketExists indicates an attempt to create a bucket that already
	// exists.
	ErrBucketExists = Err.Code("ErrBucketExists")

	// ErrBucketNameRequired indicates an attempt to create a bucket with a
	// blank name.
	ErrBucketNameRequired = Err.Code("ErrBucketNameRequired")

	// ErrKeyRequired indicates at attempt to insert a zero-length key.
	ErrKeyRequired = Err.Code("ErrKeyRequired")

	// ErrIncompatibleValue indicates the value in question is invalid for
	// the specific requested operation.  For example, trying create or
	// delete a bucket with an existing non-bucket key, attempting to create
	// or delete a non-bucket key with an existing bucket key, or trying to
	// delete a value via a cursor when it points to a nested bucket.
	ErrIncompatibleValue = Err.Code("ErrIncompatibleValue")

	// ***************************************
	// Errors related to block I/O operations.
	// ***************************************

	// ErrBlockNotFound indicates a block with the provided hash does not
	// exist in the database.
	ErrBlockNotFound = Err.Code("ErrBlockNotFound")

	// ErrBlockExists indicates a block with the provided hash already
	// exists in the database.
	ErrBlockExists = Err.Code("ErrBlockExists")

	// ErrBlockRegionInvalid indicates a region that exceeds the bounds of
	// the specified block was requested.  When the hash provided by the
	// region does not correspond to an existing block, the error will be
	// ErrBlockNotFound instead.
	ErrBlockRegionInvalid = Err.Code("ErrBlockRegionInvalid")

	// ***********************************
	// Support for driver-specific errors.
	// ***********************************

	// ErrDriverSpecific indicates the Err field is a driver-specific error.
	// This provides a mechanism for drivers to plug-in their own custom
	// errors for any situations which aren't already covered by the error
	// codes provided by this package.
	ErrDriverSpecific = Err.Code("ErrDriverSpecific")
)

// makeError creates an Error given a set of arguments.  The error code must
// be one of the error codes provided by this package.
func makeError(c *er.ErrorCode, desc string, err er.R) er.R {
	return c.New(desc, err)
}

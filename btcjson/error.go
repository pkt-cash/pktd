// Copyright (c) 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcjson

import (
	"github.com/pkt-cash/pktd/btcutil/er"
)

// ErrorCode identifies a kind of error.  These error codes are NOT used for
// JSON-RPC response errors.
// type ErrorCode int

// InternalErr is a type for the json errors which are not response types
var InternalErr er.ErrorType = er.NewErrorType("btcjson.InternalErr")

var (
	ErrDuplicateMethod = InternalErr.CodeWithDetail("ErrDuplicateMethod",
		"a command with the specified method already exists.")

	ErrInvalidUsageFlags = InternalErr.CodeWithDetail("ErrInvalidUsageFlags",
		"one or more unrecognized flag bits were specified.")

	ErrInvalidType = InternalErr.CodeWithDetail("ErrInvalidType",
		"a type was passed that is not the required type")

	ErrEmbeddedType = InternalErr.CodeWithDetail("ErrEmbeddedType",
		"provided command struct contains an embedded type which is not not supported")

	ErrUnexportedField = InternalErr.CodeWithDetail("ErrUnexportedField",
		"provided command struct contains an unexported field which is not supported")

	ErrUnsupportedFieldType = InternalErr.CodeWithDetail("ErrUnsupportedFieldType",
		"the type of a field in the provided command struct is not one of the supported types")

	ErrNonOptionalField = InternalErr.CodeWithDetail("ErrNonOptionalField",
		"a non-optional field was specified after an optional field")

	ErrNonOptionalDefault = InternalErr.CodeWithDetail("ErrNonOptionalDefault",
		"a 'jsonrpcdefault' struct tag was specified for a non-optional field")

	ErrMismatchedDefault = InternalErr.CodeWithDetail("ErrMismatchedDefault",
		"a 'jsonrpcdefault' struct tag contains a value that doesn't match the type of the field")

	ErrUnregisteredMethod = InternalErr.CodeWithDetail("ErrUnregisteredMethod",
		"a method was specified that has not been registered")

	ErrMissingDescription = InternalErr.CodeWithDetail("ErrMissingDescription",
		"a description required to generate help is missing")

	ErrNumParams = InternalErr.CodeWithDetail("ErrNumParams",
		"the number of params supplied do not match the requirements of the associated command")
)

func makeError(c *er.ErrorCode, str string) er.R {
	return c.New(str, nil)
}

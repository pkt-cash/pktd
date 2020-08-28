// Copyright (c) 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcjson

import "github.com/pkt-cash/pktd/btcutil/er"

// Err is an error type for the btcjson errors
var Err er.ErrorType = er.NewErrorType("btcjson.Err")

// Standard JSON-RPC 2.0 errors.
var (
	ErrRPCInvalidRequest = Err.CodeWithNumber("ErrRPCInvalidRequest", -32600)
	ErrRPCMethodNotFound = Err.CodeWithNumber("ErrRPCMethodNotFound", -32601)
	ErrRPCInvalidParams  = Err.CodeWithNumber("ErrRPCInvalidParams", -32602)
	ErrRPCInternal       = Err.CodeWithNumber("ErrRPCInternal", -32603)
	ErrRPCParse          = Err.CodeWithNumber("ErrRPCParse", -32700)
)

func NewErrRPCInternal() er.R {
	return NewRPCError(ErrRPCInternal, "Internal error", nil)
}

// General application defined JSON errors.
var (
	ErrRPCMisc                = Err.CodeWithNumber("ErrRPCMisc", -1)
	ErrRPCType                = Err.CodeWithNumber("ErrRPCType", -3)
	ErrRPCInvalidAddressOrKey = Err.CodeWithNumber("ErrRPCInvalidAddressOrKey", -5)
	ErrRPCInvalidParameter    = Err.CodeWithNumber("ErrRPCInvalidParameter", -8)
	ErrRPCDatabase            = Err.CodeWithNumber("ErrRPCDatabase", -20)
	ErrRPCDeserialization     = Err.CodeWithNumber("ErrRPCDeserialization", -22)
	ErrRPCVerify              = Err.CodeWithNumber("ErrRPCVerify", -25)
	ErrRPCInWarmup            = Err.CodeWithNumber("RPCErrorCode", -28)
)

// Peer-to-peer client errors.
var (
	ErrRPCClientInInitialDownload = Err.CodeWithNumber("ErrRPCClientInInitialDownload", -10)
	ErrRPCClientNodeNotAdded      = Err.CodeWithNumber("ErrRPCClientNodeNotAdded", -24)
)

// Wallet JSON errors
var (
	ErrRPCWallet                   = Err.CodeWithNumber("ErrRPCWallet", -4)
	ErrRPCWalletInvalidAccountName = Err.CodeWithNumber("ErrRPCWalletInvalidAccountName", -11)
	ErrRPCWalletUnlockNeeded       = Err.CodeWithNumberAndDetail("ErrRPCWalletUnlockNeeded", -13,
		"Enter the wallet passphrase with walletpassphrase first")
	ErrRPCWalletPassphraseIncorrect = Err.CodeWithNumberAndDetail("ErrRPCWalletPassphraseIncorrect", -14,
		"Incorrect passphrase")
)

// Specific Errors related to commands.  These are the ones a user of the RPC
// server are most likely to see.  Generally, the codes should match one of the
// more general errors above.
var (
	ErrRPCBlockNotFound = Err.CodeWithNumber("ErrRPCBlockNotFound", -5)
	ErrRPCDifficulty    = Err.CodeWithNumber("ErrRPCDifficulty", -5)
	ErrRPCOutOfRange    = Err.CodeWithNumber("ErrRPCOutOfRange", -1)
	ErrRPCNoTxInfo      = Err.CodeWithNumberAndDetail("ErrRPCNoTxInfo", -5,
		"No information for transaction")
	ErrRPCNoCFIndex        = Err.CodeWithNumber("ErrRPCNoCFIndex", -5)
	ErrRPCInvalidTxVout    = Err.CodeWithNumber("ErrRPCInvalidTxVout", -5)
	ErrRPCDecodeHexString  = Err.CodeWithNumber("ErrRPCDecodeHexString", -22)
	ErrRPCTxError          = Err.CodeWithNumber("ErrRPCTxError", -25)
	ErrRPCTxRejected       = Err.CodeWithNumber("ErrRPCTxRejected", -26)
	ErrRPCTxAlreadyInChain = Err.CodeWithNumber("ErrRPCTxAlreadyInChain", -27)
)

// Errors that are specific to pktd.
var (
	ErrRPCNoWallet      = Err.CodeWithNumber("ErrRPCNoWallet", -1)
	ErrRPCUnimplemented = Err.CodeWithNumber("ErrRPCUnimplemented", -1)
)

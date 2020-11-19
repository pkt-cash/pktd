// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package legacyrpc

import (
	"github.com/pkt-cash/pktd/btcjson"
	"github.com/pkt-cash/pktd/btcutil/er"
)

func errNeedPositiveMinconf() er.R {
	return btcjson.ErrRPCInvalidParameter.New("minconf must be positive", nil)
}

func errNeedPositiveAmount() er.R {
	return btcjson.ErrRPCInvalidParameter.New("amount must be positive", nil)
}

// DeserializationError describes a failed deserializaion due to bad
// user input.  It corresponds to btcjson.ErrRPCDeserialization.
func errDeserialization(msg string, err er.R) er.R {
	return btcjson.ErrRPCDeserialization.New(msg, err)
}

// ParseError describes a failed parse due to bad user input.  It
// corresponds to btcjson.ErrRPCParse.
func errParse(msg string, err er.R) er.R {
	return btcjson.ErrRPCParse.New(msg, err)
}

// These are a few thin wrappers around RPC errors, they're defined here so that the
// message text will be reliable because we *know* that people are going to start depending
// on them.

func errAccountNameNotFound() er.R {
	return btcjson.ErrRPCWalletInvalidAccountName.New("account name not found", nil)
}

func errNotImportedAccount() er.R {
	return btcjson.ErrRPCWallet.New("imported addresses must belong to the imported account", nil)
}

func errCommentsUnsupported() er.R {
	return btcjson.ErrRPCUnimplemented.New("Transaction comments are not yet supported", nil)
}

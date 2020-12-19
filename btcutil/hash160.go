// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcutil

import (
	"hash"

	sha256 "github.com/minio/sha256-simd"

	//lint:ignore SA1019 ripemd160 may be deprecated but it is not going away.
	"golang.org/x/crypto/ripemd160"
)

const Hash160Size = ripemd160.Size

// Calculate the hash of hasher over buf.
func calcHash(buf []byte, hasher hash.Hash) []byte {
	hasher.Write(buf)
	return hasher.Sum(nil)
}

// Ripemd160 calculates a ripemd160 hash directly
func Ripemd160(buf []byte) []byte {
	return calcHash(buf, ripemd160.New())
}

// Hash160 calculates the hash ripemd160(sha256(b)).
func Hash160(buf []byte) []byte {
	return Ripemd160(calcHash(buf, sha256.New()))
}

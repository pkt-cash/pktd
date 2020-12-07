// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pcutil

import (
	"encoding/binary"
	"fmt"

	"github.com/aead/chacha20"
	"golang.org/x/crypto/blake2b"
)

func HashExpand(out, key []byte, counter uint32) {
	if len(key) != 32 {
		panic("panic: PacketCrypt.Pcutil.Hash.HashExpand: unexpected key length")
	}
	nonce := []byte("____PC_EXPND")
	binary.LittleEndian.PutUint32(nonce[0:4], counter)
	for i := range out {
		out[i] = 0
	}
	//chacha20.XORKeyStream(out, out, &nonce, &key)
	chacha20.XORKeyStream(out, out, nonce, key)
}

func HashCompress(out, in []byte) {
	if len(out) < 32 {
		panic("need 32 byte output to place hash in")
	}
	b2, berr := blake2b.New256(nil)
	if berr != nil {
		panic(fmt.Sprintf("panic: PacketCrypt.Pcutil.Hash.HashCompress.blake2b.New256() failure\n	%v", berr))
	}
	_, err := b2.Write(in)
	if err != nil {
		panic(fmt.Sprintf("panic: PacketCrypt.Pcutil.Hash.HashCompress.blake2b.Write() failure\n	%v", err))
	}
	// blake2 wants to *append* the hash
	b2.Sum(out[:0])
}

func HashCompress64(out, in []byte) {
	if len(out) < 64 {
		panic("panic: PacketCrypt.Pcutil.Hash.HashCompress64 failure: need 64 byte output to place hash in")
	}
	b2, berr := blake2b.New512(nil)
	if berr != nil {
		panic(fmt.Sprintf("panic: PacketCrypt.Pcutil.Hash.HashCompress64.blake2b.New256() failure\n	%v", berr))
	}
	_, err := b2.Write(in)
	if err != nil {
		panic(fmt.Sprintf("panic: PacketCrypt.Pcutil.Hash.HasCompress64.blake2b.Write() failure\n	%v", err))
	}
	// blake2 wants to *append* the hash
	b2.Sum(out[:0])
}

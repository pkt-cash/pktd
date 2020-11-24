// Copyright (c) 2020 Anode LLC
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
//
// +build long_tests

package seedwords_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/pkt-cash/pktd/pktwallet/wallet/seedwords"
)

func TestEncrypt(t *testing.T) {
	seed, err := seedwords.RandomSeed()
	if err != nil {
		t.Error(err)
		return
	}
	t0 := time.Now()
	se := seed.Encrypt([]byte("password"))
	t1 := time.Now()
	fmt.Printf("Time spent encrypting: %s\n", t1.Sub(t0))
	if seed1, err := se.Decrypt([]byte("password"), false); err != nil {
		t.Error(err)
		return
	} else if !bytes.Equal(seed1.Bytes(), seed.Bytes()) {
		t.Error("Seed decrypt is not the same")
	}
}

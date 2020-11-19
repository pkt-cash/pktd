// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package packetcrypt_test

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/pkt-cash/pktd/blockchain/packetcrypt"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/cryptocycle"
	"github.com/pkt-cash/pktd/blockchain/testdata"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/chaincfg/globalcfg"
	"github.com/pkt-cash/pktd/wire"
)

func div16Ceil(x int) int {
	l := x / 16
	if l*16 < x {
		l++
	}
	return l
}

func doCryptoCycle(msg, additional, nonce, key []byte, decrypt bool) ([]byte, []byte, er.R) {
	if decrypt {
		msg = msg[:len(msg)-16]
	}
	s := cryptocycle.State{}
	s.SetDecrypt(decrypt)

	l16 := div16Ceil(len(msg))
	s.SetLength(byte(l16))
	s.SetTrailingZeros(byte(l16*16 - len(msg)))

	if additional == nil {
		additional = make([]byte, 0)
	}
	a16 := div16Ceil(len(additional))
	s.SetAddLen(byte(a16))
	s.SetAdditionalZeros(byte(a16*16 - len(additional)))

	copy(s.Bytes[16:48], key)
	copy(s.Bytes[48:48+len(additional)], additional)
	copy(s.Bytes[48+a16*16:], msg)

	cryptocycle.CryptoCycle(&s)

	if s.IsFailed() {
		return nil, nil, er.New("failed")
	}
	return s.Bytes[16:32], s.Bytes[48+a16*16:][:len(msg)], nil
}

func doGoCrypt(msg, additional, nonce, key []byte, decrypt bool) ([]byte, []byte, er.R) {
	chacha, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, nil, er.New("chacha.New()")
	}
	if decrypt {
		plain := make([]byte, 0, len(msg)-16)
		plain, err := chacha.Open(plain, nonce, msg, additional)
		if err != nil {
			return nil, nil, er.New("key.Open()")
		}
		return msg[len(msg)-16:], plain, nil
	}
	crypt := make([]byte, 0, len(msg)+16)
	crypt = chacha.Seal(crypt, nonce, msg, additional)
	return crypt[len(crypt)-16:], crypt[:len(crypt)-16], nil
}

func testCrypt(t *testing.T, msg, additional, nonce, key []byte, decrypt bool) []byte {
	if additional == nil {
		additional = make([]byte, 0)
	}
	if nonce == nil {
		nonce = make([]byte, 12)
	}
	gcp, gcm, err := doGoCrypt(msg, additional, nonce, key, decrypt)
	if err != nil {
		t.Errorf("failed doGoCrypt(%v %v %v %v %v)",
			hex.EncodeToString(msg), hex.EncodeToString(additional),
			hex.EncodeToString(nonce), hex.EncodeToString(key), decrypt)
	}
	ccp, ccm, err := doCryptoCycle(msg, additional, nonce, key, decrypt)
	if err != nil {
		t.Errorf("failed doGoCrypt(%v %v %v %v %v)",
			hex.EncodeToString(msg), hex.EncodeToString(additional),
			hex.EncodeToString(nonce), hex.EncodeToString(key), decrypt)
	}
	hccp := hex.EncodeToString(ccp)
	hgcp := hex.EncodeToString(gcp)
	hccm := hex.EncodeToString(ccm)
	hgcm := hex.EncodeToString(gcm)
	if hccp != hgcp {
		t.Errorf("(%v %v %v %v %v)",
			hex.EncodeToString(msg), hex.EncodeToString(additional),
			hex.EncodeToString(nonce), hex.EncodeToString(key), decrypt)
		t.Errorf("poly1305 different %v != %v", hccp, hgcp)
	}
	if hccm != hgcm {
		t.Errorf("(%v %v %v %v %v)",
			hex.EncodeToString(msg), hex.EncodeToString(additional),
			hex.EncodeToString(nonce), hex.EncodeToString(key), decrypt)
		t.Errorf("message different %v != %v", hccm, hgcm)
	}
	if decrypt {
		return ccm
	}
	out := make([]byte, len(ccm)+len(ccp))
	copy(out[:len(ccm)], ccm)
	copy(out[len(ccm):], ccp)
	return out
}

var dumbKey []byte = []byte("abcdefghijklmnopqrstuvwxyz012345")

func TestCryptoCycleEncrypt2(t *testing.T) {
	testMsg := []byte("16byte long test")
	res := hex.EncodeToString(testCrypt(t, testMsg, nil, nil, dumbKey, false))
	if res != "aea649b893a601fc2654e9d57d0ad1620351b1b107b3d352e7110d1c140a8e2d" {
		t.Fail()
	}
}

func TestCryptoCycleEncryptAdd(t *testing.T) {
	testMsg := []byte("17byte long test.")
	testAdd := []byte("additional")
	res := hex.EncodeToString(testCrypt(t, testMsg, testAdd, nil, dumbKey, false))
	if res != "aea749b893a601fc2654e9d57d0ad162baf683f7db7ae9ffcd943578e350ab9f74" {
		t.Fail()
	}
}

func TestCryptoCycleDecrypt2(t *testing.T) {
	msg, _ := hex.DecodeString("aea649b893a601fc2654e9d57d0ad1620351b1b107b3d352e7110d1c140a8e2d")
	res := hex.EncodeToString(testCrypt(t, msg, nil, nil, dumbKey, true))
	expectedResult := hex.EncodeToString([]byte("16byte long test"))
	if res != expectedResult {
		t.Fail()
	}
}

func TestCryptoCycleEncrypt3(t *testing.T) {
	testMsg := []byte("17byte long test.")
	res := hex.EncodeToString(testCrypt(t, testMsg, nil, nil, dumbKey, false))
	if res != "aea749b893a601fc2654e9d57d0ad162ba013c000dbaad4d14eecf09da264443d3" {
		t.Fail()
	}
}

func TestCryptoCycleDecrypt3(t *testing.T) {
	msg, _ := hex.DecodeString("aea749b893a601fc2654e9d57d0ad162ba013c000dbaad4d14eecf09da264443d3")
	res := hex.EncodeToString(testCrypt(t, msg, nil, nil, dumbKey, true))
	expectedResult := hex.EncodeToString([]byte("17byte long test."))
	if res != expectedResult {
		t.Fail()
	}
}

func TestCryptoCycleWireguard(t *testing.T) {
	// taken from wireguard-go tests (with some modification to make them produce the same key)
	keyBytes, err := hex.DecodeString("6d26aa2ebac8769929832918edbdaa5aa64488d6d489571704b02512b233c6a0")
	if err != nil {
		t.Fail()
	}
	testMsg := []byte("wireguard test message 1")
	out := testCrypt(t, testMsg, nil, nil, keyBytes, false)
	resultHex := hex.EncodeToString(out)
	if resultHex != "3014f78ee7d2fe4455f5d3c2ab2315938f53550e56e2f01aa2738191abf38d3d96f73057fbf116ff" {
		t.Fail()
	}
}

func TestInsertCoinbaseCommit(t *testing.T) {
	cbcBytes, _ := hex.DecodeString(
		"09f91102ffff0320e531a06a3c672d3d6f3d31cd9e8c77b2c0afe03d0a1b9546" +
			"e8c20af5f17700d7ba00000000000000")
	cbc := wire.PcCoinbaseCommit{}
	copy(cbc.Bytes[:], cbcBytes)

	block := testdata.GetBlock("../testdata/277647.dat.bz2", t)
	mb := block.MsgBlock()

	packetcrypt.InsertCoinbaseCommit(mb.Transactions[0], &cbc)

	cbc2 := packetcrypt.ExtractCoinbaseCommit(mb.Transactions[0])
	if cbc2 == nil {
		t.Errorf("ExtractCoinbaseCommit() returned nil")
	}

	if !bytes.Equal(cbc2.Bytes[:], cbc.Bytes[:]) {
		t.Errorf("expected %v", hex.EncodeToString(cbc.Bytes[:]))
		t.Errorf("got      %v", hex.EncodeToString(cbc2.Bytes[:]))
		t.Errorf("cbc mismatch")
		return
	}
}

func TestMain(m *testing.M) {
	globalcfg.SelectConfig(chaincfg.PktMainNetParams.GlobalConf)
	os.Exit(m.Run())
}

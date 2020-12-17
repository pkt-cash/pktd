// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package packetcrypt

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/dchest/blake2b"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/announce"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/block"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/pcutil"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/wire"
	ed25519 "github.com/hdevalence/ed25519consensus"
)

func ValidatePcAnn(p *wire.PacketCryptAnn, parentBlockHash *chainhash.Hash, packetCryptVersion int) (*chainhash.Hash, er.R) {
	return announce.CheckAnn(p, parentBlockHash, packetCryptVersion)
}

func checkContentProof(ann *wire.PacketCryptAnn, proofIdx uint32, cpb io.Reader) er.R {
	contentLength := ann.GetContentLength()
	totalBlocks := contentLength / 32
	if totalBlocks*32 < contentLength {
		totalBlocks++
	}

	blockToProve := proofIdx % totalBlocks
	depth := pcutil.Log2ceil(uint64(totalBlocks))
	var buf [64]byte
	var hash [32]byte

	if _, err := io.ReadFull(cpb, hash[:]); err != nil {
		return er.Errorf("checkContentProof: 0 unable to read ann content proof [%s]", err)
	}
	blockSize := uint32(32)
	for i := 0; i < depth; i++ {
		if blockSize*(blockToProve^1) >= contentLength {
			blockToProve >>= 1
			blockSize <<= 1
			continue
		}
		copy(buf[((blockToProve)&1)*32:][:32], hash[:])
		if _, err := io.ReadFull(cpb, buf[((^blockToProve)&1)*32:][:32]); err != nil {
			return er.Errorf("checkContentProof: 1 unable to read ann content proof [%s]", err)
		}
		blockToProve >>= 1
		blockSize <<= 1
		b2 := blake2b.New256()
		b2.Write(buf[:])
		x := b2.Sum(nil)
		copy(hash[:], x)
	}
	if !bytes.Equal(hash[:], ann.GetContentHash()) {
		return er.New("announcement content proof hash mismatch")
	}
	return nil
}

func contentProofIdx2(mb *wire.MsgBlock) uint32 {
	b2 := blake2b.New256()
	mb.Header.Serialize(b2)
	buf := b2.Sum(nil)
	return binary.LittleEndian.Uint32(buf) ^ mb.Pcp.Nonce
}

func ValidatePcBlock(mb *wire.MsgBlock, height int32, shareTarget uint32, annParentHashes []*chainhash.Hash) (bool, er.R) {
	if len(annParentHashes) != 4 {
		return false, er.New("wrong number of annParentHashes")
	}
	if mb.Pcp == nil {
		return false, er.New("missing packetcrypt proof")
	}

	// Check ann sigs
	for i, ann := range mb.Pcp.Announcements {
		if !ann.HasSigningKey() {
		} else if mb.Pcp.Signatures[i] == nil {
			return false, er.Errorf("missing announcement signature for key [%s]",
				hex.EncodeToString(ann.GetSigningKey()))
		} else if !ed25519.Verify(ann.GetSigningKey(), ann.Header[:], mb.Pcp.Signatures[i]) {
			return false, er.New("invalid announcement signature")
		}
	}

	// Check content proofs
	proofIdx := contentProofIdx2(mb)
	var contentProofs [][]byte
	if mb.Pcp.Version <= 1 {
		var err er.R
		contentProofs, err = mb.Pcp.SplitContentProof(proofIdx)
		if err != nil {
			return false, err
		}
		for i, ann := range mb.Pcp.Announcements {
			if ann.GetContentLength() <= 32 {
				continue
			}
			if contentProofs[i] == nil {
				return false, er.New("missing announcement content proof")
			}
			contentBuf := bytes.NewBuffer(contentProofs[i])
			if err := checkContentProof(&ann, proofIdx, contentBuf); err != nil {
				return false, err
			}
		}
	} else if mb.Pcp.ContentProof != nil {
		return false, er.Errorf("For PcP type [%d] content proof must be nil", mb.Pcp.Version)
	}

	coinbase := mb.Transactions[0]
	if coinbase == nil {
		return false, er.New("missing coinbase")
	}
	cbc := ExtractCoinbaseCommit(coinbase)
	if cbc == nil {
		return false, er.New("missing packetcrypt commitment")
	}
	return block.ValidatePcProof(
		mb.Pcp, height, &mb.Header, cbc, shareTarget, annParentHashes, contentProofs, mb.Pcp.Version)
}

var pcCoinbasePrefix = [...]byte{0x6a, 0x30, 0x09, 0xf9, 0x11, 0x02}

func ExtractCoinbaseCommit(coinbaseTx *wire.MsgTx) *wire.PcCoinbaseCommit {
	for _, tx := range coinbaseTx.TxOut {
		if len(tx.PkScript) > 6 && bytes.Equal(tx.PkScript[:6], pcCoinbasePrefix[:]) {
			out := wire.PcCoinbaseCommit{}
			copy(out.Bytes[:], tx.PkScript[2:])
			return &out
		}
	}
	return nil
}

func InsertCoinbaseCommit(coinbaseTx *wire.MsgTx, cbc *wire.PcCoinbaseCommit) {
	buf := make([]byte, len(cbc.Bytes)+2)
	buf[0] = 0x6a
	buf[1] = 0x30
	copy(buf[2:], cbc.Bytes[:])
	coinbaseTx.AddTxOut(&wire.TxOut{PkScript: buf})
}

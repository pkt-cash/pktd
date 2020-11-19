// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"encoding/binary"
	"io"

	"github.com/pkt-cash/pktd/blockchain/packetcrypt/pcutil"
	"github.com/pkt-cash/pktd/btcutil/er"
)

// PacketCryptAnn is the in-memory structure of a PacketCrypt announcement
type PacketCryptAnn struct {
	Header [1024]byte
}

// PcAnnSerializeSize how big the announcement is when serialized
const PcAnnSerializeSize = 1024

// PcAnnHeaderLen is the length of the announcement header (not including merkle proof)
const PcAnnHeaderLen = 88

// PcAnnMerkleProofLen is the length of the merkle proof
const PcAnnMerkleProofLen = 896

// PcItem4PrefixLen is the length of the item 4 prefix which follows the merkle proof
const PcItem4PrefixLen = PcAnnSerializeSize - (PcAnnHeaderLen + PcAnnMerkleProofLen)

// GetVersion gets the version number of the announcement.
func (p *PacketCryptAnn) GetVersion() uint {
	return uint(p.Header[0])
}

// GetAnnounceHeader provides the header without the merkle proof
func (p *PacketCryptAnn) GetAnnounceHeader() []byte {
	return p.Header[:PcAnnHeaderLen]
}

// GetMerkleProof provides the memory location in the announcement of the merkle branch
func (p *PacketCryptAnn) GetMerkleProof() []byte {
	return p.Header[PcAnnHeaderLen : PcAnnHeaderLen+PcAnnMerkleProofLen]
}

// GetItem4Prefix provides the memory location of item4 which must be stored in the announcement
func (p *PacketCryptAnn) GetItem4Prefix() []byte {
	return p.Header[PcAnnHeaderLen+PcAnnMerkleProofLen:]
}

// GetSoftNonce outputs a byte slice with bytes 1, 2 and 3
func (p *PacketCryptAnn) GetSoftNonce() []byte {
	return p.Header[1:4]
}

// GetParentBlockHeight returns the block height of the block whose hash is
// committed in this announcement, the age of the announcement (in blocks)
// influences it's value to a miner.
func (p *PacketCryptAnn) GetParentBlockHeight() uint32 {
	return binary.LittleEndian.Uint32(p.Header[12:16])
}

// GetWorkTarget provides the PoW target for the announcement, in bitcoin
// compressed bignum format
func (p *PacketCryptAnn) GetWorkTarget() uint32 {
	return binary.LittleEndian.Uint32(p.Header[8:12])
}

// // GetContentType provides the uint32 content type ID
// func (p *PacketCryptAnn) GetContentType() uint32 {
// 	return binary.LittleEndian.Uint32(p.Header[16:20])
// }

// GetContentLength provides the length of the announcement content
func (p *PacketCryptAnn) GetContentLength() uint32 {
	return binary.LittleEndian.Uint32(p.Header[20:24])
}

// GetContentHash returns the content merkle root in the event that the
// announcement has external content, or in the event that the announcement
// has internal content, it is the content itself (right-padded to 32 bytes)
func (p *PacketCryptAnn) GetContentHash() []byte {
	return p.Header[24:56]
}

// GetSigningKey returns the memory location in the ann of the signing key
// warning: mutating the result will mutate the announcement
func (p *PacketCryptAnn) GetSigningKey() []byte {
	return p.Header[56:88]
}

// HasSigningKey returns true if a signing key is specified
func (p *PacketCryptAnn) HasSigningKey() bool {
	return !pcutil.IsZero(p.GetSigningKey())
}

// BtcDecode decodes an announcement from a reader
func (p *PacketCryptAnn) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) er.R {
	_, err := io.ReadFull(r, p.Header[:])
	return er.E(err)
}

// BtcEncode encodes an announcement to a writer
func (p *PacketCryptAnn) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) er.R {
	_, err := w.Write(p.Header[:])
	return er.E(err)
}

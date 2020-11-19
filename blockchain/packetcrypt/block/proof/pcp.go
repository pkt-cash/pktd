// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package proof

import (
	"bytes"
	"encoding/binary"

	"github.com/pkt-cash/pktd/blockchain/packetcrypt/pcutil"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/wire"
)

const uint64Max uint64 = 0xffffffffffffffff

func PcpHash(
	annHashes *[4][32]byte,
	annCount uint64,
	annIndexes *[4]uint64,
	pcp *wire.PacketCryptProof,
) (*[32]byte, er.R) {

	// We need to bump the numbers to account for the zero entry
	var annIdxs [4]uint64
	for i := 0; i < 4; i++ {
		annIdxs[i] = (annIndexes[i] % annCount) + 1
	}
	annCount++

	tree, err := NewTree(annCount, &annIdxs)
	if err != nil {
		return nil, err
	}

	defer tree.DumpTree()

	// fill in announcement hashes
	for i := 0; i < 4; i++ {
		e := tree.GetAnnEntry(annIdxs[i])
		log.Trace("Setting ann hash")
		if !e.SetHash(annHashes[i][:]) {
			return nil, er.New("SetHash returned false, duplicate data")
		}
	}

	buf := bytes.NewBuffer(pcp.AnnProof)

	// Fill in the hashes and ranges which are provided
	for i := 0; i < len(tree.entries); i++ {
		e := &tree.entries[i]
		if e.HasExplicitRange() {
			var raNge [8]byte
			if _, err := buf.Read(raNge[:]); err != nil {
				return nil, er.New("runt input")
			}
			log.Trace("Setting explicit range")
			if !e.SetRange(binary.LittleEndian.Uint64(raNge[:])) {
				return nil, er.New("SetRange returned false, duplicate data")
			}
		}
		if (e.Flags() & (FHasHash | FComputable)) == 0 {
			var hash [32]byte
			if _, err := buf.Read(hash[:]); err != nil {
				return nil, er.New("runt input")
			}
			log.Trace("Setting provided hash")
			if !e.SetHash(hash[:]) {
				return nil, er.New("SetHash returned false, duplicate data")
			}
		}
	}
	if buf.Len() != 0 {
		return nil, er.New("extra data at the end of the proof")
	}

	// The rules should have triggered computation of everything missing
	r := tree.GetRoot()
	if r.Flags() != FComputable|FFirstEntry|FHasHash|FHasRange|FHasStart|FHasEnd {
		return nil, er.Errorf("root has unexpected flags %v", r.Flags())
	}
	if r.Start() != 0 || r.End() != uint64Max || r.Range() != uint64Max {
		return nil, er.New("root has unexpected values for start/end/range")
	}

	var outBuf [48]byte
	copy(outBuf[:32], r.Hash())
	pcutil.Memset(outBuf[40:], 0xff)

	ret := new([32]byte)
	pcutil.HashCompress(ret[:], outBuf[:])
	return ret, nil
}

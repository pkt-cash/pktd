// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"bytes"
	"encoding/binary"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/blockchain/packetcrypt/cryptocycle"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/difficulty"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/pcutil"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/randhash/interpret"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/randhash/randgen"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/randhash/util"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/wire"
)

const (
	announceMerkleDepth int    = 13
	announceTableSz     uint64 = 1 << uint(announceMerkleDepth)
)

type context struct {
	itemBytes [1024]byte
	ann       wire.PacketCryptAnn
	annHash0  [64]byte
	annHash1  [64]byte
	ccState   cryptocycle.State
	progBuf   cryptocycle.Context
	item4Hash [64]byte
}

func memocycle(item *[1024]byte, bufcount, cycles int) {
	var tmpbuf [128]byte
	for cycle := 0; cycle < cycles; cycle++ {
		for i := 0; i < bufcount; i++ {
			p := (i - 1 + bufcount) % bufcount
			q := int(binary.LittleEndian.Uint32(item[64*p:][:4]) % uint32(bufcount-1))
			j := (i + q) % bufcount
			copy(tmpbuf[:64], item[64*p:][:64])
			copy(tmpbuf[64:], item[64*j:][:64])
			pcutil.HashCompress64(item[i*64:][:64], tmpbuf[:])
		}
	}
}

const announceItemHashcount int = 1024 / 64

func MkItem(itemNo int, item *[1024]byte, seed []byte) {
	pcutil.HashExpand(item[:64], seed, uint32(itemNo))
	for i := 1; i < announceItemHashcount; i++ {
		pcutil.HashCompress64(item[64*i:][:64], item[64*(i-1):][:64])
	}
	memocycle(item, announceItemHashcount, util.Conf_AnnHash_MEMOHASH_CYCLES)
}

type mkItem2Program struct {
	memory [8192]byte
	prog   []uint32
}

func mkItem2Prog(out *mkItem2Program, seed []byte) int {
	pcutil.HashExpand(out.memory[:], seed, 0)
	prog, err := randgen.Generate(seed)
	if err != nil {
		return -1
	}
	for i := 0; i < len(prog); i++ {
		binary.LittleEndian.PutUint32(out.memory[i*4:i*4+4], prog[i])
	}
	out.prog = prog
	return 0
}

func mkItem2(itemNo int, item []byte, seed []byte, prog *mkItem2Program) int {
	state := cryptocycle.State{}
	cryptocycle.Init(&state, seed, uint64(itemNo))
	memoryBeginning := itemNo % ((len(prog.memory) / 4) - interpret.RandHash_MEMORY_SZ)
	memoryEnd := memoryBeginning + interpret.RandHash_MEMORY_SZ
	memorySlice := prog.memory[4*memoryBeginning : 4*memoryEnd]
	if interpret.Interpret(prog.prog, state.Bytes[:], memorySlice, 2) != nil {
		return -1
	}
	state.MakeFuzzable()
	cryptocycle.CryptoCycle(&state)
	if state.IsFailed() {
		panic("CryptoCycle went into a failed state, should not happen")
	}
	copy(item, state.Bytes[:1024])
	return 0
}

func annDecrypt(pcAnn *wire.PacketCryptAnn, state *cryptocycle.State) *wire.PacketCryptAnn {
	out := wire.PacketCryptAnn{}
	copy(out.Header[:], pcAnn.Header[:])
	j := 0
	for i := wire.PcAnnHeaderLen; i < wire.PcAnnHeaderLen+wire.PcAnnMerkleProofLen-64; i++ {
		out.Header[i] ^= state.Bytes[j]
		j++
	}
	for i := wire.PcAnnHeaderLen + wire.PcAnnMerkleProofLen; i < 1024; i++ {
		out.Header[i] ^= state.Bytes[j]
		j++
	}
	return &out
}

func merkleIsValid(merkleProof []byte, item4Hash *[64]byte, itemNo int) bool {
	var buf [128]byte
	copy(buf[64*(itemNo&1):][:64], item4Hash[:])
	for i := 0; i < announceMerkleDepth; i++ {
		copy(buf[64*((^itemNo)&1):][:64], merkleProof[i*64:][:64])
		itemNo >>= 1
		pcutil.HashCompress64(buf[64*(itemNo&1):][:64], buf[:])
	}
	return bytes.Equal(buf[64*(itemNo&1):][:64], merkleProof[64*announceMerkleDepth:])
}

func CheckAnn(pcAnn *wire.PacketCryptAnn, parentBlockHash *chainhash.Hash, packetCryptVersion int) (*chainhash.Hash, er.R) {
	if pcAnn.GetVersion() > 0 && pcAnn.GetParentBlockHeight() < 103869 {
		return nil, er.New("Validate_checkAnn_ANN_VERSION_NOT_ALLOWED")
	} else if packetCryptVersion > 1 && pcAnn.GetVersion() == 0 {
		return nil, er.New("Validate_checkAnn_ANN_VERSION_MISMATCH")
	}
	ctx := new(context)
	copy(ctx.ann.GetAnnounceHeader(), pcAnn.GetAnnounceHeader())
	copy(ctx.ann.GetMerkleProof()[:32], parentBlockHash[:])
	pcutil.Zero(ctx.ann.GetSoftNonce())
	pcutil.HashCompress64(ctx.annHash0[:], ctx.ann.Header[:wire.PcAnnHeaderLen+64])
	copy(ctx.ann.GetMerkleProof(), pcAnn.GetMerkleProof()[13*64:])
	pcutil.HashCompress64(ctx.annHash1[:], ctx.ann.Header[:wire.PcAnnHeaderLen+64])

	var softNonceBuf [4]byte
	copy(softNonceBuf[:], pcAnn.GetSoftNonce())
	softNonce := binary.LittleEndian.Uint32(softNonceBuf[:])
	mkItemSeed := ctx.annHash0[:32]
	randHashCycles := util.Conf_AnnHash_RANDHASH_CYCLES
	version := pcAnn.GetVersion()
	prog := mkItem2Program{}
	if version > 0 {
		randHashCycles = 0
		if softNonce > difficulty.Pc2AnnSoftNonceMax(pcAnn.GetWorkTarget()) {
			return nil, er.New("Validate_checkAnn_SOFT_NONCE_HIGH")
		}
		buf := make([]byte, 64*2)
		copy(buf[:64], pcAnn.GetMerkleProof()[13*64:])
		copy(buf[64:], ctx.annHash0[:])
		pcutil.HashCompress64(buf[:64], buf)
		mkItemSeed = buf[:64]
		if mkItem2Prog(&prog, mkItemSeed[:32]) != 0 {
			return nil, er.New("Validate_checkAnn_BAD_PROGRAM")
		}
	}
	cryptocycle.Init(&ctx.ccState, ctx.annHash1[:32], uint64(softNonce))
	itemNo := -1
	for i := 0; i < 4; i++ {
		itemNo = int(cryptocycle.GetItemNo(&ctx.ccState) % announceTableSz)
		if version > 0 {
			if mkItem2(itemNo, ctx.itemBytes[:], mkItemSeed[32:], &prog) != 0 {
				return nil, er.New("Validate_checkAnn_BAD_PROGRAM_EXEC")
			}
		} else {
			// only 32 bytes of the seed are used
			MkItem(itemNo, &ctx.itemBytes, mkItemSeed)
		}
		if !cryptocycle.Update(
			&ctx.ccState, ctx.itemBytes[:], nil, randHashCycles, &ctx.progBuf) {
			return nil, er.New("Validate_checkAnn_INVAL")
		}
	}

	cryptocycle.Final(&ctx.ccState)
	if version > 0 {
		pcAnn = annDecrypt(pcAnn, &ctx.ccState)
	}

	// fmt.Printf("%s\n", hex.EncodeToString(pcAnn.Header[:]))

	if version > 0 {
		if !pcutil.IsZero(pcAnn.GetItem4Prefix()) {
			return nil, er.New("Validate_checkAnn_INVAL_ITEM4")
		}
		if mkItem2Prog(&prog, ctx.annHash0[:32]) != 0 {
			return nil, er.New("Validate_checkAnn_BAD_PROGRAM0")
		}
		if mkItem2(itemNo, ctx.itemBytes[:], ctx.annHash0[32:], &prog) != 0 {
			return nil, er.New("Validate_checkAnn_BAD_PROGRAM0_EXEC")
		}
	} else if !bytes.Equal(ctx.itemBytes[:wire.PcItem4PrefixLen], pcAnn.GetItem4Prefix()) {
		return nil, er.New("Validate_checkAnn_INVAL_ITEM4")
	}
	pcutil.HashCompress64(ctx.item4Hash[:], ctx.itemBytes[:])
	if !merkleIsValid(pcAnn.GetMerkleProof(), &ctx.item4Hash, itemNo) {
		return nil, er.New("Validate_checkAnn_INVAL_MERKLE")
	}

	target := pcAnn.GetWorkTarget()

	h := chainhash.Hash{}
	copy(h[:], ctx.ccState.Bytes[:32])
	if !difficulty.IsOk(ctx.ccState.Bytes[:32], target) {
		return &h, er.Errorf("Validate_checkAnn_INSUF_POW need target [%x] "+
			"but ann work hash is [%s]", target, h.String())
	}

	return &h, nil
}

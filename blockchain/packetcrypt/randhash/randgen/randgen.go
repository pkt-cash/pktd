// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package randgen

import (
	"encoding/binary"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/blockchain/packetcrypt/pcutil"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/randhash/opcodes"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/randhash/util"
)

type randGen struct {
	randseed [32]byte
	randbuf  [64]byte
	nextInt  int
	ctr      uint32

	insns []uint32

	vars  []uint32
	scope int
}

func randu32(ctx *randGen) uint32 {
	if ctx.nextInt >= (len(ctx.randbuf) / 4) {
		pcutil.HashExpand(ctx.randbuf[:], ctx.randseed[:], ctx.ctr)
		ctx.ctr++
		ctx.nextInt = 0
	}
	out := binary.LittleEndian.Uint32(ctx.randbuf[ctx.nextInt*4:][:4])
	ctx.nextInt++
	return out
}

func cointoss(ctx *randGen, oneIn int) bool {
	return (randu32(ctx) % uint32(oneIn)) == 0
}

func randRange(ctx *randGen, start, end uint32) uint32 {
	return randu32(ctx)%(end-start) + start
}

func spend(budget *int, amount int) bool {
	if *budget >= amount {
		*budget -= amount
		return true
	}
	return false
}

func emit(ctx *randGen, insn uint32) {
	op := util.DecodeInsn_OP(insn)
	if op <= opcodes.OpCode_INVALID_ZERO || op >= opcodes.OpCode_INVALID_BIG {
		panic("op out of range")
	}
	if len(ctx.insns) > util.Conf_RandGen_MAX_INSNS {
		return
	}
	ctx.insns = append(ctx.insns, insn)
}

func scope(ctx *randGen) { ctx.scope++; ctx.vars = append(ctx.vars, ^uint32(0)) }
func end(ctx *randGen) {
	emit(ctx, opcodes.OpCode_END)
	ctx.scope--
	for {
		top := ctx.vars[len(ctx.vars)-1]
		ctx.vars = ctx.vars[:len(ctx.vars)-1]
		if top == ^uint32(0) {
			break
		}
	}
}

func mkVar(ctx *randGen) { ctx.vars = append(ctx.vars, 0) }

func _getVar(ctx *randGen, dbl bool) int {
	eof := len(ctx.vars)
	bof := eof - 1
	for ; bof >= 0; bof-- {
		if ctx.vars[bof] != ^uint32(0) {
			continue
		}
		// only 1 var in this frame and we're looking for dword, continue looking
		if dbl {
			if bof >= eof-2 {
				goto nextFrame
			}
		} else {
			// no vars in this frame, continue looking
			if bof >= eof-1 {
				goto nextFrame
			}
		}
		// end of the line, this is tested after because first frame should always have 4 vars.
		if bof == 0 {
			break
		}
		// walk up to a higher scope
		if !cointoss(ctx, util.Conf_RandGen_HIGHER_SCOPE_LIKELIHOOD) {
			break
		}
	nextFrame:
		eof = bof
	}
	// printf("%d %d - %d [%08x]\n", bof, eof, dbl, ctx->vars.elems[0]);
	if bof < 0 {
		panic("bof out of range")
	}
	start := randRange(ctx, uint32(bof+1), uint32(eof))
	// printf("%d %d %d - %d\n", bof, eof, start, dbl);
	for j := int(start) + 1; ; j++ {
		if j >= eof {
			j = bof + 1
		}
		// printf("%08x %d\n", ctx->vars.elems[j], j);
		if (!dbl || (j > bof+1)) && cointoss(ctx, util.Conf_RandGen_VAR_REUSE_LIKELIHOOD) {
			// printf("reuse\n");
			return j
		} else if (ctx.vars[j] & 1) == 0 {
			if !dbl || (ctx.vars[j-1]&1) == 0 {
				return j
			}
		}
	}
}

func getVar(ctx *randGen, dbl bool) int {
	out := _getVar(ctx, dbl)
	if ctx.vars[out] == ^uint32(0) {
		panic("getVar out references a frame boundary")
	}
	ctx.vars[out] |= 1
	if dbl {
		if ctx.vars[out-1] == ^uint32(0) {
			panic("getVar out-1 references a frame boundary")
		}
		ctx.vars[out-1] |= 1
	}
	return out
}

func getA(ctx *randGen, dbl bool) uint32 { return uint32(getVar(ctx, dbl)) << 9 }
func getB(ctx *randGen, dbl bool) uint32 {
	if cointoss(ctx, util.Conf_RandGen_IMMEDIATE_LIKELIHOOD) {
		return (randu32(ctx) << 20) | (1 << 18)
	}
	return uint32(getVar(ctx, dbl)) << 20
}

type opType int

const (
	opType_1_1 = iota
	opType_2_1
	opType_2_2
	opType_4_2
	opType_4_4
)

var costByType = [...]int{
	1,  // opType_1_1
	2,  // opType_2_1
	4,  // opType_2_2
	8,  // opType_4_2
	16, // opType_4_4
}

func getOp(codes []opcodes.OpCode, randval uint32) uint32 {
	return uint32(codes[randval%uint32(len(codes))])
}

func op(ctx *randGen, t opType, budget *int) bool {
	rand := randu32(ctx)
	if !spend(budget, costByType[t]) {
		return false
	}
	switch t {
	case opType_1_1:
		{
			emit(ctx, getOp(opcodes.CODES_1_1[:], rand)|getA(ctx, false))
			mkVar(ctx)
			break
		}
	case opType_2_1:
		{
			emit(ctx, getOp(opcodes.CODES_2_1[:], rand)|getA(ctx, false)|getB(ctx, false))
			mkVar(ctx)
			break
		}
	case opType_2_2:
		{
			emit(ctx, getOp(opcodes.CODES_2_2[:], rand)|getA(ctx, false)|getB(ctx, false))
			mkVar(ctx)
			mkVar(ctx)
			break
		}
	case opType_4_2:
		{
			emit(ctx, getOp(opcodes.CODES_4_2[:], rand)|getA(ctx, true)|getB(ctx, true))
			mkVar(ctx)
			mkVar(ctx)
			break
		}
	case opType_4_4:
		{
			emit(ctx, getOp(opcodes.CODES_4_4[:], rand)|getA(ctx, true)|getB(ctx, true))
			mkVar(ctx)
			mkVar(ctx)
			mkVar(ctx)
			mkVar(ctx)
			break
		}
	}
	return true
}

func input(ctx *randGen, budget *int) bool {
	if !spend(budget, util.Conf_RandGen_INPUT_COST) {
		return false
	}
	mkVar(ctx)
	emit(ctx, (randu32(ctx)<<8)|uint32(opcodes.OpCode_IN))
	return true
}

func branch(ctx *randGen, budget *int) bool {
	if !spend(budget, util.Conf_RandGen_BRANCH_COST) {
		return false
	}
	op := opcodes.OpCode_IF_LIKELY
	if cointoss(ctx, util.Conf_RandGen_RANDOM_BRANCH_LIKELIHOOD) {
		op = opcodes.OpCode_IF_RANDOM
	}

	emit(ctx, getA(ctx, false)|uint32(op)|(2<<20))
	j1 := uint32(len(ctx.insns))
	emit(ctx, opcodes.OpCode_JMP)

	b1 := util.Conf_RandGen_IF_BODY_BUDGET(*budget, ctx.scope)
	body(ctx, &b1, true)

	j2 := uint32(len(ctx.insns))
	emit(ctx, opcodes.OpCode_JMP)

	b2 := util.Conf_RandGen_IF_BODY_BUDGET(*budget, ctx.scope)
	body(ctx, &b2, true)

	if (j2 - j1) >= (1 << 23) {
		panic("if body too large")
	}
	if (uint32(len(ctx.insns)) - j2) >= (1 << 23) {
		panic("if else body too large")
	}

	// Now we fill in the first jmp
	ctx.insns[j1] = ((j2 - j1) << 8) | uint32(opcodes.OpCode_JMP)

	// and then the else jmp
	ctx.insns[j2] = ((uint32(len(ctx.insns)) - j2 - 1) << 8) | uint32(opcodes.OpCode_JMP)
	return true
}

func loop(ctx *randGen, budget *int) int {
	loopLen := int(randRange(ctx,
		util.Conf_RandGen_LOOP_MIN_CYCLES,
		uint32(util.Conf_RandGen_LOOP_MAX_CYCLES(ctx.scope))))
	// this must be at least 2
	numMemAcc := int(randRange(ctx, 2, 4))

	if *budget < (util.Conf_RandGen_MEMORY_COST * loopLen) {
		return 0
	}
	*budget /= loopLen
	emit(ctx, (uint32(loopLen)<<20)|uint32(opcodes.OpCode_LOOP))
	scope(ctx)

	memTemplate := (randu32(ctx) << 8) | uint32(opcodes.OpCode_MEMORY)
	for i := 0; i < numMemAcc; i++ {
		if !spend(budget, util.Conf_RandGen_MEMORY_COST) {
			break
		}
		mkVar(ctx)
		emit(ctx, util.DecodeInsn_MEMORY_WITH_CARRY(memTemplate, randu32(ctx)))
	}
	ret := body(ctx, budget, false)
	end(ctx)
	return ret
}

func body(ctx *randGen, budget *int, createScope bool) int {
	if createScope {
		scope(ctx)
	}
	for {
		if len(ctx.insns) > util.Conf_RandGen_MAX_INSNS {
			goto out
		}
		max := int(randRange(ctx, 2, 12))
		for i := 1; i <= max; i++ {
			if cointoss(ctx, 4*max/i) && op(ctx, opType_4_4, budget) {
				continue
			}
			if cointoss(ctx, 3*max/i) && op(ctx, opType_4_2, budget) {
				continue
			}
			if cointoss(ctx, 3*max/i) && op(ctx, opType_2_2, budget) {
				continue
			}
			if cointoss(ctx, 2*max/i) && op(ctx, opType_2_1, budget) {
				continue
			}
			if cointoss(ctx, 1*i) && input(ctx, budget) {
				continue
			}
			if op(ctx, opType_1_1, budget) {
				continue
			}
			goto out
		}
		if util.Conf_RandGen_SHOULD_BRANCH(randu32(ctx), len(ctx.insns)) && !branch(ctx, budget) {
			goto out
		}
		if util.Conf_RandGen_SHOULD_LOOP(randu32(ctx)) && loop(ctx, budget) == 0 {
			goto out
		}
	}
out:
	if createScope {
		end(ctx)
	}
	return 0
}

func Generate(seed []byte) ([]uint32, er.R) {
	budget := util.Conf_RandGen_INITIAL_BUDGET
	ctx := randGen{}
	copy(ctx.randseed[:], seed[:32])
	ctx.insns = make([]uint32, 0)
	ctx.vars = make([]uint32, 0)
	ctx.nextInt = 1000000

	loop(&ctx, &budget)

	if len(ctx.insns) < util.Conf_RandGen_MIN_INSNS {
		return nil, er.New("insn count < Conf_RandGen_MIN_INSNS")
	} else if len(ctx.insns) > util.Conf_RandGen_MAX_INSNS {
		return nil, er.New("insn count > Conf_RandGen_MAX_INSNS")
	}

	return ctx.insns, nil
}

// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package interpret

import (
	"encoding/binary"
	"fmt"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/blockchain/packetcrypt/randhash/opcodes"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/randhash/util"
)

type Uint64 struct {
	lo uint32
	hi uint32
}

// keep these powers of 2 because there is unsigned modulo using &
// Also be careful not to change these without also checking the buffers
// which are passed to RandHash_execute()
const (
	RandHash_MEMORY_SZ int   = 256
	RandHash_INOUT_SZ  int64 = 256
)

func getReg(stack []uint32, index uint32) uint32 {
	if index > uint32(len(stack)) {
		panic("reference to out-of-bounds register")
	}
	return stack[index]
}

func getA(ctx *Context, insn uint32) uint32 {
	return getReg(ctx.stack, util.DecodeInsn_REGA(insn))
}

func getB(ctx *Context, insn uint32) uint32 {
	if util.DecodeInsn_HAS_IMM(insn) {
		return uint32(util.DecodeInsn_immLo(insn))
	}
	return getReg(ctx.stack, util.DecodeInsn_REGB(insn))
}

func getA2(ctx *Context, insn uint32) Uint64 {
	return Uint64{
		lo: getReg(ctx.stack, util.DecodeInsn_REGA(insn)-1),
		hi: getReg(ctx.stack, util.DecodeInsn_REGA(insn)),
	}
}

func getB2(ctx *Context, insn uint32) Uint64 {
	if util.DecodeInsn_HAS_IMM(insn) {
		imm := util.DecodeInsn_imm(insn)
		return Uint64{
			lo: uint32(imm),
			hi: uint32(uint64(imm) >> 32),
		}
	}
	return Uint64{
		lo: getReg(ctx.stack, util.DecodeInsn_REGB(insn)-1),
		hi: getReg(ctx.stack, util.DecodeInsn_REGB(insn)),
	}
}

func out1(ctx *Context, val uint32) {
	ctx.varCount++
	ctx.stack = append(ctx.stack, val)
}

func out2(ctx *Context, a uint64) {
	ctx.varCount += 2
	ctx.stack = append(ctx.stack, uint32(a), uint32(a>>32))
}

func out4(ctx *Context, a uint128) {
	ctx.varCount += 4
	ctx.stack = append(ctx.stack, U128_0(a), U128_1(a), U128_2(a), U128_3(a))
}

type Context struct {
	stack    []uint32
	scopes   []int
	varCount int
	opCtr    int
	prog     []uint32

	hashIn    []byte
	hashOut   []byte
	memory    []byte
	hashctr   int64
	loopCycle int
}

func branch(ctx *Context, a bool, insn uint32, pc int) int {
	if util.DecodeInsn_imm(insn) != 2 {
		panic("count should be 2")
	}
	if a {
		return interpret(ctx, pc+2)
	}
	return interpret(ctx, pc+1)
}

const debug = false

func pad(ctx *Context) {
	for i := 0; i < len(ctx.scopes)+1; i++ {
		fmt.Printf("  ")
	}
	fmt.Printf(" ")
}

func interpret(ctx *Context, pc int) int {
	// spacing added in RandGen
	if pc != 0 {
		ctx.stack = append(ctx.stack, ^uint32(0))
		ctx.scopes = append(ctx.scopes, ctx.varCount)
		ctx.varCount = 0
	}
	for ; ; pc++ {
		if ctx.opCtr > util.Conf_RandHash_MAX_OPS {
			return -1
		}
		ctx.opCtr++
		insn := ctx.prog[pc]
		op := opcodes.OpCode(util.DecodeInsn_OP(insn))
		if op <= opcodes.OpCode_INVALID_ZERO || op >= opcodes.OpCode_INVALID_BIG {
			panic("op out of range")
		}
		switch op {
		case opcodes.OpCode_MEMORY:
			base := int(util.DecodeInsn_MEMORY_BASE(insn))
			step := int(util.DecodeInsn_MEMORY_STEP(insn))
			carry := int(util.DecodeInsn_MEMORY_CARRY(insn))
			// memory[(base + ((loopVar + carry) * step)) % RandHash_MEMORY_SZ]
			idx := (base + ((ctx.loopCycle + carry) * step)) & (RandHash_MEMORY_SZ - 1)
			hi := binary.LittleEndian.Uint32(ctx.memory[idx*4:][:4])
			if debug {
				pad(ctx)
				fmt.Printf("MEMORY(%d, 0x%08x, %d, %d) -> %08x (%08x)\n",
					ctx.loopCycle, base, step, carry,
					hi, idx)
			}
			out1(ctx, hi)

		case opcodes.OpCode_IN:
			idx := int64(uint32(util.DecodeInsn_imm(insn))) % RandHash_INOUT_SZ
			hi := binary.LittleEndian.Uint32(ctx.hashIn[idx*4:][:4])
			if debug {
				pad(ctx)
				fmt.Printf("IN %d -> %08x\n", idx, hi)
			}
			out1(ctx, hi)

		case opcodes.OpCode_LOOP:
			if debug {

				pad(ctx)
				fmt.Printf("%s (%08x) %d\n", op.String(), insn, pc)
			}
			count := int(util.DecodeInsn_imm(insn))
			ret := pc
			for i := 0; i < count; i++ {
				ctx.loopCycle = i
				ret = interpret(ctx, pc+1)
			}
			if ctx.opCtr > util.Conf_RandHash_MAX_OPS {
				return -1
			}
			pc = ret
			if pc == len(ctx.prog)-1 {
				if len(ctx.stack) != 0 {
					panic("leftover stack")
				}
				if len(ctx.scopes) != 0 {
					panic("leftover scopes")
				}
				if ctx.varCount != 0 {
					panic("varCount not 0")
				}
				return pc
			}

		case opcodes.OpCode_IF_LIKELY:
			if debug {
				pad(ctx)
				fmt.Printf("%s (%08x) %d\n", op.String(), insn, pc)
			}
			pc = branch(ctx, getA(ctx, insn)&7 != 0, insn, pc)

		case opcodes.OpCode_IF_RANDOM:
			if debug {
				pad(ctx)
				fmt.Printf("%s (%08x) %d\n", op.String(), insn, pc)
			}
			pc = branch(ctx, getA(ctx, insn)&1 != 0, insn, pc)

		case opcodes.OpCode_JMP:
			if debug {
				pad(ctx)
				fmt.Printf("%s (%08x) %d\n", op.String(), insn, pc)
			}
			count := (insn >> 8)
			pc += int(count)

		case opcodes.OpCode_END:
			if debug {
				pad(ctx)
				fmt.Printf("%s (%08x) %d\n", op.String(), insn, pc)
			}
			// output everything first
			if len(ctx.stack)-ctx.varCount <= 0 {
				panic("insane varcount")
			}
			// printf("xx %d\n", ctx->vars.count);
			for i := len(ctx.stack) - ctx.varCount; i < len(ctx.stack); i++ {
				// printf("// out1(%08x) %d\n", ctx->vars.elems[i], ctx->hashctr);
				// DEBUGF("out1 %08x (%d)\n", ctx->vars.elems[i], ctx->hashctr);
				if debug {
					pad(ctx)
					fmt.Printf("OUTPUT %08x (%d)\n", ctx.stack[i], ctx.hashctr)
				}
				h := binary.LittleEndian.Uint32(ctx.hashOut[ctx.hashctr*4:][:4])
				h += ctx.stack[i]
				binary.LittleEndian.PutUint32(ctx.hashOut[ctx.hashctr*4:][:4], h)
				ctx.hashctr = (ctx.hashctr + 1) % RandHash_INOUT_SZ
			}
			ctx.stack = ctx.stack[:len(ctx.stack)-ctx.varCount]
			if ctx.stack[len(ctx.stack)-1] != ^uint32(0) {
				fmt.Printf("Stack content: %v", ctx.stack)
				panic("corrupt stack")
			}
			ctx.varCount = ctx.scopes[len(ctx.scopes)-1]

			// pop pop
			ctx.stack = ctx.stack[:len(ctx.stack)-1]
			ctx.scopes = ctx.scopes[:len(ctx.scopes)-1]

			return pc

		default:
			doOp(ctx, insn, op)
		}
	}
}

func Interpret(prog []uint32, ccState, memory []byte, cycles int) er.R {
	if len(memory) < RandHash_MEMORY_SZ*4 {
		panic("memory size too small")
	}
	ctx := Context{}

	ctx.memory = memory
	ctx.hashIn = ccState[:len(ccState)/2]
	ctx.hashOut = ccState[len(ccState)/2:]
	if int64(len(ccState))/4/2 != RandHash_INOUT_SZ {
		panic("weird size")
	}
	ctx.prog = prog

	for i := 0; i < cycles; i++ {
		ctx.opCtr = 0
		interpret(&ctx, 0)

		if ctx.opCtr > util.Conf_RandHash_MAX_OPS {
			return er.New("RandHash_TOO_LONG")
		} else if ctx.opCtr < util.Conf_RandHash_MIN_OPS {
			return er.New("RandHash_TOO_SHORT")
		}

		ctx.hashctr = 0
		ctx.hashOut, ctx.hashIn = ctx.hashIn, ctx.hashOut
	}

	return nil
}

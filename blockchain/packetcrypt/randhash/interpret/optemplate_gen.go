// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package interpret

import (
	"encoding/binary"
	"math/bits"
)

func bint(x bool) int {
	if x {
		return 1
	}
	return 0
}

func add8(a uint8, b uint8) uint8 { return a + b }
func ADD8_16(a uint16, b uint16) uint16 {
	return (uint16(add8(uint8(a>>8), uint8(b>>8))) << 8) | uint16(add8(uint8(a), uint8(b)))
}

func ADD8(a uint32, b uint32) uint32 {
	return (uint32(ADD8_16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(ADD8_16(uint16(a), uint16(b)))
}
func sub8(a uint8, b uint8) uint8 { return a - b }
func SUB8_16(a uint16, b uint16) uint16 {
	return (uint16(sub8(uint8(a>>8), uint8(b>>8))) << 8) | uint16(sub8(uint8(a), uint8(b)))
}

func SUB8(a uint32, b uint32) uint32 {
	return (uint32(SUB8_16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(SUB8_16(uint16(a), uint16(b)))
}

func shll8(a uint8, b uint8) uint8 { return a << (b & 7) }
func SHLL8_16(a uint16, b uint16) uint16 {
	return (uint16(shll8(uint8(a>>8), uint8(b>>8))) << 8) | uint16(shll8(uint8(a), uint8(b)))
}

func SHLL8(a uint32, b uint32) uint32 {
	return (uint32(SHLL8_16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(SHLL8_16(uint16(a), uint16(b)))
}
func shrl8(a uint8, b uint8) uint8 { return a >> (b & 7) }
func SHRL8_16(a uint16, b uint16) uint16 {
	return (uint16(shrl8(uint8(a>>8), uint8(b>>8))) << 8) | uint16(shrl8(uint8(a), uint8(b)))
}

func SHRL8(a uint32, b uint32) uint32 {
	return (uint32(SHRL8_16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(SHRL8_16(uint16(a), uint16(b)))
}
func shra8(a uint8, b uint8) uint8 { return uint8(int8(a) >> (b & 7)) }
func SHRA8_16(a uint16, b uint16) uint16 {
	return (uint16(shra8(uint8(a>>8), uint8(b>>8))) << 8) | uint16(shra8(uint8(a), uint8(b)))
}

func SHRA8(a uint32, b uint32) uint32 {
	return (uint32(SHRA8_16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(SHRA8_16(uint16(a), uint16(b)))
}
func rotl8(a uint8, b uint8) uint8 { return shll8(a, b) | shrl8(a, 8-b) }
func ROTL8_16(a uint16, b uint16) uint16 {
	return (uint16(rotl8(uint8(a>>8), uint8(b>>8))) << 8) | uint16(rotl8(uint8(a), uint8(b)))
}

func ROTL8(a uint32, b uint32) uint32 {
	return (uint32(ROTL8_16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(ROTL8_16(uint16(a), uint16(b)))
}

// func rotr8(a uint8, b uint8) uint8 { return shll8(a, 8-b) | shrl8(a, b) }
// func ROTR8_16(a uint16, b uint16) uint16 {
// 	return (uint16(rotr8(uint8(a>>8), uint8(b>>8))) << 8) | uint16(rotr8(uint8(a), uint8(b)))
// }
// func ROTR8(a uint32, b uint32) uint32 {
// 	return (uint32(ROTR8_16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(ROTR8_16(uint16(a), uint16(b)))
// }

func mul8(a uint8, b uint8) uint8 { return a * b }

func MUL8_16(a uint16, b uint16) uint16 {
	return (uint16(mul8(uint8(a>>8), uint8(b>>8))) << 8) | uint16(mul8(uint8(a), uint8(b)))
}

func MUL8(a uint32, b uint32) uint32 {
	return (uint32(MUL8_16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(MUL8_16(uint16(a), uint16(b)))
}

func add8C(a uint8, b uint8) uint16 { return uint16(a) + uint16(b) }
func ADD8C_16C(a uint16, b uint16) uint32 {
	return (uint32(add8C(uint8(a>>8), uint8(b>>8))) << 16) | uint32(add8C(uint8(a), uint8(b)))
}

func ADD8C(a uint32, b uint32) uint64 {
	return (uint64(ADD8C_16C(uint16(a>>16), uint16(b>>16))) << 32) | uint64(ADD8C_16C(uint16(a), uint16(b)))
}
func sub8C(a uint8, b uint8) uint16 { return uint16(a) - uint16(b) }
func SUB8C_16C(a uint16, b uint16) uint32 {
	return (uint32(sub8C(uint8(a>>8), uint8(b>>8))) << 16) | uint32(sub8C(uint8(a), uint8(b)))
}

func SUB8C(a uint32, b uint32) uint64 {
	return (uint64(SUB8C_16C(uint16(a>>16), uint16(b>>16))) << 32) | uint64(SUB8C_16C(uint16(a), uint16(b)))
}

func mul8C(a uint8, b uint8) uint16 { return uint16(int16(int8(a)) * int16(int8(b))) }
func MUL8C_16C(a uint16, b uint16) uint32 {
	return (uint32(mul8C(uint8(a>>8), uint8(b>>8))) << 16) | uint32(mul8C(uint8(a), uint8(b)))
}

func MUL8C(a uint32, b uint32) uint64 {
	return (uint64(MUL8C_16C(uint16(a>>16), uint16(b>>16))) << 32) | uint64(MUL8C_16C(uint16(a), uint16(b)))
}
func mulu8C(a uint8, b uint8) uint16 { return uint16(a) * uint16(b) }
func MULU8C_16C(a uint16, b uint16) uint32 {
	return (uint32(mulu8C(uint8(a>>8), uint8(b>>8))) << 16) | uint32(mulu8C(uint8(a), uint8(b)))
}

func MULU8C(a uint32, b uint32) uint64 {
	return (uint64(MULU8C_16C(uint16(a>>16), uint16(b>>16))) << 32) | uint64(MULU8C_16C(uint16(a), uint16(b)))
}
func mulsu8C(a uint8, b uint8) uint16 { return uint16(int16(int8(a)) * int16(b)) }
func MULSU8C_16C(a uint16, b uint16) uint32 {
	return (uint32(mulsu8C(uint8(a>>8), uint8(b>>8))) << 16) | uint32(mulsu8C(uint8(a), uint8(b)))
}

func MULSU8C(a uint32, b uint32) uint64 {
	return (uint64(MULSU8C_16C(uint16(a>>16), uint16(b>>16))) << 32) | uint64(MULSU8C_16C(uint16(a), uint16(b)))
}

func add16(a uint16, b uint16) uint16 { return a + b }
func ADD16(a uint32, b uint32) uint32 {
	return (uint32(add16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(add16(uint16(a), uint16(b)))
}
func sub16(a uint16, b uint16) uint16 { return a - b }
func SUB16(a uint32, b uint32) uint32 {
	return (uint32(sub16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(sub16(uint16(a), uint16(b)))
}

func shll16(a uint16, b uint16) uint16 { return a << (b & 15) }
func SHLL16(a uint32, b uint32) uint32 {
	return (uint32(shll16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(shll16(uint16(a), uint16(b)))
}
func shrl16(a uint16, b uint16) uint16 { return a >> (b & 15) }
func SHRL16(a uint32, b uint32) uint32 {
	return (uint32(shrl16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(shrl16(uint16(a), uint16(b)))
}
func shra16(a uint16, b uint16) uint16 { return uint16(int16(a) >> (b & 15)) }
func SHRA16(a uint32, b uint32) uint32 {
	return (uint32(shra16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(shra16(uint16(a), uint16(b)))
}
func rotl16(a uint16, b uint16) uint16 { return shll16(a, b) | shrl16(a, 16-b) }
func ROTL16(a uint32, b uint32) uint32 {
	return (uint32(rotl16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(rotl16(uint16(a), uint16(b)))
}

// func rotr16(a uint16, b uint16) uint16 { return shll16(a, 16-b) | shrl16(a, b) }
// func ROTR16(a uint32, b uint32) uint32 {
// 	return (uint32(rotr16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(rotr16(uint16(a), uint16(b)))
// }

func mul16(a uint16, b uint16) uint16 { return a * b }

func MUL16(a uint32, b uint32) uint32 {
	return (uint32(mul16(uint16(a>>16), uint16(b>>16))) << 16) | uint32(mul16(uint16(a), uint16(b)))
}

func add16C(a uint16, b uint16) uint32 { return uint32(a) + uint32(b) }
func ADD16C(a uint32, b uint32) uint64 {
	return (uint64(add16C(uint16(a>>16), uint16(b>>16))) << 32) | uint64(add16C(uint16(a), uint16(b)))
}
func sub16C(a uint16, b uint16) uint32 { return uint32(a) - uint32(b) }
func SUB16C(a uint32, b uint32) uint64 {
	return (uint64(sub16C(uint16(a>>16), uint16(b>>16))) << 32) | uint64(sub16C(uint16(a), uint16(b)))
}

func mul16C(a uint16, b uint16) uint32 { return uint32(int32(int16(a)) * int32(int16(b))) }
func MUL16C(a uint32, b uint32) uint64 {
	return (uint64(mul16C(uint16(a>>16), uint16(b>>16))) << 32) | uint64(mul16C(uint16(a), uint16(b)))
}
func mulu16C(a uint16, b uint16) uint32 { return uint32(a) * uint32(b) }
func MULU16C(a uint32, b uint32) uint64 {
	return (uint64(mulu16C(uint16(a>>16), uint16(b>>16))) << 32) | uint64(mulu16C(uint16(a), uint16(b)))
}
func mulsu16C(a uint16, b uint16) uint32 { return uint32(int32(int16(a)) * int32(b)) }
func MULSU16C(a uint32, b uint32) uint64 {
	return (uint64(mulsu16C(uint16(a>>16), uint16(b>>16))) << 32) | uint64(mulsu16C(uint16(a), uint16(b)))
}

func add32(a uint32, b uint32) uint32 { return a + b }
func ADD32(a uint32, b uint32) uint32 { return add32(a, b) }
func sub32(a uint32, b uint32) uint32 { return a - b }
func SUB32(a uint32, b uint32) uint32 { return sub32(a, b) }

func shll32(a uint32, b uint32) uint32 { return a << (b & 31) }
func SHLL32(a uint32, b uint32) uint32 { return shll32(a, b) }
func shrl32(a uint32, b uint32) uint32 { return a >> (b & 31) }
func SHRL32(a uint32, b uint32) uint32 { return shrl32(a, b) }
func shra32(a uint32, b uint32) uint32 { return uint32(int32(a) >> (b & 31)) }
func SHRA32(a uint32, b uint32) uint32 { return shra32(a, b) }
func rotl32(a uint32, b uint32) uint32 { return shll32(a, b) | shrl32(a, 32-b) }
func ROTL32(a uint32, b uint32) uint32 { return rotl32(a, b) }

// func rotr32(a uint32, b uint32) uint32 { return shll32(a, 32-b) | shrl32(a, b) }
// func ROTR32(a uint32, b uint32) uint32 { return rotr32(a, b) }

func mul32(a uint32, b uint32) uint32 { return a * b }
func MUL32(a uint32, b uint32) uint32 { return mul32(a, b) }

func add32C(a uint32, b uint32) uint64 { return uint64(a) + uint64(b) }
func ADD32C(a uint32, b uint32) uint64 { return add32C(a, b) }
func sub32C(a uint32, b uint32) uint64 { return uint64(a) - uint64(b) }
func SUB32C(a uint32, b uint32) uint64 { return sub32C(a, b) }

func mul32C(a uint32, b uint32) uint64   { return uint64(int64(int32(a)) * int64(int32(b))) }
func MUL32C(a uint32, b uint32) uint64   { return mul32C(a, b) }
func mulu32C(a uint32, b uint32) uint64  { return uint64(a) * uint64(b) }
func MULU32C(a uint32, b uint32) uint64  { return mulu32C(a, b) }
func mulsu32C(a uint32, b uint32) uint64 { return uint64(int64(int32(a)) * int64(b)) }
func MULSU32C(a uint32, b uint32) uint64 { return mulsu32C(a, b) }

func add64(a uint64, b uint64) uint64 { return a + b }
func ADD64(a0 uint32, a1 uint32, b0 uint32, b1 uint32) uint64 {
	a := uint64(a1)
	a <<= 32
	a |= uint64(a0)
	b := uint64(b1)
	b <<= 32
	b |= uint64(b0)
	return add64(a, b)
}
func sub64(a uint64, b uint64) uint64 { return a - b }
func SUB64(a0 uint32, a1 uint32, b0 uint32, b1 uint32) uint64 {
	a := uint64(a1)
	a <<= 32
	a |= uint64(a0)
	b := uint64(b1)
	b <<= 32
	b |= uint64(b0)
	return sub64(a, b)
}

func shll64(a uint64, b uint64) uint64 { return a << (b & 63) }
func SHLL64(a0 uint32, a1 uint32, b0 uint32, b1 uint32) uint64 {
	a := uint64(a1)
	a <<= 32
	a |= uint64(a0)
	b := uint64(b1)
	b <<= 32
	b |= uint64(b0)
	return shll64(a, b)
}
func shrl64(a uint64, b uint64) uint64 { return a >> (b & 63) }
func SHRL64(a0 uint32, a1 uint32, b0 uint32, b1 uint32) uint64 {
	a := uint64(a1)
	a <<= 32
	a |= uint64(a0)
	b := uint64(b1)
	b <<= 32
	b |= uint64(b0)
	return shrl64(a, b)
}
func shra64(a uint64, b uint64) uint64 { return uint64(int64(a) >> (b & 63)) }
func SHRA64(a0 uint32, a1 uint32, b0 uint32, b1 uint32) uint64 {
	a := uint64(a1)
	a <<= 32
	a |= uint64(a0)
	b := uint64(b1)
	b <<= 32
	b |= uint64(b0)
	return shra64(a, b)
}
func rotl64(a uint64, b uint64) uint64 { return shll64(a, b) | shrl64(a, 64-b) }
func ROTL64(a0 uint32, a1 uint32, b0 uint32, b1 uint32) uint64 {
	a := uint64(a1)
	a <<= 32
	a |= uint64(a0)
	b := uint64(b1)
	b <<= 32
	b |= uint64(b0)
	return rotl64(a, b)
}
func rotr64(a uint64, b uint64) uint64 { return shll64(a, 64-b) | shrl64(a, b) }
func ROTR64(a0 uint32, a1 uint32, b0 uint32, b1 uint32) uint64 {
	a := uint64(a1)
	a <<= 32
	a |= uint64(a0)
	b := uint64(b1)
	b <<= 32
	b |= uint64(b0)
	return rotr64(a, b)
}

func mul64(a uint64, b uint64) uint64 { return a * b }
func MUL64(a0 uint32, a1 uint32, b0 uint32, b1 uint32) uint64 {
	a := uint64(a1)
	a <<= 32
	a |= uint64(a0)
	b := uint64(b1)
	b <<= 32
	b |= uint64(b0)
	return mul64(a, b)
}

type uint128 struct {
	bytes [16]byte
}

func MK128(lo, hi uint64) uint128 {
	out := uint128{}
	binary.LittleEndian.PutUint64(out.bytes[:8], (lo))
	binary.LittleEndian.PutUint64(out.bytes[8:], (hi))
	return out
}
func U128_0(v uint128) uint32 { return binary.LittleEndian.Uint32(v.bytes[:4]) }
func U128_1(v uint128) uint32 { return binary.LittleEndian.Uint32(v.bytes[4:8]) }
func U128_2(v uint128) uint32 { return binary.LittleEndian.Uint32(v.bytes[8:12]) }
func U128_3(v uint128) uint32 { return binary.LittleEndian.Uint32(v.bytes[12:16]) }
func mulhu64(a uint64, b uint64) uint64 {
	a0 := uint32(a)
	a1 := uint32((a >> 32))
	b0 := uint32(b)
	b1 := uint32((b >> 32))
	r00 := uint64((uint64(a0) * uint64(b0)))
	r01 := uint64((uint64(a0) * uint64(b1)))
	r10 := uint64((uint64(a1) * uint64(b0)))
	r11 := uint64((uint64(a1) * uint64(b1)))
	c := uint64(((r00 >> 32) + uint64(uint32(r01)) + uint64(uint32(r10))))
	c = (c >> 32) + (r01 >> 32) + (r10 >> 32) + uint64(uint32(r11))
	r2 := uint32(c)
	r3 := uint32(((c >> 32) + (r11 >> 32)))
	return (uint64(r3) << 32) | uint64(r2)
}

func mulh64(a int64, b int64) uint64 {
	negate := bool(((a < 0) != (b < 0)))
	_a := uint64(a)
	_b := uint64(b)
	{
		if a < 0 {
			_a = uint64(-a)
		}
	}
	{
		if b < 0 {
			_b = uint64(-b)
		}
	}
	res := uint64(mulhu64(_a, _b))
	{
		if negate {
			res = (^(res))
			if a*b == 0 {
				res++
			}
		}
	}
	return res
}

func mulhsu64(a int64, b uint64) uint64 {
	negate := bool((a < 0))
	_a := uint64(a)
	{
		if a < 0 {
			_a = uint64(-a)
		}
	}
	res := uint64(mulhu64(_a, b))
	{
		if negate {
			res = (^(res))
			if uint64(a)*b == 0 {
				res++
			}
		}
	}
	return res
}
func mul64C(a uint64, b uint64) uint128 { return MK128((a * b), mulh64(int64(a), int64(b))) }

func mulsu64C(a uint64, b uint64) uint128 { return MK128((a * b), mulhsu64(int64(a), b)) }

func mulu64C(a uint64, b uint64) uint128 { return MK128((a * b), mulhu64(a, b)) }

func add64C(a uint64, b uint64) uint128 {
	res := uint64((a + b))
	return MK128(res, uint64(bint(res < b)))
}

func ADD64C(a0 uint32, a1 uint32, b0 uint32, b1 uint32) uint128 {
	a := uint64(a1)
	a <<= 32
	a |= uint64(a0)
	b := uint64(b1)
	b <<= 32
	b |= uint64(b0)
	return add64C(a, b)
}

func sub64C(a uint64, b uint64) uint128 {
	res := uint64((a - b))
	return MK128(res, uint64(0)-uint64(bint(a < b)))
}

func SUB64C(a0 uint32, a1 uint32, b0 uint32, b1 uint32) uint128 {
	a := uint64(a1)
	a <<= 32
	a |= uint64(a0)
	b := uint64(b1)
	b <<= 32
	b |= uint64(b0)
	return sub64C(a, b)
}

func MUL64C(a0 uint32, a1 uint32, b0 uint32, b1 uint32) uint128 {
	a := uint64(a1)
	a <<= 32
	a |= uint64(a0)
	b := uint64(b1)
	b <<= 32
	b |= uint64(b0)
	return mul64C(a, b)
}

func MULSU64C(a0 uint32, a1 uint32, b0 uint32, b1 uint32) uint128 {
	a := uint64(a1)
	a <<= 32
	a |= uint64(a0)
	b := uint64(b1)
	b <<= 32
	b |= uint64(b0)
	return mulsu64C(a, b)
}

func MULU64C(a0 uint32, a1 uint32, b0 uint32, b1 uint32) uint128 {
	a := uint64(a1)
	a <<= 32
	a |= uint64(a0)
	b := uint64(b1)
	b <<= 32
	b |= uint64(b0)
	return mulu64C(a, b)
}

func XOR(a uint32, b uint32) uint32 { return a ^ b }
func OR(a uint32, b uint32) uint32  { return a | b }
func AND(a uint32, b uint32) uint32 { return a & b }
func popcnt8(a uint8) uint8         { return uint8(bits.OnesCount8(a)) }
func POPCNT8_16(a uint16) uint16 {
	return (uint16(popcnt8(uint8(a>>8))) << 8) | uint16(popcnt8(uint8(a)))
}

func POPCNT8(a uint32) uint32 {
	return (uint32(POPCNT8_16(uint16(a>>16))) << 16) | uint32(POPCNT8_16(uint16(a)))
}
func clz8(a uint8) uint8      { return uint8(bits.LeadingZeros8(a)) }
func CLZ8_16(a uint16) uint16 { return (uint16(clz8(uint8(a>>8))) << 8) | uint16(clz8(uint8(a))) }
func CLZ8(a uint32) uint32 {
	return (uint32(CLZ8_16(uint16(a>>16))) << 16) | uint32(CLZ8_16(uint16(a)))
}

// func bswap8(a uint8) uint8      { return a }
// func BSWAP8_16(a uint16) uint16 { return (uint16(bswap8(uint8(a>>8))) << 8) | uint16(bswap8(uint8(a))) }
// func BSWAP8(a uint32) uint32 {
// 	return (uint32(BSWAP8_16(uint16(a>>16))) << 16) | uint32(BSWAP8_16(uint16(a)))
// }
func ctz8(a uint8) uint8      { return uint8(bits.TrailingZeros8(a)) }
func CTZ8_16(a uint16) uint16 { return (uint16(ctz8(uint8(a>>8))) << 8) | uint16(ctz8(uint8(a))) }
func CTZ8(a uint32) uint32 {
	return (uint32(CTZ8_16(uint16(a>>16))) << 16) | uint32(CTZ8_16(uint16(a)))
}

func popcnt16(a uint16) uint16 { return uint16(bits.OnesCount16(a)) }
func POPCNT16(a uint32) uint32 {
	return (uint32(popcnt16(uint16(a>>16))) << 16) | uint32(popcnt16(uint16(a)))
}
func clz16(a uint16) uint16   { return uint16(bits.LeadingZeros16(a)) }
func CLZ16(a uint32) uint32   { return (uint32(clz16(uint16(a>>16))) << 16) | uint32(clz16(uint16(a))) }
func bswap16(a uint16) uint16 { return bits.ReverseBytes16(a) }
func BSWAP16(a uint32) uint32 {
	return (uint32(bswap16(uint16(a>>16))) << 16) | uint32(bswap16(uint16(a)))
}
func ctz16(a uint16) uint16 { return uint16(bits.TrailingZeros16(a)) }
func CTZ16(a uint32) uint32 { return (uint32(ctz16(uint16(a>>16))) << 16) | uint32(ctz16(uint16(a))) }

func popcnt32(a uint32) uint32 { return uint32(bits.OnesCount32(a)) }
func POPCNT32(a uint32) uint32 { return popcnt32(a) }
func clz32(a uint32) uint32    { return uint32(bits.LeadingZeros32(a)) }
func CLZ32(a uint32) uint32    { return clz32(a) }
func bswap32(a uint32) uint32  { return bits.ReverseBytes32(a) }
func BSWAP32(a uint32) uint32  { return bswap32(a) }
func ctz32(a uint32) uint32    { return uint32(bits.TrailingZeros32(a)) }
func CTZ32(a uint32) uint32    { return ctz32(a) }

// func popcnt64(a uint64) uint64           { return uint64(bits.OnesCount64(a)) }
// func POPCNT64(a uint32, b uint32) uint64 { return popcnt64((uint64(b) << 32) | uint64(a)) }
// func clz64(a uint64) uint64              { return uint64(bits.LeadingZeros64(a)) }
// func CLZ64(a uint32, b uint32) uint64    { return clz64((uint64(b) << 32) | uint64(a)) }
// func bswap64(a uint64) uint64            { return bits.ReverseBytes64(a) }
// func BSWAP64(a uint32, b uint32) uint64  { return bswap64((uint64(b) << 32) | uint64(a)) }
// func ctz64(a uint64) uint64              { return uint64(bits.TrailingZeros64(a)) }
// func CTZ64(a uint32, b uint32) uint64    { return ctz64((uint64(b) << 32) | uint64(a)) }

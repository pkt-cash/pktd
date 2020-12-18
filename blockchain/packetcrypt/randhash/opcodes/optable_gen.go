// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package opcodes

type OpCode byte

const (
	OpCode_INVALID_ZERO = iota

	OpCode_POPCNT8  = iota
	OpCode_POPCNT16 = iota
	OpCode_POPCNT32 = iota
	OpCode_CLZ8     = iota
	OpCode_CLZ16    = iota
	OpCode_CLZ32    = iota
	OpCode_CTZ8     = iota
	OpCode_CTZ16    = iota
	OpCode_CTZ32    = iota

	OpCode_BSWAP16 = iota
	OpCode_BSWAP32 = iota

	OpCode_ADD8   = iota
	OpCode_ADD16  = iota
	OpCode_ADD32  = iota
	OpCode_SUB8   = iota
	OpCode_SUB16  = iota
	OpCode_SUB32  = iota
	OpCode_SHLL8  = iota
	OpCode_SHLL16 = iota
	OpCode_SHLL32 = iota
	OpCode_SHRL8  = iota
	OpCode_SHRL16 = iota
	OpCode_SHRL32 = iota
	OpCode_SHRA8  = iota
	OpCode_SHRA16 = iota
	OpCode_SHRA32 = iota
	OpCode_ROTL8  = iota
	OpCode_ROTL16 = iota
	OpCode_ROTL32 = iota
	OpCode_MUL8   = iota
	OpCode_MUL16  = iota
	OpCode_MUL32  = iota

	OpCode_AND = iota
	OpCode_OR  = iota
	OpCode_XOR = iota

	OpCode_ADD8C    = iota
	OpCode_ADD16C   = iota
	OpCode_ADD32C   = iota
	OpCode_SUB8C    = iota
	OpCode_SUB16C   = iota
	OpCode_SUB32C   = iota
	OpCode_MUL8C    = iota
	OpCode_MUL16C   = iota
	OpCode_MUL32C   = iota
	OpCode_MULSU8C  = iota
	OpCode_MULSU16C = iota
	OpCode_MULSU32C = iota
	OpCode_MULU8C   = iota
	OpCode_MULU16C  = iota
	OpCode_MULU32C  = iota

	OpCode_ADD64  = iota
	OpCode_SUB64  = iota
	OpCode_SHLL64 = iota
	OpCode_SHRL64 = iota
	OpCode_SHRA64 = iota
	OpCode_ROTL64 = iota
	OpCode_ROTR64 = iota
	OpCode_MUL64  = iota

	OpCode_ADD64C   = iota
	OpCode_SUB64C   = iota
	OpCode_MUL64C   = iota
	OpCode_MULSU64C = iota
	OpCode_MULU64C  = iota

	OpCode_IN     = iota
	OpCode_MEMORY = iota

	OpCode_LOOP      = iota
	OpCode_IF_LIKELY = iota
	OpCode_IF_RANDOM = iota
	OpCode_JMP       = iota
	OpCode_END       = iota

	OpCode_INVALID_BIG = iota
)

func (o OpCode) String() string {
	switch o {

	case OpCode_INVALID_ZERO:
		return "INVALID_ZERO"

	case OpCode_POPCNT8:
		return "POPCNT8"
	case OpCode_POPCNT16:
		return "POPCNT16"
	case OpCode_POPCNT32:
		return "POPCNT32"
	case OpCode_CLZ8:
		return "CLZ8"
	case OpCode_CLZ16:
		return "CLZ16"
	case OpCode_CLZ32:
		return "CLZ32"
	case OpCode_CTZ8:
		return "CTZ8"
	case OpCode_CTZ16:
		return "CTZ16"
	case OpCode_CTZ32:
		return "CTZ32"

	case OpCode_BSWAP16:
		return "BSWAP16"
	case OpCode_BSWAP32:
		return "BSWAP32"

	case OpCode_ADD8:
		return "ADD8"
	case OpCode_ADD16:
		return "ADD16"
	case OpCode_ADD32:
		return "ADD32"
	case OpCode_SUB8:
		return "SUB8"
	case OpCode_SUB16:
		return "SUB16"
	case OpCode_SUB32:
		return "SUB32"
	case OpCode_SHLL8:
		return "SHLL8"
	case OpCode_SHLL16:
		return "SHLL16"
	case OpCode_SHLL32:
		return "SHLL32"
	case OpCode_SHRL8:
		return "SHRL8"
	case OpCode_SHRL16:
		return "SHRL16"
	case OpCode_SHRL32:
		return "SHRL32"
	case OpCode_SHRA8:
		return "SHRA8"
	case OpCode_SHRA16:
		return "SHRA16"
	case OpCode_SHRA32:
		return "SHRA32"
	case OpCode_ROTL8:
		return "ROTL8"
	case OpCode_ROTL16:
		return "ROTL16"
	case OpCode_ROTL32:
		return "ROTL32"
	case OpCode_MUL8:
		return "MUL8"
	case OpCode_MUL16:
		return "MUL16"
	case OpCode_MUL32:
		return "MUL32"

	case OpCode_AND:
		return "AND"
	case OpCode_OR:
		return "OR"
	case OpCode_XOR:
		return "XOR"

	case OpCode_ADD8C:
		return "ADD8C"
	case OpCode_ADD16C:
		return "ADD16C"
	case OpCode_ADD32C:
		return "ADD32C"
	case OpCode_SUB8C:
		return "SUB8C"
	case OpCode_SUB16C:
		return "SUB16C"
	case OpCode_SUB32C:
		return "SUB32C"
	case OpCode_MUL8C:
		return "MUL8C"
	case OpCode_MUL16C:
		return "MUL16C"
	case OpCode_MUL32C:
		return "MUL32C"
	case OpCode_MULSU8C:
		return "MULSU8C"
	case OpCode_MULSU16C:
		return "MULSU16C"
	case OpCode_MULSU32C:
		return "MULSU32C"
	case OpCode_MULU8C:
		return "MULU8C"
	case OpCode_MULU16C:
		return "MULU16C"
	case OpCode_MULU32C:
		return "MULU32C"

	case OpCode_ADD64:
		return "ADD64"
	case OpCode_SUB64:
		return "SUB64"
	case OpCode_SHLL64:
		return "SHLL64"
	case OpCode_SHRL64:
		return "SHRL64"
	case OpCode_SHRA64:
		return "SHRA64"
	case OpCode_ROTL64:
		return "ROTL64"
	case OpCode_ROTR64:
		return "ROTR64"
	case OpCode_MUL64:
		return "MUL64"

	case OpCode_ADD64C:
		return "ADD64C"
	case OpCode_SUB64C:
		return "SUB64C"
	case OpCode_MUL64C:
		return "MUL64C"
	case OpCode_MULSU64C:
		return "MULSU64C"
	case OpCode_MULU64C:
		return "MULU64C"

	case OpCode_IN:
		return "IN"
	case OpCode_MEMORY:
		return "MEMORY"

	case OpCode_LOOP:
		return "LOOP"
	case OpCode_IF_LIKELY:
		return "IF_LIKELY"
	case OpCode_IF_RANDOM:
		return "IF_RANDOM"
	case OpCode_JMP:
		return "JMP"
	case OpCode_END:
		return "END"

	case OpCode_INVALID_BIG:
		return "INVALID_BIG"
	default:
		return "unknown"
	}
}

var CODES_1_1 = [...]OpCode{

	OpCode_POPCNT8, OpCode_POPCNT16, OpCode_POPCNT32,
	OpCode_CLZ8, OpCode_CLZ16, OpCode_CLZ32,
	OpCode_CTZ8, OpCode_CTZ16, OpCode_CTZ32,

	OpCode_BSWAP16,
	OpCode_BSWAP32,
}

var CODES_2_1 = [...]OpCode{

	OpCode_ADD8, OpCode_ADD16, OpCode_ADD32,
	OpCode_SUB8, OpCode_SUB16, OpCode_SUB32,
	OpCode_SHLL8, OpCode_SHLL16, OpCode_SHLL32,
	OpCode_SHRL8, OpCode_SHRL16, OpCode_SHRL32,
	OpCode_SHRA8, OpCode_SHRA16, OpCode_SHRA32,
	OpCode_ROTL8, OpCode_ROTL16, OpCode_ROTL32,
	OpCode_MUL8, OpCode_MUL16, OpCode_MUL32,

	OpCode_AND,
	OpCode_OR,
	OpCode_XOR,
}

var CODES_2_2 = [...]OpCode{

	OpCode_ADD8C, OpCode_ADD16C, OpCode_ADD32C,
	OpCode_SUB8C, OpCode_SUB16C, OpCode_SUB32C,
	OpCode_MUL8C, OpCode_MUL16C, OpCode_MUL32C,
	OpCode_MULSU8C, OpCode_MULSU16C, OpCode_MULSU32C,
	OpCode_MULU8C, OpCode_MULU16C, OpCode_MULU32C,
}

var CODES_4_2 = [...]OpCode{

	OpCode_ADD64,
	OpCode_SUB64,
	OpCode_SHLL64,
	OpCode_SHRL64,
	OpCode_SHRA64,
	OpCode_ROTL64,
	OpCode_ROTR64,
	OpCode_MUL64,
}

var CODES_4_4 = [...]OpCode{

	OpCode_ADD64C,
	OpCode_SUB64C,
	OpCode_MUL64C,
	OpCode_MULSU64C,
	OpCode_MULU64C,
}

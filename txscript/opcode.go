// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/pkt-cash/pktd/btcutil/sha1"

	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/txscript/opcode"
	"github.com/pkt-cash/pktd/txscript/params"
	"github.com/pkt-cash/pktd/txscript/parsescript"
	"github.com/pkt-cash/pktd/txscript/scriptnum"
	"github.com/pkt-cash/pktd/txscript/txscripterr"
	"github.com/pkt-cash/pktd/wire/constants"

	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
)

// Conditional execution constants.
const (
	OpCondFalse = 0
	OpCondTrue  = 1
	OpCondSkip  = 2
)

func executeOp(po *parsescript.ParsedOpcode, e *Engine) er.R {
	switch po.Opcode.Value {
	case opcode.OP_FALSE:
		return opcodeFalse(po, e)

	case opcode.OP_DATA_1:
		return opcodePushData(po, e)
	case opcode.OP_DATA_2:
		return opcodePushData(po, e)
	case opcode.OP_DATA_3:
		return opcodePushData(po, e)
	case opcode.OP_DATA_4:
		return opcodePushData(po, e)
	case opcode.OP_DATA_5:
		return opcodePushData(po, e)
	case opcode.OP_DATA_6:
		return opcodePushData(po, e)
	case opcode.OP_DATA_7:
		return opcodePushData(po, e)
	case opcode.OP_DATA_8:
		return opcodePushData(po, e)
	case opcode.OP_DATA_9:
		return opcodePushData(po, e)
	case opcode.OP_DATA_10:
		return opcodePushData(po, e)
	case opcode.OP_DATA_11:
		return opcodePushData(po, e)
	case opcode.OP_DATA_12:
		return opcodePushData(po, e)
	case opcode.OP_DATA_13:
		return opcodePushData(po, e)
	case opcode.OP_DATA_14:
		return opcodePushData(po, e)
	case opcode.OP_DATA_15:
		return opcodePushData(po, e)
	case opcode.OP_DATA_16:
		return opcodePushData(po, e)
	case opcode.OP_DATA_17:
		return opcodePushData(po, e)
	case opcode.OP_DATA_18:
		return opcodePushData(po, e)
	case opcode.OP_DATA_19:
		return opcodePushData(po, e)
	case opcode.OP_DATA_20:
		return opcodePushData(po, e)
	case opcode.OP_DATA_21:
		return opcodePushData(po, e)
	case opcode.OP_DATA_22:
		return opcodePushData(po, e)
	case opcode.OP_DATA_23:
		return opcodePushData(po, e)
	case opcode.OP_DATA_24:
		return opcodePushData(po, e)
	case opcode.OP_DATA_25:
		return opcodePushData(po, e)
	case opcode.OP_DATA_26:
		return opcodePushData(po, e)
	case opcode.OP_DATA_27:
		return opcodePushData(po, e)
	case opcode.OP_DATA_28:
		return opcodePushData(po, e)
	case opcode.OP_DATA_29:
		return opcodePushData(po, e)
	case opcode.OP_DATA_30:
		return opcodePushData(po, e)
	case opcode.OP_DATA_31:
		return opcodePushData(po, e)
	case opcode.OP_DATA_32:
		return opcodePushData(po, e)
	case opcode.OP_DATA_33:
		return opcodePushData(po, e)
	case opcode.OP_DATA_34:
		return opcodePushData(po, e)
	case opcode.OP_DATA_35:
		return opcodePushData(po, e)
	case opcode.OP_DATA_36:
		return opcodePushData(po, e)
	case opcode.OP_DATA_37:
		return opcodePushData(po, e)
	case opcode.OP_DATA_38:
		return opcodePushData(po, e)
	case opcode.OP_DATA_39:
		return opcodePushData(po, e)
	case opcode.OP_DATA_40:
		return opcodePushData(po, e)
	case opcode.OP_DATA_41:
		return opcodePushData(po, e)
	case opcode.OP_DATA_42:
		return opcodePushData(po, e)
	case opcode.OP_DATA_43:
		return opcodePushData(po, e)
	case opcode.OP_DATA_44:
		return opcodePushData(po, e)
	case opcode.OP_DATA_45:
		return opcodePushData(po, e)
	case opcode.OP_DATA_46:
		return opcodePushData(po, e)
	case opcode.OP_DATA_47:
		return opcodePushData(po, e)
	case opcode.OP_DATA_48:
		return opcodePushData(po, e)
	case opcode.OP_DATA_49:
		return opcodePushData(po, e)
	case opcode.OP_DATA_50:
		return opcodePushData(po, e)
	case opcode.OP_DATA_51:
		return opcodePushData(po, e)
	case opcode.OP_DATA_52:
		return opcodePushData(po, e)
	case opcode.OP_DATA_53:
		return opcodePushData(po, e)
	case opcode.OP_DATA_54:
		return opcodePushData(po, e)
	case opcode.OP_DATA_55:
		return opcodePushData(po, e)
	case opcode.OP_DATA_56:
		return opcodePushData(po, e)
	case opcode.OP_DATA_57:
		return opcodePushData(po, e)
	case opcode.OP_DATA_58:
		return opcodePushData(po, e)
	case opcode.OP_DATA_59:
		return opcodePushData(po, e)
	case opcode.OP_DATA_60:
		return opcodePushData(po, e)
	case opcode.OP_DATA_61:
		return opcodePushData(po, e)
	case opcode.OP_DATA_62:
		return opcodePushData(po, e)
	case opcode.OP_DATA_63:
		return opcodePushData(po, e)
	case opcode.OP_DATA_64:
		return opcodePushData(po, e)
	case opcode.OP_DATA_65:
		return opcodePushData(po, e)
	case opcode.OP_DATA_66:
		return opcodePushData(po, e)
	case opcode.OP_DATA_67:
		return opcodePushData(po, e)
	case opcode.OP_DATA_68:
		return opcodePushData(po, e)
	case opcode.OP_DATA_69:
		return opcodePushData(po, e)
	case opcode.OP_DATA_70:
		return opcodePushData(po, e)
	case opcode.OP_DATA_71:
		return opcodePushData(po, e)
	case opcode.OP_DATA_72:
		return opcodePushData(po, e)
	case opcode.OP_DATA_73:
		return opcodePushData(po, e)
	case opcode.OP_DATA_74:
		return opcodePushData(po, e)
	case opcode.OP_DATA_75:
		return opcodePushData(po, e)
	case opcode.OP_PUSHDATA1:
		return opcodePushData(po, e)
	case opcode.OP_PUSHDATA2:
		return opcodePushData(po, e)
	case opcode.OP_PUSHDATA4:
		return opcodePushData(po, e)
	case opcode.OP_1NEGATE:
		return opcode1Negate(po, e)
	case opcode.OP_RESERVED:
		return opcodeReserved(po, e)
	case opcode.OP_TRUE:
		return opcodeN(po, e)
	case opcode.OP_2:
		return opcodeN(po, e)
	case opcode.OP_3:
		return opcodeN(po, e)
	case opcode.OP_4:
		return opcodeN(po, e)
	case opcode.OP_5:
		return opcodeN(po, e)
	case opcode.OP_6:
		return opcodeN(po, e)
	case opcode.OP_7:
		return opcodeN(po, e)
	case opcode.OP_8:
		return opcodeN(po, e)
	case opcode.OP_9:
		return opcodeN(po, e)
	case opcode.OP_10:
		return opcodeN(po, e)
	case opcode.OP_11:
		return opcodeN(po, e)
	case opcode.OP_12:
		return opcodeN(po, e)
	case opcode.OP_13:
		return opcodeN(po, e)
	case opcode.OP_14:
		return opcodeN(po, e)
	case opcode.OP_15:
		return opcodeN(po, e)
	case opcode.OP_16:
		return opcodeN(po, e)

	// Control opcodes.
	case opcode.OP_NOP:
		return opcodeNop(po, e)
	case opcode.OP_VER:
		return opcodeReserved(po, e)
	case opcode.OP_IF:
		return opcodeIf(po, e)
	case opcode.OP_NOTIF:
		return opcodeNotIf(po, e)
	case opcode.OP_VERIF:
		return opcodeReserved(po, e)
	case opcode.OP_VOTE:
		return opcode2Drop(po, e)
	case opcode.OP_ELSE:
		return opcodeElse(po, e)
	case opcode.OP_ENDIF:
		return opcodeEndif(po, e)
	case opcode.OP_VERIFY:
		return opcodeVerify(po, e)
	case opcode.OP_RETURN:
		return opcodeReturn(po, e)
	case opcode.OP_CHECKLOCKTIMEVERIFY:
		return opcodeCheckLockTimeVerify(po, e)
	case opcode.OP_CHECKSEQUENCEVERIFY:
		return opcodeCheckSequenceVerify(po, e)

	// Stack opcodes.
	case opcode.OP_TOALTSTACK:
		return opcodeToAltStack(po, e)
	case opcode.OP_FROMALTSTACK:
		return opcodeFromAltStack(po, e)
	case opcode.OP_2DROP:
		return opcode2Drop(po, e)
	case opcode.OP_2DUP:
		return opcode2Dup(po, e)
	case opcode.OP_3DUP:
		return opcode3Dup(po, e)
	case opcode.OP_2OVER:
		return opcode2Over(po, e)
	case opcode.OP_2ROT:
		return opcode2Rot(po, e)
	case opcode.OP_2SWAP:
		return opcode2Swap(po, e)
	case opcode.OP_IFDUP:
		return opcodeIfDup(po, e)
	case opcode.OP_DEPTH:
		return opcodeDepth(po, e)
	case opcode.OP_DROP:
		return opcodeDrop(po, e)
	case opcode.OP_DUP:
		return opcodeDup(po, e)
	case opcode.OP_NIP:
		return opcodeNip(po, e)
	case opcode.OP_OVER:
		return opcodeOver(po, e)
	case opcode.OP_PICK:
		return opcodePick(po, e)
	case opcode.OP_ROLL:
		return opcodeRoll(po, e)
	case opcode.OP_ROT:
		return opcodeRot(po, e)
	case opcode.OP_SWAP:
		return opcodeSwap(po, e)
	case opcode.OP_TUCK:
		return opcodeTuck(po, e)

	// Splice opcodes.
	case opcode.OP_CAT:
		return opcodeDisabled(po, e)
	case opcode.OP_SUBSTR:
		return opcodeDisabled(po, e)
	case opcode.OP_LEFT:
		return opcodeDisabled(po, e)
	case opcode.OP_RIGHT:
		return opcodeDisabled(po, e)
	case opcode.OP_SIZE:
		return opcodeSize(po, e)

	// Bitwise logic opcodes.
	case opcode.OP_INVERT:
		return opcodeDisabled(po, e)
	case opcode.OP_AND:
		return opcodeDisabled(po, e)
	case opcode.OP_OR:
		return opcodeDisabled(po, e)
	case opcode.OP_XOR:
		return opcodeDisabled(po, e)
	case opcode.OP_EQUAL:
		return opcodeEqual(po, e)
	case opcode.OP_EQUALVERIFY:
		return opcodeEqualVerify(po, e)
	case opcode.OP_RESERVED1:
		return opcodeReserved(po, e)
	case opcode.OP_RESERVED2:
		return opcodeReserved(po, e)

	// Numeric related opcodes.
	case opcode.OP_1ADD:
		return opcode1Add(po, e)
	case opcode.OP_1SUB:
		return opcode1Sub(po, e)
	case opcode.OP_2MUL:
		return opcodeDisabled(po, e)
	case opcode.OP_2DIV:
		return opcodeDisabled(po, e)
	case opcode.OP_NEGATE:
		return opcodeNegate(po, e)
	case opcode.OP_ABS:
		return opcodeAbs(po, e)
	case opcode.OP_NOT:
		return opcodeNot(po, e)
	case opcode.OP_0NOTEQUAL:
		return opcode0NotEqual(po, e)
	case opcode.OP_ADD:
		return opcodeAdd(po, e)
	case opcode.OP_SUB:
		return opcodeSub(po, e)
	case opcode.OP_MUL:
		return opcodeDisabled(po, e)
	case opcode.OP_DIV:
		return opcodeDisabled(po, e)
	case opcode.OP_MOD:
		return opcodeDisabled(po, e)
	case opcode.OP_LSHIFT:
		return opcodeDisabled(po, e)
	case opcode.OP_RSHIFT:
		return opcodeDisabled(po, e)
	case opcode.OP_BOOLAND:
		return opcodeBoolAnd(po, e)
	case opcode.OP_BOOLOR:
		return opcodeBoolOr(po, e)
	case opcode.OP_NUMEQUAL:
		return opcodeNumEqual(po, e)
	case opcode.OP_NUMEQUALVERIFY:
		return opcodeNumEqualVerify(po, e)
	case opcode.OP_NUMNOTEQUAL:
		return opcodeNumNotEqual(po, e)
	case opcode.OP_LESSTHAN:
		return opcodeLessThan(po, e)
	case opcode.OP_GREATERTHAN:
		return opcodeGreaterThan(po, e)
	case opcode.OP_LESSTHANOREQUAL:
		return opcodeLessThanOrEqual(po, e)
	case opcode.OP_GREATERTHANOREQUAL:
		return opcodeGreaterThanOrEqual(po, e)
	case opcode.OP_MIN:
		return opcodeMin(po, e)
	case opcode.OP_MAX:
		return opcodeMax(po, e)
	case opcode.OP_WITHIN:
		return opcodeWithin(po, e)

	// Crypto opcodes.
	case opcode.OP_RIPEMD160:
		return opcodeRipemd160(po, e)
	case opcode.OP_SHA1:
		return opcodeSha1(po, e)
	case opcode.OP_SHA256:
		return opcodeSha256(po, e)
	case opcode.OP_HASH160:
		return opcodeHash160(po, e)
	case opcode.OP_HASH256:
		return opcodeHash256(po, e)
	case opcode.OP_CODESEPARATOR:
		return opcodeCodeSeparator(po, e)
	case opcode.OP_CHECKSIG:
		return opcodeCheckSig(po, e)
	case opcode.OP_CHECKSIGVERIFY:
		return opcodeCheckSigVerify(po, e)
	case opcode.OP_CHECKMULTISIG:
		return opcodeCheckMultiSig(po, e)
	case opcode.OP_CHECKMULTISIGVERIFY:
		return opcodeCheckMultiSigVerify(po, e)

	// Reserved opcodes.
	case opcode.OP_NOP1:
		return opcodeNop(po, e)
	case opcode.OP_NOP4:
		return opcodeNop(po, e)
	case opcode.OP_NOP5:
		return opcodeNop(po, e)
	case opcode.OP_NOP6:
		return opcodeNop(po, e)
	case opcode.OP_NOP7:
		return opcodeNop(po, e)
	case opcode.OP_NOP8:
		return opcodeNop(po, e)
	case opcode.OP_NOP9:
		return opcodeNop(po, e)
	case opcode.OP_NOP10:
		return opcodeNop(po, e)

	// Undefined opcodes.
	case opcode.OP_UNKNOWN186:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN187:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN188:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN189:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN190:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN191:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN192:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN193:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN194:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN195:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN196:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN197:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN198:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN199:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN200:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN201:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN202:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN203:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN204:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN205:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN206:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN207:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN208:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN209:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN210:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN211:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN212:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN213:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN214:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN215:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN216:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN217:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN218:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN219:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN220:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN221:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN222:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN223:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN224:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN225:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN226:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN227:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN228:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN229:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN230:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN231:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN232:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN233:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN234:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN235:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN236:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN237:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN238:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN239:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN240:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN241:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN242:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN243:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN244:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN245:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN246:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN247:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN248:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN249:
		return opcodeInvalid(po, e)

	// Bitcoin Core internal use opcode.  Defined here for completeness.
	case opcode.OP_SMALLINTEGER:
		return opcodeInvalid(po, e)
	case opcode.OP_PUBKEYS:
		return opcodeInvalid(po, e)
	case opcode.OP_UNKNOWN252:
		return opcodeInvalid(po, e)
	case opcode.OP_PUBKEYHASH:
		return opcodeInvalid(po, e)
	case opcode.OP_PUBKEY:
		return opcodeInvalid(po, e)

	case opcode.OP_INVALIDOPCODE:
		return opcodeInvalid(po, e)
	}

	// Should never happen
	return opcodeInvalid(po, e)
}

// opcodeOnelineRepls defines opcode names which are replaced when doing a
// one-line disassembly.  This is done to match the output of the reference
// implementation while not changing the opcode names in the nicer full
// disassembly.
var opcodeOnelineRepls = map[string]string{
	"OP_1NEGATE": "-1",
	"OP_0":       "0",
	"OP_1":       "1",
	"OP_2":       "2",
	"OP_3":       "3",
	"OP_4":       "4",
	"OP_5":       "5",
	"OP_6":       "6",
	"OP_7":       "7",
	"OP_8":       "8",
	"OP_9":       "9",
	"OP_10":      "10",
	"OP_11":      "11",
	"OP_12":      "12",
	"OP_13":      "13",
	"OP_14":      "14",
	"OP_15":      "15",
	"OP_16":      "16",
}

// popIsDisabled returns whether or not the opcode is disabled and thus is always
// bad to see in the instruction stream (even if turned off by a conditional).
func popIsDisabled(pop *parsescript.ParsedOpcode) bool {
	switch pop.Opcode.Value {
	case opcode.OP_CAT:
		return true
	case opcode.OP_SUBSTR:
		return true
	case opcode.OP_LEFT:
		return true
	case opcode.OP_RIGHT:
		return true
	case opcode.OP_INVERT:
		return true
	case opcode.OP_AND:
		return true
	case opcode.OP_OR:
		return true
	case opcode.OP_XOR:
		return true
	case opcode.OP_2MUL:
		return true
	case opcode.OP_2DIV:
		return true
	case opcode.OP_MUL:
		return true
	case opcode.OP_DIV:
		return true
	case opcode.OP_MOD:
		return true
	case opcode.OP_LSHIFT:
		return true
	case opcode.OP_RSHIFT:
		return true
	default:
		return false
	}
}

// popAlwaysIllegal returns whether or not the opcode is always illegal when passed
// over by the program counter even if in a non-executed branch (it isn't a
// coincidence that they are conditionals).
func popAlwaysIllegal(pop *parsescript.ParsedOpcode) bool {
	switch pop.Opcode.Value {
	case opcode.OP_VERIF:
		return true
	default:
		return false
	}
}

// popIsConditional returns whether or not the opcode is a conditional opcode which
// changes the conditional execution stack when executed.
func popIsConditional(pop *parsescript.ParsedOpcode) bool {
	switch pop.Opcode.Value {
	case opcode.OP_IF:
		return true
	case opcode.OP_NOTIF:
		return true
	case opcode.OP_ELSE:
		return true
	case opcode.OP_ENDIF:
		return true
	default:
		return false
	}
}

// popCheckMinimalDataPush returns whether or not the current data push uses the
// smallest possible opcode to represent it.  For example, the value 15 could
// be pushed with OP_DATA_1 15 (among other variations); however, OP_15 is a
// single opcode that represents the same value and is only a single byte versus
// two bytes.
func popCheckMinimalDataPush(pop *parsescript.ParsedOpcode) er.R {
	data := pop.Data
	dataLen := len(data)
	op := pop.Opcode.Value

	if dataLen == 0 && op != opcode.OP_0 {
		str := fmt.Sprintf("zero length data push is encoded with "+
			"opcode %s instead of OP_0", opcode.OpcodeName(pop.Opcode.Value))
		return txscripterr.ScriptError(txscripterr.ErrMinimalData, str)
	} else if dataLen == 1 && data[0] >= 1 && data[0] <= 16 {
		if op != opcode.OP_1+data[0]-1 {
			// Should have used OP_1 .. OP_16
			str := fmt.Sprintf("data push of the value %d encoded "+
				"with opcode %s instead of OP_%d", data[0],
				opcode.OpcodeName(pop.Opcode.Value), data[0])
			return txscripterr.ScriptError(txscripterr.ErrMinimalData, str)
		}
	} else if dataLen == 1 && data[0] == 0x81 {
		if op != opcode.OP_1NEGATE {
			str := fmt.Sprintf("data push of the value -1 encoded "+
				"with opcode %s instead of OP_1NEGATE",
				opcode.OpcodeName(pop.Opcode.Value))
			return txscripterr.ScriptError(txscripterr.ErrMinimalData, str)
		}
	} else if dataLen <= 75 {
		if int(op) != dataLen {
			// Should have used a direct push
			str := fmt.Sprintf("data push of %d bytes encoded "+
				"with opcode %s instead of OP_DATA_%d", dataLen,
				opcode.OpcodeName(pop.Opcode.Value), dataLen)
			return txscripterr.ScriptError(txscripterr.ErrMinimalData, str)
		}
	} else if dataLen <= 255 {
		if op != opcode.OP_PUSHDATA1 {
			str := fmt.Sprintf("data push of %d bytes encoded "+
				"with opcode %s instead of OP_PUSHDATA1",
				dataLen, opcode.OpcodeName(pop.Opcode.Value))
			return txscripterr.ScriptError(txscripterr.ErrMinimalData, str)
		}
	} else if dataLen <= 65535 {
		if op != opcode.OP_PUSHDATA2 {
			str := fmt.Sprintf("data push of %d bytes encoded "+
				"with opcode %s instead of OP_PUSHDATA2",
				dataLen, opcode.OpcodeName(pop.Opcode.Value))
			return txscripterr.ScriptError(txscripterr.ErrMinimalData, str)
		}
	}
	return nil
}

// popPrint returns a human-readable string representation of the opcode for use
// in script disassembly.
func popPrint(pop *parsescript.ParsedOpcode, oneline bool) string {
	// The reference implementation one-line disassembly replaces opcodes
	// which represent values (e.g. OP_0 through OP_16 and OP_1NEGATE)
	// with the raw value.  However, when not doing a one-line dissassembly,
	// we prefer to show the actual opcode names.  Thus, only replace the
	// opcodes in question when the oneline flag is set.
	opcodeName := opcode.OpcodeName(pop.Opcode.Value)
	if oneline {
		if replName, ok := opcodeOnelineRepls[opcodeName]; ok {
			opcodeName = replName
		}

		// Nothing more to do for non-data push opcodes.
		if pop.Opcode.Length == 1 {
			return opcodeName
		}

		return fmt.Sprintf("%x", pop.Data)
	}

	// Nothing more to do for non-data push opcodes.
	if pop.Opcode.Length == 1 {
		return opcodeName
	}

	// Add length for the OP_PUSHDATA# opcodes.
	retString := opcodeName
	switch pop.Opcode.Length {
	case -1:
		retString += fmt.Sprintf(" 0x%02x", len(pop.Data))
	case -2:
		retString += fmt.Sprintf(" 0x%04x", len(pop.Data))
	case -4:
		retString += fmt.Sprintf(" 0x%08x", len(pop.Data))
	}

	return fmt.Sprintf("%s 0x%02x", retString, pop.Data)
}

// popBytes returns any data associated with the opcode encoded as it would be in
// a script.  This is used for unparsing scripts from parsed opcodes.
func popBytes(pop *parsescript.ParsedOpcode) ([]byte, er.R) {
	var retbytes []byte
	if pop.Opcode.Length > 0 {
		retbytes = make([]byte, 1, pop.Opcode.Length)
	} else {
		retbytes = make([]byte, 1, 1+len(pop.Data)-
			pop.Opcode.Length)
	}

	retbytes[0] = pop.Opcode.Value
	if pop.Opcode.Length == 1 {
		if len(pop.Data) != 0 {
			str := fmt.Sprintf("internal consistency error - "+
				"parsed opcode %s has data length %d when %d "+
				"was expected", opcode.OpcodeName(pop.Opcode.Value), len(pop.Data),
				0)
			return nil, txscripterr.ScriptError(txscripterr.ErrInternal, str)
		}
		return retbytes, nil
	}
	nbytes := pop.Opcode.Length
	if pop.Opcode.Length < 0 {
		l := len(pop.Data)
		// tempting just to hardcode to avoid the complexity here.
		switch pop.Opcode.Length {
		case -1:
			retbytes = append(retbytes, byte(l))
			nbytes = int(retbytes[1]) + len(retbytes)
		case -2:
			retbytes = append(retbytes, byte(l&0xff),
				byte(l>>8&0xff))
			nbytes = int(binary.LittleEndian.Uint16(retbytes[1:])) +
				len(retbytes)
		case -4:
			retbytes = append(retbytes, byte(l&0xff),
				byte((l>>8)&0xff), byte((l>>16)&0xff),
				byte((l>>24)&0xff))
			nbytes = int(binary.LittleEndian.Uint32(retbytes[1:])) +
				len(retbytes)
		}
	}

	retbytes = append(retbytes, pop.Data...)

	if len(retbytes) != nbytes {
		str := fmt.Sprintf("internal consistency error - "+
			"parsed opcode %s has data length %d when %d was "+
			"expected", opcode.OpcodeName(pop.Opcode.Value), len(retbytes), nbytes)
		return nil, txscripterr.ScriptError(txscripterr.ErrInternal, str)
	}

	return retbytes, nil
}

// *******************************************
// Opcode implementation functions start here.
// *******************************************

// opcodeDisabled is a common handler for disabled opcodes.  It returns an
// appropriate error indicating the opcode is disabled.  While it would
// ordinarily make more sense to detect if the script contains any disabled
// opcodes before executing in an initial parse step, the consensus rules
// dictate the script doesn't fail until the program counter passes over a
// disabled opcode (even when they appear in a branch that is not executed).
func opcodeDisabled(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	str := fmt.Sprintf("attempt to execute disabled opcode %s",
		opcode.OpcodeName(op.Opcode.Value))
	return txscripterr.ScriptError(txscripterr.ErrDisabledOpcode, str)
}

// opcodeReserved is a common handler for all reserved opcodes.  It returns an
// appropriate error indicating the opcode is reserved.
func opcodeReserved(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	str := fmt.Sprintf("attempt to execute reserved opcode %s",
		opcode.OpcodeName(op.Opcode.Value))
	return txscripterr.ScriptError(txscripterr.ErrReservedOpcode, str)
}

// opcodeInvalid is a common handler for all invalid opcodes.  It returns an
// appropriate error indicating the opcode is invalid.
func opcodeInvalid(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	str := fmt.Sprintf("attempt to execute invalid opcode %s",
		opcode.OpcodeName(op.Opcode.Value))
	return txscripterr.ScriptError(txscripterr.ErrReservedOpcode, str)
}

// opcodeFalse pushes an empty array to the data stack to represent false.  Note
// that 0, when encoded as a number according to the numeric encoding consensus
// rules, is an empty array.
func opcodeFalse(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	vm.dstack.PushByteArray(nil)
	return nil
}

// opcodePushData is a common handler for the vast majority of opcodes that push
// raw data (bytes) to the data stack.
func opcodePushData(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	vm.dstack.PushByteArray(op.Data)
	return nil
}

// opcode1Negate pushes -1, encoded as a number, to the data stack.
func opcode1Negate(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	vm.dstack.PushInt(scriptnum.ScriptNum(-1))
	return nil
}

// opcodeN is a common handler for the small integer data push opcodes.  It
// pushes the numeric value the opcode represents (which will be from 1 to 16)
// onto the data stack.
func opcodeN(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	// The opcodes are all defined consecutively, so the numeric value is
	// the difference.
	vm.dstack.PushInt(scriptnum.ScriptNum((op.Opcode.Value - (opcode.OP_1 - 1))))
	return nil
}

// opcodeNop is a common handler for the NOP family of opcodes.  As the name
// implies it generally does nothing, however, it will return an error when
// the flag to discourage use of NOPs is set for select opcodes.
func opcodeNop(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	switch op.Opcode.Value {
	case opcode.OP_NOP1, opcode.OP_NOP4, opcode.OP_NOP5,
		opcode.OP_NOP6, opcode.OP_NOP7, opcode.OP_NOP8, opcode.OP_NOP9, opcode.OP_NOP10:
		if vm.hasFlag(ScriptDiscourageUpgradableNops) {
			str := fmt.Sprintf("OP_NOP%d reserved for soft-fork "+
				"upgrades", op.Opcode.Value-(opcode.OP_NOP1-1))
			return txscripterr.ScriptError(txscripterr.ErrDiscourageUpgradableNOPs, str)
		}
	}
	return nil
}

// popIfBool enforces the "minimal if" policy during script execution if the
// particular flag is set.  If so, in order to eliminate an additional source
// of nuisance malleability, post-segwit for version 0 witness programs, we now
// require the following: for OP_IF and OP_NOT_IF, the top stack item MUST
// either be an empty byte slice, or [0x01]. Otherwise, the item at the top of
// the stack will be popped and interpreted as a boolean.
func popIfBool(vm *Engine) (bool, er.R) {
	// When not in witness execution mode, not executing a v0 witness
	// program, or the minimal if flag isn't set pop the top stack item as
	// a normal bool.
	if !vm.isWitnessVersionActive(0) || !vm.hasFlag(ScriptVerifyMinimalIf) {
		return vm.dstack.PopBool()
	}

	// At this point, a v0 witness program is being executed and the minimal
	// if flag is set, so enforce additional constraints on the top stack
	// item.
	so, err := vm.dstack.PopByteArray()
	if err != nil {
		return false, err
	}

	// The top element MUST have a length of at least one.
	if len(so) > 1 {
		str := fmt.Sprintf("minimal if is active, top element MUST "+
			"have a length of at least, instead length is %v",
			len(so))
		return false, txscripterr.ScriptError(txscripterr.ErrMinimalIf, str)
	}

	// Additionally, if the length is one, then the value MUST be 0x01.
	if len(so) == 1 && so[0] != 0x01 {
		str := fmt.Sprintf("minimal if is active, top stack item MUST "+
			"be an empty byte array or 0x01, is instead: %v",
			so[0])
		return false, txscripterr.ScriptError(txscripterr.ErrMinimalIf, str)
	}

	return asBool(so), nil
}

// opcodeIf treats the top item on the data stack as a boolean and removes it.
//
// An appropriate entry is added to the conditional stack depending on whether
// the boolean is true and whether this if is on an executing branch in order
// to allow proper execution of further opcodes depending on the conditional
// logic.  When the boolean is true, the first branch will be executed (unless
// this opcode is nested in a non-executed branch).
//
// <expression> if [statements] [else [statements]] endif
//
// Note that, unlike for all non-conditional opcodes, this is executed even when
// it is on a non-executing branch so proper nesting is maintained.
//
// Data stack transformation: [... bool] -> [...]
// Conditional stack transformation: [...] -> [... OpCondValue]
func opcodeIf(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	condVal := OpCondFalse
	if vm.isBranchExecuting() {
		ok, err := popIfBool(vm)
		if err != nil {
			return err
		}

		if ok {
			condVal = OpCondTrue
		}
	} else {
		condVal = OpCondSkip
	}
	vm.condStack = append(vm.condStack, condVal)
	return nil
}

// opcodeNotIf treats the top item on the data stack as a boolean and removes
// it.
//
// An appropriate entry is added to the conditional stack depending on whether
// the boolean is true and whether this if is on an executing branch in order
// to allow proper execution of further opcodes depending on the conditional
// logic.  When the boolean is false, the first branch will be executed (unless
// this opcode is nested in a non-executed branch).
//
// <expression> notif [statements] [else [statements]] endif
//
// Note that, unlike for all non-conditional opcodes, this is executed even when
// it is on a non-executing branch so proper nesting is maintained.
//
// Data stack transformation: [... bool] -> [...]
// Conditional stack transformation: [...] -> [... OpCondValue]
func opcodeNotIf(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	condVal := OpCondFalse
	if vm.isBranchExecuting() {
		ok, err := popIfBool(vm)
		if err != nil {
			return err
		}

		if !ok {
			condVal = OpCondTrue
		}
	} else {
		condVal = OpCondSkip
	}
	vm.condStack = append(vm.condStack, condVal)
	return nil
}

// opcodeElse inverts conditional execution for other half of if/else/endif.
//
// An error is returned if there has not already been a matching OP_IF.
//
// Conditional stack transformation: [... OpCondValue] -> [... !OpCondValue]
func opcodeElse(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	if len(vm.condStack) == 0 {
		str := fmt.Sprintf("encountered opcode %s with no matching "+
			"opcode to begin conditional execution", opcode.OpcodeName(op.Opcode.Value))
		return txscripterr.ScriptError(txscripterr.ErrUnbalancedConditional, str)
	}

	conditionalIdx := len(vm.condStack) - 1
	switch vm.condStack[conditionalIdx] {
	case OpCondTrue:
		vm.condStack[conditionalIdx] = OpCondFalse
	case OpCondFalse:
		vm.condStack[conditionalIdx] = OpCondTrue
	case OpCondSkip:
		// Value doesn't change in skip since it indicates this opcode
		// is nested in a non-executed branch.
	}
	return nil
}

// opcodeEndif terminates a conditional block, removing the value from the
// conditional execution stack.
//
// An error is returned if there has not already been a matching OP_IF.
//
// Conditional stack transformation: [... OpCondValue] -> [...]
func opcodeEndif(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	if len(vm.condStack) == 0 {
		str := fmt.Sprintf("encountered opcode %s with no matching "+
			"opcode to begin conditional execution", opcode.OpcodeName(op.Opcode.Value))
		return txscripterr.ScriptError(txscripterr.ErrUnbalancedConditional, str)
	}

	vm.condStack = vm.condStack[:len(vm.condStack)-1]
	return nil
}

// abstractVerify examines the top item on the data stack as a boolean value and
// verifies it evaluates to true.  An error is returned either when there is no
// item on the stack or when that item evaluates to false.  In the latter case
// where the verification fails specifically due to the top item evaluating
// to false, the returned error will use the passed error code.
func abstractVerify(op *parsescript.ParsedOpcode, vm *Engine, c *er.ErrorCode) er.R {
	verified, err := vm.dstack.PopBool()
	if err != nil {
		return err
	}

	if !verified {
		str := fmt.Sprintf("%s failed", opcode.OpcodeName(op.Opcode.Value))
		return txscripterr.ScriptError(c, str)
	}
	return nil
}

// opcodeVerify examines the top item on the data stack as a boolean value and
// verifies it evaluates to true.  An error is returned if it does not.
func opcodeVerify(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return abstractVerify(op, vm, txscripterr.ErrVerify)
}

// opcodeReturn returns an appropriate error since it is always an error to
// return early from a script.
func opcodeReturn(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return txscripterr.ScriptError(txscripterr.ErrEarlyReturn, "script returned early")
}

// verifyLockTime is a helper function used to validate locktimes.
func verifyLockTime(txLockTime, threshold, lockTime int64) er.R {
	// The lockTimes in both the script and transaction must be of the same
	// type.
	if !((txLockTime < threshold && lockTime < threshold) ||
		(txLockTime >= threshold && lockTime >= threshold)) {
		str := fmt.Sprintf("mismatched locktime types -- tx locktime "+
			"%d, stack locktime %d", txLockTime, lockTime)
		return txscripterr.ScriptError(txscripterr.ErrUnsatisfiedLockTime, str)
	}

	if lockTime > txLockTime {
		str := fmt.Sprintf("locktime requirement not satisfied -- "+
			"locktime is greater than the transaction locktime: "+
			"%d > %d", lockTime, txLockTime)
		return txscripterr.ScriptError(txscripterr.ErrUnsatisfiedLockTime, str)
	}

	return nil
}

// opcodeCheckLockTimeVerify compares the top item on the data stack to the
// LockTime field of the transaction containing the script signature
// validating if the transaction outputs are spendable yet.  If flag
// ScriptVerifyCheckLockTimeVerify is not set, the code continues as if OP_NOP2
// were executed.
func opcodeCheckLockTimeVerify(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	// If the ScriptVerifyCheckLockTimeVerify script flag is not set, treat
	// opcode as OP_NOP2 instead.
	if !vm.hasFlag(ScriptVerifyCheckLockTimeVerify) {
		if vm.hasFlag(ScriptDiscourageUpgradableNops) {
			return txscripterr.ScriptError(txscripterr.ErrDiscourageUpgradableNOPs,
				"OP_NOP2 reserved for soft-fork upgrades")
		}
		return nil
	}

	// The current transaction locktime is a uint32 resulting in a maximum
	// locktime of 2^32-1 (the year 2106).  However, scriptNums are signed
	// and therefore a standard 4-byte scriptNum would only support up to a
	// maximum of 2^31-1 (the year 2038).  Thus, a 5-byte scriptNum is used
	// here since it will support up to 2^39-1 which allows dates beyond the
	// current locktime limit.
	//
	// PeekByteArray is used here instead of PeekInt because we do not want
	// to be limited to a 4-byte integer for reasons specified above.
	so, err := vm.dstack.PeekByteArray(0)
	if err != nil {
		return err
	}
	lockTime, err := scriptnum.MakeScriptNum(so, vm.dstack.verifyMinimalData, 5)
	if err != nil {
		return err
	}

	// In the rare event that the argument needs to be < 0 due to some
	// arithmetic being done first, you can always use
	// 0 OP_MAX OP_CHECKLOCKTIMEVERIFY.
	if lockTime < 0 {
		str := fmt.Sprintf("negative lock time: %d", lockTime)
		return txscripterr.ScriptError(txscripterr.ErrNegativeLockTime, str)
	}

	// The lock time field of a transaction is either a block height at
	// which the transaction is finalized or a timestamp depending on if the
	// value is before the txscript.LockTimeThreshold.  When it is under the
	// threshold it is a block height.
	err = verifyLockTime(int64(vm.tx.LockTime), params.LockTimeThreshold,
		int64(lockTime))
	if err != nil {
		return err
	}

	// The lock time feature can also be disabled, thereby bypassing
	// OP_CHECKLOCKTIMEVERIFY, if every transaction input has been finalized by
	// setting its sequence to the maximum value (constants.MaxTxInSequenceNum).  This
	// condition would result in the transaction being allowed into the blockchain
	// making the opcode ineffective.
	//
	// This condition is prevented by enforcing that the input being used by
	// the opcode is unlocked (its sequence number is less than the max
	// value).  This is sufficient to prove correctness without having to
	// check every input.
	//
	// NOTE: This implies that even if the transaction is not finalized due to
	// another input being unlocked, the opcode execution will still fail when the
	// input being used by the opcode is locked.
	if vm.tx.TxIn[vm.txIdx].Sequence == constants.MaxTxInSequenceNum {
		return txscripterr.ScriptError(txscripterr.ErrUnsatisfiedLockTime,
			"transaction input is finalized")
	}

	return nil
}

// opcodeCheckSequenceVerify compares the top item on the data stack to the
// LockTime field of the transaction containing the script signature
// validating if the transaction outputs are spendable yet.  If flag
// ScriptVerifyCheckSequenceVerify is not set, the code continues as if OP_NOP3
// were executed.
func opcodeCheckSequenceVerify(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	// If the ScriptVerifyCheckSequenceVerify script flag is not set, treat
	// opcode as OP_NOP3 instead.
	if !vm.hasFlag(ScriptVerifyCheckSequenceVerify) {
		if vm.hasFlag(ScriptDiscourageUpgradableNops) {
			return txscripterr.ScriptError(txscripterr.ErrDiscourageUpgradableNOPs,
				"OP_NOP3 reserved for soft-fork upgrades")
		}
		return nil
	}

	// The current transaction sequence is a uint32 resulting in a maximum
	// sequence of 2^32-1.  However, scriptNums are signed and therefore a
	// standard 4-byte scriptNum would only support up to a maximum of
	// 2^31-1.  Thus, a 5-byte scriptNum is used here since it will support
	// up to 2^39-1 which allows sequences beyond the current sequence
	// limit.
	//
	// PeekByteArray is used here instead of PeekInt because we do not want
	// to be limited to a 4-byte integer for reasons specified above.
	so, err := vm.dstack.PeekByteArray(0)
	if err != nil {
		return err
	}
	stackSequence, err := scriptnum.MakeScriptNum(so, vm.dstack.verifyMinimalData, 5)
	if err != nil {
		return err
	}

	// In the rare event that the argument needs to be < 0 due to some
	// arithmetic being done first, you can always use
	// 0 OP_MAX OP_CHECKSEQUENCEVERIFY.
	if stackSequence < 0 {
		str := fmt.Sprintf("negative sequence: %d", stackSequence)
		return txscripterr.ScriptError(txscripterr.ErrNegativeLockTime, str)
	}

	sequence := int64(stackSequence)

	// To provide for future soft-fork extensibility, if the
	// operand has the disabled lock-time flag set,
	// CHECKSEQUENCEVERIFY behaves as a NOP.
	if sequence&int64(constants.SequenceLockTimeDisabled) != 0 {
		return nil
	}

	// Transaction version numbers not high enough to trigger CSV rules must
	// fail.
	if vm.tx.Version < 2 {
		str := fmt.Sprintf("invalid transaction version: %d",
			vm.tx.Version)
		return txscripterr.ScriptError(txscripterr.ErrUnsatisfiedLockTime, str)
	}

	// Sequence numbers with their most significant bit set are not
	// consensus constrained. Testing that the transaction's sequence
	// number does not have this bit set prevents using this property
	// to get around a CHECKSEQUENCEVERIFY check.
	txSequence := int64(vm.tx.TxIn[vm.txIdx].Sequence)
	if txSequence&int64(constants.SequenceLockTimeDisabled) != 0 {
		str := fmt.Sprintf("transaction sequence has sequence "+
			"locktime disabled bit set: 0x%x", txSequence)
		return txscripterr.ScriptError(txscripterr.ErrUnsatisfiedLockTime, str)
	}

	// Mask off non-consensus bits before doing comparisons.
	lockTimeMask := int64(constants.SequenceLockTimeIsSeconds |
		constants.SequenceLockTimeMask)
	return verifyLockTime(txSequence&lockTimeMask,
		constants.SequenceLockTimeIsSeconds, sequence&lockTimeMask)
}

// opcodeToAltStack removes the top item from the main data stack and pushes it
// onto the alternate data stack.
//
// Main data stack transformation: [... x1 x2 x3] -> [... x1 x2]
// Alt data stack transformation:  [... y1 y2 y3] -> [... y1 y2 y3 x3]
func opcodeToAltStack(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	so, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}
	vm.astack.PushByteArray(so)

	return nil
}

// opcodeFromAltStack removes the top item from the alternate data stack and
// pushes it onto the main data stack.
//
// Main data stack transformation: [... x1 x2 x3] -> [... x1 x2 x3 y3]
// Alt data stack transformation:  [... y1 y2 y3] -> [... y1 y2]
func opcodeFromAltStack(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	so, err := vm.astack.PopByteArray()
	if err != nil {
		return err
	}
	vm.dstack.PushByteArray(so)

	return nil
}

// opcode2Drop removes the top 2 items from the data stack.
//
// Stack transformation: [... x1 x2 x3] -> [... x1]
func opcode2Drop(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return vm.dstack.DropN(2)
}

// opcode2Dup duplicates the top 2 items on the data stack.
//
// Stack transformation: [... x1 x2 x3] -> [... x1 x2 x3 x2 x3]
func opcode2Dup(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return vm.dstack.DupN(2)
}

// opcode3Dup duplicates the top 3 items on the data stack.
//
// Stack transformation: [... x1 x2 x3] -> [... x1 x2 x3 x1 x2 x3]
func opcode3Dup(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return vm.dstack.DupN(3)
}

// opcode2Over duplicates the 2 items before the top 2 items on the data stack.
//
// Stack transformation: [... x1 x2 x3 x4] -> [... x1 x2 x3 x4 x1 x2]
func opcode2Over(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return vm.dstack.OverN(2)
}

// opcode2Rot rotates the top 6 items on the data stack to the left twice.
//
// Stack transformation: [... x1 x2 x3 x4 x5 x6] -> [... x3 x4 x5 x6 x1 x2]
func opcode2Rot(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return vm.dstack.RotN(2)
}

// opcode2Swap swaps the top 2 items on the data stack with the 2 that come
// before them.
//
// Stack transformation: [... x1 x2 x3 x4] -> [... x3 x4 x1 x2]
func opcode2Swap(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return vm.dstack.SwapN(2)
}

// opcodeIfDup duplicates the top item of the stack if it is not zero.
//
// Stack transformation (x1==0): [... x1] -> [... x1]
// Stack transformation (x1!=0): [... x1] -> [... x1 x1]
func opcodeIfDup(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	so, err := vm.dstack.PeekByteArray(0)
	if err != nil {
		return err
	}

	// Push copy of data iff it isn't zero
	if asBool(so) {
		vm.dstack.PushByteArray(so)
	}

	return nil
}

// opcodeDepth pushes the depth of the data stack prior to executing this
// opcode, encoded as a number, onto the data stack.
//
// Stack transformation: [...] -> [... <num of items on the stack>]
// Example with 2 items: [x1 x2] -> [x1 x2 2]
// Example with 3 items: [x1 x2 x3] -> [x1 x2 x3 3]
func opcodeDepth(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	vm.dstack.PushInt(scriptnum.ScriptNum(vm.dstack.Depth()))
	return nil
}

// opcodeDrop removes the top item from the data stack.
//
// Stack transformation: [... x1 x2 x3] -> [... x1 x2]
func opcodeDrop(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return vm.dstack.DropN(1)
}

// opcodeDup duplicates the top item on the data stack.
//
// Stack transformation: [... x1 x2 x3] -> [... x1 x2 x3 x3]
func opcodeDup(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return vm.dstack.DupN(1)
}

// opcodeNip removes the item before the top item on the data stack.
//
// Stack transformation: [... x1 x2 x3] -> [... x1 x3]
func opcodeNip(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return vm.dstack.NipN(1)
}

// opcodeOver duplicates the item before the top item on the data stack.
//
// Stack transformation: [... x1 x2 x3] -> [... x1 x2 x3 x2]
func opcodeOver(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return vm.dstack.OverN(1)
}

// opcodePick treats the top item on the data stack as an integer and duplicates
// the item on the stack that number of items back to the top.
//
// Stack transformation: [xn ... x2 x1 x0 n] -> [xn ... x2 x1 x0 xn]
// Example with n=1: [x2 x1 x0 1] -> [x2 x1 x0 x1]
// Example with n=2: [x2 x1 x0 2] -> [x2 x1 x0 x2]
func opcodePick(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	val, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	return vm.dstack.PickN(val.Int32())
}

// opcodeRoll treats the top item on the data stack as an integer and moves
// the item on the stack that number of items back to the top.
//
// Stack transformation: [xn ... x2 x1 x0 n] -> [... x2 x1 x0 xn]
// Example with n=1: [x2 x1 x0 1] -> [x2 x0 x1]
// Example with n=2: [x2 x1 x0 2] -> [x1 x0 x2]
func opcodeRoll(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	val, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	return vm.dstack.RollN(val.Int32())
}

// opcodeRot rotates the top 3 items on the data stack to the left.
//
// Stack transformation: [... x1 x2 x3] -> [... x2 x3 x1]
func opcodeRot(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return vm.dstack.RotN(1)
}

// opcodeSwap swaps the top two items on the stack.
//
// Stack transformation: [... x1 x2] -> [... x2 x1]
func opcodeSwap(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return vm.dstack.SwapN(1)
}

// opcodeTuck inserts a duplicate of the top item of the data stack before the
// second-to-top item.
//
// Stack transformation: [... x1 x2] -> [... x2 x1 x2]
func opcodeTuck(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	return vm.dstack.Tuck()
}

// opcodeSize pushes the size of the top item of the data stack onto the data
// stack.
//
// Stack transformation: [... x1] -> [... x1 len(x1)]
func opcodeSize(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	so, err := vm.dstack.PeekByteArray(0)
	if err != nil {
		return err
	}

	vm.dstack.PushInt(scriptnum.ScriptNum(len(so)))
	return nil
}

// opcodeEqual removes the top 2 items of the data stack, compares them as raw
// bytes, and pushes the result, encoded as a boolean, back to the stack.
//
// Stack transformation: [... x1 x2] -> [... bool]
func opcodeEqual(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	a, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}
	b, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	vm.dstack.PushBool(bytes.Equal(a, b))
	return nil
}

// opcodeEqualVerify is a combination of opcodeEqual and opcodeVerify.
// Specifically, it removes the top 2 items of the data stack, compares them,
// and pushes the result, encoded as a boolean, back to the stack.  Then, it
// examines the top item on the data stack as a boolean value and verifies it
// evaluates to true.  An error is returned if it does not.
//
// Stack transformation: [... x1 x2] -> [... bool] -> [...]
func opcodeEqualVerify(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	err := opcodeEqual(op, vm)
	if err == nil {
		err = abstractVerify(op, vm, txscripterr.ErrEqualVerify)
	}
	return err
}

// opcode1Add treats the top item on the data stack as an integer and replaces
// it with its incremented value (plus 1).
//
// Stack transformation: [... x1 x2] -> [... x1 x2+1]
func opcode1Add(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	m, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	vm.dstack.PushInt(m + 1)
	return nil
}

// opcode1Sub treats the top item on the data stack as an integer and replaces
// it with its decremented value (minus 1).
//
// Stack transformation: [... x1 x2] -> [... x1 x2-1]
func opcode1Sub(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	m, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}
	vm.dstack.PushInt(m - 1)

	return nil
}

// opcodeNegate treats the top item on the data stack as an integer and replaces
// it with its negation.
//
// Stack transformation: [... x1 x2] -> [... x1 -x2]
func opcodeNegate(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	m, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	vm.dstack.PushInt(-m)
	return nil
}

// opcodeAbs treats the top item on the data stack as an integer and replaces it
// it with its absolute value.
//
// Stack transformation: [... x1 x2] -> [... x1 abs(x2)]
func opcodeAbs(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	m, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if m < 0 {
		m = -m
	}
	vm.dstack.PushInt(m)
	return nil
}

// opcodeNot treats the top item on the data stack as an integer and replaces
// it with its "inverted" value (0 becomes 1, non-zero becomes 0).
//
// NOTE: While it would probably make more sense to treat the top item as a
// boolean, and push the opposite, which is really what the intention of this
// opcode is, it is extremely important that is not done because integers are
// interpreted differently than booleans and the consensus rules for this opcode
// dictate the item is interpreted as an integer.
//
// Stack transformation (x2==0): [... x1 0] -> [... x1 1]
// Stack transformation (x2!=0): [... x1 1] -> [... x1 0]
// Stack transformation (x2!=0): [... x1 17] -> [... x1 0]
func opcodeNot(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	m, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if m == 0 {
		vm.dstack.PushInt(scriptnum.ScriptNum(1))
	} else {
		vm.dstack.PushInt(scriptnum.ScriptNum(0))
	}
	return nil
}

// opcode0NotEqual treats the top item on the data stack as an integer and
// replaces it with either a 0 if it is zero, or a 1 if it is not zero.
//
// Stack transformation (x2==0): [... x1 0] -> [... x1 0]
// Stack transformation (x2!=0): [... x1 1] -> [... x1 1]
// Stack transformation (x2!=0): [... x1 17] -> [... x1 1]
func opcode0NotEqual(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	m, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if m != 0 {
		m = 1
	}
	vm.dstack.PushInt(m)
	return nil
}

// opcodeAdd treats the top two items on the data stack as integers and replaces
// them with their sum.
//
// Stack transformation: [... x1 x2] -> [... x1+x2]
func opcodeAdd(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	vm.dstack.PushInt(v0 + v1)
	return nil
}

// opcodeSub treats the top two items on the data stack as integers and replaces
// them with the result of subtracting the top entry from the second-to-top
// entry.
//
// Stack transformation: [... x1 x2] -> [... x1-x2]
func opcodeSub(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	vm.dstack.PushInt(v1 - v0)
	return nil
}

// opcodeBoolAnd treats the top two items on the data stack as integers.  When
// both of them are not zero, they are replaced with a 1, otherwise a 0.
//
// Stack transformation (x1==0, x2==0): [... 0 0] -> [... 0]
// Stack transformation (x1!=0, x2==0): [... 5 0] -> [... 0]
// Stack transformation (x1==0, x2!=0): [... 0 7] -> [... 0]
// Stack transformation (x1!=0, x2!=0): [... 4 8] -> [... 1]
func opcodeBoolAnd(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v0 != 0 && v1 != 0 {
		vm.dstack.PushInt(scriptnum.ScriptNum(1))
	} else {
		vm.dstack.PushInt(scriptnum.ScriptNum(0))
	}

	return nil
}

// opcodeBoolOr treats the top two items on the data stack as integers.  When
// either of them are not zero, they are replaced with a 1, otherwise a 0.
//
// Stack transformation (x1==0, x2==0): [... 0 0] -> [... 0]
// Stack transformation (x1!=0, x2==0): [... 5 0] -> [... 1]
// Stack transformation (x1==0, x2!=0): [... 0 7] -> [... 1]
// Stack transformation (x1!=0, x2!=0): [... 4 8] -> [... 1]
func opcodeBoolOr(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v0 != 0 || v1 != 0 {
		vm.dstack.PushInt(scriptnum.ScriptNum(1))
	} else {
		vm.dstack.PushInt(scriptnum.ScriptNum(0))
	}

	return nil
}

// opcodeNumEqual treats the top two items on the data stack as integers.  When
// they are equal, they are replaced with a 1, otherwise a 0.
//
// Stack transformation (x1==x2): [... 5 5] -> [... 1]
// Stack transformation (x1!=x2): [... 5 7] -> [... 0]
func opcodeNumEqual(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v0 == v1 {
		vm.dstack.PushInt(scriptnum.ScriptNum(1))
	} else {
		vm.dstack.PushInt(scriptnum.ScriptNum(0))
	}

	return nil
}

// opcodeNumEqualVerify is a combination of opcodeNumEqual and opcodeVerify.
//
// Specifically, treats the top two items on the data stack as integers.  When
// they are equal, they are replaced with a 1, otherwise a 0.  Then, it examines
// the top item on the data stack as a boolean value and verifies it evaluates
// to true.  An error is returned if it does not.
//
// Stack transformation: [... x1 x2] -> [... bool] -> [...]
func opcodeNumEqualVerify(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	err := opcodeNumEqual(op, vm)
	if err == nil {
		err = abstractVerify(op, vm, txscripterr.ErrNumEqualVerify)
	}
	return err
}

// opcodeNumNotEqual treats the top two items on the data stack as integers.
// When they are NOT equal, they are replaced with a 1, otherwise a 0.
//
// Stack transformation (x1==x2): [... 5 5] -> [... 0]
// Stack transformation (x1!=x2): [... 5 7] -> [... 1]
func opcodeNumNotEqual(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v0 != v1 {
		vm.dstack.PushInt(scriptnum.ScriptNum(1))
	} else {
		vm.dstack.PushInt(scriptnum.ScriptNum(0))
	}

	return nil
}

// opcodeLessThan treats the top two items on the data stack as integers.  When
// the second-to-top item is less than the top item, they are replaced with a 1,
// otherwise a 0.
//
// Stack transformation: [... x1 x2] -> [... bool]
func opcodeLessThan(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v1 < v0 {
		vm.dstack.PushInt(scriptnum.ScriptNum(1))
	} else {
		vm.dstack.PushInt(scriptnum.ScriptNum(0))
	}

	return nil
}

// opcodeGreaterThan treats the top two items on the data stack as integers.
// When the second-to-top item is greater than the top item, they are replaced
// with a 1, otherwise a 0.
//
// Stack transformation: [... x1 x2] -> [... bool]
func opcodeGreaterThan(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v1 > v0 {
		vm.dstack.PushInt(scriptnum.ScriptNum(1))
	} else {
		vm.dstack.PushInt(scriptnum.ScriptNum(0))
	}
	return nil
}

// opcodeLessThanOrEqual treats the top two items on the data stack as integers.
// When the second-to-top item is less than or equal to the top item, they are
// replaced with a 1, otherwise a 0.
//
// Stack transformation: [... x1 x2] -> [... bool]
func opcodeLessThanOrEqual(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v1 <= v0 {
		vm.dstack.PushInt(scriptnum.ScriptNum(1))
	} else {
		vm.dstack.PushInt(scriptnum.ScriptNum(0))
	}
	return nil
}

// opcodeGreaterThanOrEqual treats the top two items on the data stack as
// integers.  When the second-to-top item is greater than or equal to the top
// item, they are replaced with a 1, otherwise a 0.
//
// Stack transformation: [... x1 x2] -> [... bool]
func opcodeGreaterThanOrEqual(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v1 >= v0 {
		vm.dstack.PushInt(scriptnum.ScriptNum(1))
	} else {
		vm.dstack.PushInt(scriptnum.ScriptNum(0))
	}

	return nil
}

// opcodeMin treats the top two items on the data stack as integers and replaces
// them with the minimum of the two.
//
// Stack transformation: [... x1 x2] -> [... min(x1, x2)]
func opcodeMin(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v1 < v0 {
		vm.dstack.PushInt(v1)
	} else {
		vm.dstack.PushInt(v0)
	}

	return nil
}

// opcodeMax treats the top two items on the data stack as integers and replaces
// them with the maximum of the two.
//
// Stack transformation: [... x1 x2] -> [... max(x1, x2)]
func opcodeMax(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	v0, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if v1 > v0 {
		vm.dstack.PushInt(v1)
	} else {
		vm.dstack.PushInt(v0)
	}

	return nil
}

// opcodeWithin treats the top 3 items on the data stack as integers.  When the
// value to test is within the specified range (left inclusive), they are
// replaced with a 1, otherwise a 0.
//
// The top item is the max value, the second-top-item is the minimum value, and
// the third-to-top item is the value to test.
//
// Stack transformation: [... x1 min max] -> [... bool]
func opcodeWithin(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	maxVal, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	minVal, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	x, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	if x >= minVal && x < maxVal {
		vm.dstack.PushInt(scriptnum.ScriptNum(1))
	} else {
		vm.dstack.PushInt(scriptnum.ScriptNum(0))
	}
	return nil
}

// opcodeRipemd160 treats the top item of the data stack as raw bytes and
// replaces it with ripemd160(data).
//
// Stack transformation: [... x1] -> [... ripemd160(x1)]
func opcodeRipemd160(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	buf, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	vm.dstack.PushByteArray(btcutil.Ripemd160(buf))
	return nil
}

// opcodeSha1 treats the top item of the data stack as raw bytes and replaces it
// with sha1(data).
//
// Stack transformation: [... x1] -> [... sha1(x1)]
func opcodeSha1(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	buf, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	hash := sha1.Sum(buf)
	vm.dstack.PushByteArray(hash[:])
	return nil
}

// opcodeSha256 treats the top item of the data stack as raw bytes and replaces
// it with sha256(data).
//
// Stack transformation: [... x1] -> [... sha256(x1)]
func opcodeSha256(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	buf, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	hash := sha256.Sum256(buf)
	vm.dstack.PushByteArray(hash[:])
	return nil
}

// opcodeHash160 treats the top item of the data stack as raw bytes and replaces
// it with ripemd160(sha256(data)).
//
// Stack transformation: [... x1] -> [... ripemd160(sha256(x1))]
func opcodeHash160(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	buf, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	vm.dstack.PushByteArray(btcutil.Hash160(buf))
	return nil
}

// opcodeHash256 treats the top item of the data stack as raw bytes and replaces
// it with sha256(sha256(data)).
//
// Stack transformation: [... x1] -> [... sha256(sha256(x1))]
func opcodeHash256(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	buf, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	vm.dstack.PushByteArray(chainhash.DoubleHashB(buf))
	return nil
}

// opcodeCodeSeparator stores the current script offset as the most recently
// seen OP_CODESEPARATOR which is used during signature checking.
//
// This opcode does not change the contents of the data stack.
func opcodeCodeSeparator(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	vm.lastCodeSep = vm.scriptOff
	return nil
}

// opcodeCheckSig treats the top 2 items on the stack as a public key and a
// signature and replaces them with a bool which indicates if the signature was
// successfully verified.
//
// The process of verifying a signature requires calculating a signature hash in
// the same way the transaction signer did.  It involves hashing portions of the
// transaction based on the hash type byte (which is the final byte of the
// signature) and the portion of the script starting from the most recent
// OP_CODESEPARATOR (or the beginning of the script if there are none) to the
// end of the script (with any other OP_CODESEPARATORs removed).  Once this
// "script hash" is calculated, the signature is checked using standard
// cryptographic methods against the provided public key.
//
// Stack transformation: [... signature pubkey] -> [... bool]
func opcodeCheckSig(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	pkBytes, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	fullSigBytes, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	// The signature actually needs needs to be longer than this, but at
	// least 1 byte is needed for the hash type below.  The full length is
	// checked depending on the script flags and upon parsing the signature.
	if len(fullSigBytes) < 1 {
		vm.dstack.PushBool(false)
		return nil
	}

	// Trim off hashtype from the signature string and check if the
	// signature and pubkey conform to the strict encoding requirements
	// depending on the flags.
	//
	// NOTE: When the strict encoding flags are set, any errors in the
	// signature or public encoding here result in an immediate script error
	// (and thus no result bool is pushed to the data stack).  This differs
	// from the logic below where any errors in parsing the signature is
	// treated as the signature failure resulting in false being pushed to
	// the data stack.  This is required because the more general script
	// validation consensus rules do not have the new strict encoding
	// requirements enabled by the flags.
	hashType := params.SigHashType(fullSigBytes[len(fullSigBytes)-1])
	sigBytes := fullSigBytes[:len(fullSigBytes)-1]
	if err := vm.checkHashTypeEncoding(hashType); err != nil {
		return err
	}
	if err := vm.checkSignatureEncoding(sigBytes); err != nil {
		return err
	}
	if err := vm.checkPubKeyEncoding(pkBytes); err != nil {
		return err
	}

	// Get script starting from the most recent OP_CODESEPARATOR.
	subScript := vm.subScript()

	// Generate the signature hash based on the signature hash type.
	var hash []byte
	if vm.isWitnessVersionActive(0) {
		var sigHashes *TxSigHashes
		if vm.hashCache != nil {
			sigHashes = vm.hashCache
		} else {
			sigHashes = NewTxSigHashes(&vm.tx)
		}

		hash, err = calcWitnessSignatureHash(subScript, sigHashes, hashType,
			&vm.tx, vm.txIdx, vm.inputAmount)
		if err != nil {
			return err
		}
	} else {
		// Remove the signature since there is no way for a signature
		// to sign itself.
		subScript = removeOpcodeByData(subScript, fullSigBytes)

		hash = calcSignatureHash(subScript, hashType, &vm.tx, vm.txIdx)
	}

	pubKey, err := btcec.ParsePubKey(pkBytes, btcec.S256())
	if err != nil {
		vm.dstack.PushBool(false)
		return nil
	}

	var signature *btcec.Signature
	if vm.hasFlag(ScriptVerifyStrictEncoding) ||
		vm.hasFlag(ScriptVerifyDERSignatures) {

		signature, err = btcec.ParseDERSignature(sigBytes, btcec.S256())
	} else {
		signature, err = btcec.ParseSignature(sigBytes, btcec.S256())
	}
	if err != nil {
		vm.dstack.PushBool(false)
		return nil
	}

	var valid bool
	if vm.sigCache != nil {
		var sigHash chainhash.Hash
		copy(sigHash[:], hash)

		valid = vm.sigCache.Exists(sigHash, signature, pubKey)
		if !valid && signature.Verify(hash, pubKey) {
			vm.sigCache.Add(sigHash, signature, pubKey)
			valid = true
		}
	} else {
		valid = signature.Verify(hash, pubKey)
	}

	if !valid && vm.hasFlag(ScriptVerifyNullFail) && len(sigBytes) > 0 {
		str := "signature not empty on failed checksig"
		return txscripterr.ScriptError(txscripterr.ErrNullFail, str)
	}

	vm.dstack.PushBool(valid)
	return nil
}

// opcodeCheckSigVerify is a combination of opcodeCheckSig and opcodeVerify.
// The opcodeCheckSig function is invoked followed by opcodeVerify.  See the
// documentation for each of those opcodes for more details.
//
// Stack transformation: signature pubkey] -> [... bool] -> [...]
func opcodeCheckSigVerify(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	err := opcodeCheckSig(op, vm)
	if err == nil {
		err = abstractVerify(op, vm, txscripterr.ErrCheckSigVerify)
	}
	return err
}

// parsedSigInfo houses a raw signature along with its parsed form and a flag
// for whether or not it has already been parsed.  It is used to prevent parsing
// the same signature multiple times when verifying a multisig.
type parsedSigInfo struct {
	signature       []byte
	parsedSignature *btcec.Signature
	parsed          bool
}

// opcodeCheckMultiSig treats the top item on the stack as an integer number of
// public keys, followed by that many entries as raw data representing the public
// keys, followed by the integer number of signatures, followed by that many
// entries as raw data representing the signatures.
//
// Due to a bug in the original Satoshi client implementation, an additional
// dummy argument is also required by the consensus rules, although it is not
// used.  The dummy value SHOULD be an OP_0, although that is not required by
// the consensus rules.  When the ScriptStrictMultiSig flag is set, it must be
// OP_0.
//
// All of the aforementioned stack items are replaced with a bool which
// indicates if the requisite number of signatures were successfully verified.
//
// See the opcodeCheckSigVerify documentation for more details about the process
// for verifying each signature.
//
// Stack transformation:
// [... dummy [sig ...] numsigs [pubkey ...] numpubkeys] -> [... bool]
func opcodeCheckMultiSig(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	numKeys, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}

	numPubKeys := int(numKeys.Int32())
	if numPubKeys < 0 {
		str := fmt.Sprintf("number of pubkeys %d is negative",
			numPubKeys)
		return txscripterr.ScriptError(txscripterr.ErrInvalidPubKeyCount, str)
	}
	if numPubKeys > params.MaxPubKeysPerMultiSig {
		str := fmt.Sprintf("too many pubkeys: %d > %d",
			numPubKeys, params.MaxPubKeysPerMultiSig)
		return txscripterr.ScriptError(txscripterr.ErrInvalidPubKeyCount, str)
	}
	vm.numOps += numPubKeys
	if vm.numOps > params.MaxOpsPerScript {
		str := fmt.Sprintf("exceeded max operation limit of %d",
			params.MaxOpsPerScript)
		return txscripterr.ScriptError(txscripterr.ErrTooManyOperations, str)
	}

	pubKeys := make([][]byte, 0, numPubKeys)
	for i := 0; i < numPubKeys; i++ {
		pubKey, err := vm.dstack.PopByteArray()
		if err != nil {
			return err
		}
		pubKeys = append(pubKeys, pubKey)
	}

	numSigs, err := vm.dstack.PopInt()
	if err != nil {
		return err
	}
	numSignatures := int(numSigs.Int32())
	if numSignatures < 0 {
		str := fmt.Sprintf("number of signatures %d is negative",
			numSignatures)
		return txscripterr.ScriptError(txscripterr.ErrInvalidSignatureCount, str)

	}
	if numSignatures > numPubKeys {
		str := fmt.Sprintf("more signatures than pubkeys: %d > %d",
			numSignatures, numPubKeys)
		return txscripterr.ScriptError(txscripterr.ErrInvalidSignatureCount, str)
	}

	signatures := make([]*parsedSigInfo, 0, numSignatures)
	for i := 0; i < numSignatures; i++ {
		signature, err := vm.dstack.PopByteArray()
		if err != nil {
			return err
		}
		sigInfo := &parsedSigInfo{signature: signature}
		signatures = append(signatures, sigInfo)
	}

	// A bug in the original Satoshi client implementation means one more
	// stack value than should be used must be popped.  Unfortunately, this
	// buggy behavior is now part of the consensus and a hard fork would be
	// required to fix it.
	dummy, err := vm.dstack.PopByteArray()
	if err != nil {
		return err
	}

	// Since the dummy argument is otherwise not checked, it could be any
	// value which unfortunately provides a source of malleability.  Thus,
	// there is a script flag to force an error when the value is NOT 0.
	if vm.hasFlag(ScriptStrictMultiSig) && len(dummy) != 0 {
		str := fmt.Sprintf("multisig dummy argument has length %d "+
			"instead of 0", len(dummy))
		return txscripterr.ScriptError(txscripterr.ErrSigNullDummy, str)
	}

	// Get script starting from the most recent OP_CODESEPARATOR.
	script := vm.subScript()

	// Remove the signature in pre version 0 segwit scripts since there is
	// no way for a signature to sign itself.
	if !vm.isWitnessVersionActive(0) {
		for _, sigInfo := range signatures {
			script = removeOpcodeByData(script, sigInfo.signature)
		}
	}

	success := true
	numPubKeys++
	pubKeyIdx := -1
	signatureIdx := 0
	for numSignatures > 0 {
		// When there are more signatures than public keys remaining,
		// there is no way to succeed since too many signatures are
		// invalid, so exit early.
		pubKeyIdx++
		numPubKeys--
		if numSignatures > numPubKeys {
			success = false
			break
		}

		sigInfo := signatures[signatureIdx]
		pubKey := pubKeys[pubKeyIdx]

		// The order of the signature and public key evaluation is
		// important here since it can be distinguished by an
		// OP_CHECKMULTISIG NOT when the strict encoding flag is set.

		rawSig := sigInfo.signature
		if len(rawSig) == 0 {
			// Skip to the next pubkey if signature is empty.
			continue
		}

		// Split the signature into hash type and signature components.
		hashType := params.SigHashType(rawSig[len(rawSig)-1])
		signature := rawSig[:len(rawSig)-1]

		// Only parse and check the signature encoding once.
		var parsedSig *btcec.Signature
		if !sigInfo.parsed {
			if err := vm.checkHashTypeEncoding(hashType); err != nil {
				return err
			}
			if err := vm.checkSignatureEncoding(signature); err != nil {
				return err
			}

			// Parse the signature.
			var err er.R
			if vm.hasFlag(ScriptVerifyStrictEncoding) ||
				vm.hasFlag(ScriptVerifyDERSignatures) {

				parsedSig, err = btcec.ParseDERSignature(signature,
					btcec.S256())
			} else {
				parsedSig, err = btcec.ParseSignature(signature,
					btcec.S256())
			}
			sigInfo.parsed = true
			if err != nil {
				continue
			}
			sigInfo.parsedSignature = parsedSig
		} else {
			// Skip to the next pubkey if the signature is invalid.
			if sigInfo.parsedSignature == nil {
				continue
			}

			// Use the already parsed signature.
			parsedSig = sigInfo.parsedSignature
		}

		if err := vm.checkPubKeyEncoding(pubKey); err != nil {
			return err
		}

		// Parse the pubkey.
		parsedPubKey, err := btcec.ParsePubKey(pubKey, btcec.S256())
		if err != nil {
			continue
		}

		// Generate the signature hash based on the signature hash type.
		var hash []byte
		if vm.isWitnessVersionActive(0) {
			var sigHashes *TxSigHashes
			if vm.hashCache != nil {
				sigHashes = vm.hashCache
			} else {
				sigHashes = NewTxSigHashes(&vm.tx)
			}

			hash, err = calcWitnessSignatureHash(script, sigHashes, hashType,
				&vm.tx, vm.txIdx, vm.inputAmount)
			if err != nil {
				return err
			}
		} else {
			hash = calcSignatureHash(script, hashType, &vm.tx, vm.txIdx)
		}

		var valid bool
		if vm.sigCache != nil {
			var sigHash chainhash.Hash
			copy(sigHash[:], hash)

			valid = vm.sigCache.Exists(sigHash, parsedSig, parsedPubKey)
			if !valid && parsedSig.Verify(hash, parsedPubKey) {
				vm.sigCache.Add(sigHash, parsedSig, parsedPubKey)
				valid = true
			}
		} else {
			valid = parsedSig.Verify(hash, parsedPubKey)
		}

		if valid {
			// PubKey verified, move on to the next signature.
			signatureIdx++
			numSignatures--
		}
	}

	if !success && vm.hasFlag(ScriptVerifyNullFail) {
		for _, sig := range signatures {
			if len(sig.signature) > 0 {
				str := "not all signatures empty on failed checkmultisig"
				return txscripterr.ScriptError(txscripterr.ErrNullFail, str)
			}
		}
	}

	vm.dstack.PushBool(success)
	return nil
}

// opcodeCheckMultiSigVerify is a combination of opcodeCheckMultiSig and
// opcodeVerify.  The opcodeCheckMultiSig is invoked followed by opcodeVerify.
// See the documentation for each of those opcodes for more details.
//
// Stack transformation:
// [... dummy [sig ...] numsigs [pubkey ...] numpubkeys] -> [... bool] -> [...]
func opcodeCheckMultiSigVerify(op *parsescript.ParsedOpcode, vm *Engine) er.R {
	err := opcodeCheckMultiSig(op, vm)
	if err == nil {
		err = abstractVerify(op, vm, txscripterr.ErrCheckMultiSigVerify)
	}
	return err
}

// OpcodeByName is a map that can be used to lookup an opcode by its
// human-readable name (OP_CHECKMULTISIG, OP_CHECKSIG, etc).
var OpcodeByName = make(map[string]byte)

func init() {
	// Initialize the opcode name to value map using the contents of the
	// opcode array.  Also add entries for "OP_FALSE", "OP_TRUE", and
	// "OP_NOP2" since they are aliases for "OP_0", "OP_1",
	// and "OP_CHECKLOCKTIMEVERIFY" respectively.
	for i := 0; i < 256; i++ {
		OpcodeByName[opcode.OpcodeName(byte(i))] = byte(i)
	}
	OpcodeByName["OP_FALSE"] = opcode.OP_FALSE
	OpcodeByName["OP_TRUE"] = opcode.OP_TRUE
	OpcodeByName["OP_NOP2"] = opcode.OP_CHECKLOCKTIMEVERIFY
	OpcodeByName["OP_NOP3"] = opcode.OP_CHECKSEQUENCEVERIFY
}

// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"fmt"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/btcutil/util"
	"github.com/pkt-cash/pktd/txscript/opcode"
	"github.com/pkt-cash/pktd/txscript/parsescript"
	"github.com/pkt-cash/pktd/txscript/scriptbuilder"
	"github.com/pkt-cash/pktd/txscript/txscripterr"

	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/wire"
)

const (
	// MaxDataCarrierSize is the maximum number of bytes allowed in pushed
	// data to be considered a nulldata transaction
	MaxDataCarrierSize = 80

	// StandardVerifyFlags are the script flags which are used when
	// executing transaction scripts to enforce additional checks which
	// are required for the script to be considered standard.  These checks
	// help reduce issues related to transaction malleability as well as
	// allow pay-to-script hash transactions.  Note these flags are
	// different than what is required for the consensus rules in that they
	// are more strict.
	//
	// TODO: This definition does not belong here.  It belongs in a policy
	// package.
	StandardVerifyFlags = ScriptBip16 |
		ScriptVerifyDERSignatures |
		ScriptVerifyStrictEncoding |
		ScriptVerifyMinimalData |
		ScriptStrictMultiSig |
		ScriptDiscourageUpgradableNops |
		ScriptVerifyCleanStack |
		ScriptVerifyNullFail |
		ScriptVerifyCheckLockTimeVerify |
		ScriptVerifyCheckSequenceVerify |
		ScriptVerifyLowS |
		ScriptStrictMultiSig |
		ScriptVerifyWitness |
		ScriptVerifyDiscourageUpgradeableWitnessProgram |
		ScriptVerifyMinimalIf |
		ScriptVerifyWitnessPubKeyType
)

// ScriptClass is an enumeration for the list of standard types of script.
type ScriptClass byte

// Classes of script payment known about in the blockchain.
const (
	NonStandardTy         ScriptClass = iota // None of the recognized forms.
	PubKeyTy                                 // Pay pubkey.
	PubKeyHashTy                             // Pay pubkey hash.
	WitnessV0PubKeyHashTy                    // Pay witness pubkey hash.
	ScriptHashTy                             // Pay to script hash.
	WitnessV0ScriptHashTy                    // Pay to witness script hash.
	MultiSigTy                               // Multi signature.
	NullDataTy                               // Empty data-only (provably prunable).
)

// scriptClassToName houses the human-readable strings which describe each
// script class.
var scriptClassToName = []string{
	NonStandardTy:         "nonstandard",
	PubKeyTy:              "pubkey",
	PubKeyHashTy:          "pubkeyhash",
	WitnessV0PubKeyHashTy: "witness_v0_keyhash",
	ScriptHashTy:          "scripthash",
	WitnessV0ScriptHashTy: "witness_v0_scripthash",
	MultiSigTy:            "multisig",
	NullDataTy:            "nulldata",
}

// String implements the Stringer interface by returning the name of
// the enum script class. If the enum is invalid then "Invalid" will be
// returned.
func (t ScriptClass) String() string {
	if int(t) > len(scriptClassToName) || int(t) < 0 {
		return "Invalid"
	}
	return scriptClassToName[t]
}

// IsSegwit returns true if the script is a known segwit type.
func (t ScriptClass) IsSegwit() bool {
	return t == WitnessV0PubKeyHashTy || t == WitnessV0ScriptHashTy
}

// isPubkey returns true if the script passed is a pay-to-pubkey transaction,
// false otherwise.
func isPubkey(pops []parsescript.ParsedOpcode) bool {
	// Valid pubkeys are either 33 or 65 bytes.
	return len(pops) == 2 &&
		(len(pops[0].Data) == 33 || len(pops[0].Data) == 65) &&
		pops[1].Opcode.Value == opcode.OP_CHECKSIG
}

// isPubkeyHash returns true if the script passed is a pay-to-pubkey-hash
// transaction, false otherwise.
func isPubkeyHash(pops []parsescript.ParsedOpcode) bool {
	return len(pops) == 5 &&
		pops[0].Opcode.Value == opcode.OP_DUP &&
		pops[1].Opcode.Value == opcode.OP_HASH160 &&
		pops[2].Opcode.Value == opcode.OP_DATA_20 &&
		pops[3].Opcode.Value == opcode.OP_EQUALVERIFY &&
		pops[4].Opcode.Value == opcode.OP_CHECKSIG
}

// isMultiSig returns true if the passed script is a multisig transaction, false
// otherwise.
func isMultiSig(pops []parsescript.ParsedOpcode) bool {
	// The absolute minimum is 1 pubkey:
	// OP_0/OP_1-16 <pubkey> OP_1 OP_CHECKMULTISIG
	l := len(pops)
	if l < 4 {
		return false
	}
	if !isSmallInt(pops[0].Opcode) {
		return false
	}
	if !isSmallInt(pops[l-2].Opcode) {
		return false
	}
	if pops[l-1].Opcode.Value != opcode.OP_CHECKMULTISIG {
		return false
	}

	// Verify the number of pubkeys specified matches the actual number
	// of pubkeys provided.
	if l-2-1 != asSmallInt(pops[l-2].Opcode) {
		return false
	}

	for _, pop := range pops[1 : l-2] {
		// Valid pubkeys are either 33 or 65 bytes.
		if len(pop.Data) != 33 && len(pop.Data) != 65 {
			return false
		}
	}
	return true
}

// isNullData returns true if the passed script is a null data transaction,
// false otherwise.
func isNullData(pops []parsescript.ParsedOpcode) bool {
	// A nulldata transaction is either a single OP_RETURN or an
	// OP_RETURN SMALLDATA (where SMALLDATA is a data push up to
	// MaxDataCarrierSize bytes).
	l := len(pops)
	if l == 1 && pops[0].Opcode.Value == opcode.OP_RETURN {
		return true
	}

	return l == 2 &&
		pops[0].Opcode.Value == opcode.OP_RETURN &&
		(isSmallInt(pops[1].Opcode) || pops[1].Opcode.Value <=
			opcode.OP_PUSHDATA4) &&
		len(pops[1].Data) <= MaxDataCarrierSize
}

// scriptType returns the type of the script being inspected from the known
// standard types.
func typeOfScript(pops []parsescript.ParsedOpcode) ScriptClass {
	if isPubkey(pops) {
		return PubKeyTy
	} else if isPubkeyHash(pops) {
		return PubKeyHashTy
	} else if isWitnessPubKeyHash(pops) {
		return WitnessV0PubKeyHashTy
	} else if isScriptHash(pops) {
		return ScriptHashTy
	} else if isWitnessScriptHash(pops) {
		return WitnessV0ScriptHashTy
	} else if isMultiSig(pops) {
		return MultiSigTy
	} else if isNullData(pops) {
		return NullDataTy
	}
	return NonStandardTy
}

// GetScriptClass returns the class of the script passed.
//
// NonStandardTy will be returned when the script does not parse.
func GetScriptClass(script []byte) ScriptClass {
	pops, err := parsescript.ParseScript(script)
	if err != nil {
		return NonStandardTy
	}
	pops = stripVote(pops)
	return typeOfScript(pops)
}

// expectedInputs returns the number of arguments required by a script.
// If the script is of unknown type such that the number can not be determined
// then -1 is returned. We are an internal function and thus assume that class
// is the real class of pops (and we can thus assume things that were determined
// while finding out the type).
func expectedInputs(pops []parsescript.ParsedOpcode, class ScriptClass) int {
	switch class {
	case PubKeyTy:
		return 1

	case PubKeyHashTy:
		return 2

	case WitnessV0PubKeyHashTy:
		return 2

	case ScriptHashTy:
		// Not including script.  That is handled by the caller.
		return 1

	case WitnessV0ScriptHashTy:
		// Not including script.  That is handled by the caller.
		return 1

	case MultiSigTy:
		// Standard multisig has a push a small number for the number
		// of sigs and number of keys.  Check the first push instruction
		// to see how many arguments are expected. typeOfScript already
		// checked this so we know it'll be a small int.  Also, due to
		// the original bitcoind bug where OP_CHECKMULTISIG pops an
		// additional item from the stack, add an extra expected input
		// for the extra push that is required to compensate.
		return asSmallInt(pops[0].Opcode) + 1

	case NullDataTy:
		fallthrough
	default:
		return -1
	}
}

// ScriptInfo houses information about a script pair that is determined by
// CalcScriptInfo.
type ScriptInfo struct {
	// PkScriptClass is the class of the public key script and is equivalent
	// to calling GetScriptClass on it.
	PkScriptClass ScriptClass

	// NumInputs is the number of inputs provided by the public key script.
	NumInputs int

	// ExpectedInputs is the number of outputs required by the signature
	// script and any pay-to-script-hash scripts. The number will be -1 if
	// unknown.
	ExpectedInputs int

	// SigOps is the number of signature operations in the script pair.
	SigOps int
}

// CalcScriptInfo returns a structure providing data about the provided script
// pair.  It will error if the pair is in someway invalid such that they can not
// be analyzed, i.e. if they do not parse or the pkScript is not a push-only
// script
func CalcScriptInfo(sigScript, pkScript []byte, witness wire.TxWitness,
	bip16, segwit bool) (*ScriptInfo, er.R) {
	sigPops, err := parsescript.ParseScript(sigScript)
	if err != nil {
		return nil, err
	}

	pkPops, err := parsescript.ParseScript(pkScript)
	if err != nil {
		return nil, err
	}

	// Push only sigScript makes little sense.
	si := new(ScriptInfo)
	si.PkScriptClass = typeOfScript(pkPops)

	// Can't have a signature script that doesn't just push data.
	if !parsescript.IsPushOnly(sigPops) {
		return nil, txscripterr.ScriptError(txscripterr.ErrNotPushOnly,
			"signature script is not push only")
	}

	si.ExpectedInputs = expectedInputs(pkPops, si.PkScriptClass)

	switch {
	// Count sigops taking into account pay-to-script-hash.
	case si.PkScriptClass == ScriptHashTy && bip16 && !segwit:
		// The pay-to-hash-script is the final data push of the
		// signature script.
		script := sigPops[len(sigPops)-1].Data
		shPops, err := parsescript.ParseScript(script)
		if err != nil {
			return nil, err
		}

		shInputs := expectedInputs(shPops, typeOfScript(shPops))
		if shInputs == -1 {
			si.ExpectedInputs = -1
		} else {
			si.ExpectedInputs += shInputs
		}
		si.SigOps = getSigOpCount(shPops, true)

		// All entries pushed to stack (or are OP_RESERVED and exec
		// will fail).
		si.NumInputs = len(sigPops)

	// If segwit is active, and this is a regular p2wkh output, then we'll
	// treat the script as a p2pkh output in essence.
	case si.PkScriptClass == WitnessV0PubKeyHashTy && segwit:

		si.SigOps = GetWitnessSigOpCount(sigScript, pkScript, witness)
		si.NumInputs = len(witness)

	// We'll attempt to detect the nested p2sh case so we can accurately
	// count the signature operations involved.
	case si.PkScriptClass == ScriptHashTy &&
		IsWitnessProgram(sigScript[1:]) && bip16 && segwit:

		// Extract the pushed witness program from the sigScript so we
		// can determine the number of expected inputs.
		pkPops, _ := parsescript.ParseScript(sigScript[1:])
		shInputs := expectedInputs(pkPops, typeOfScript(pkPops))
		if shInputs == -1 {
			si.ExpectedInputs = -1
		} else {
			si.ExpectedInputs += shInputs
		}

		si.SigOps = GetWitnessSigOpCount(sigScript, pkScript, witness)

		si.NumInputs = len(witness)
		si.NumInputs += len(sigPops)

	// If segwit is active, and this is a p2wsh output, then we'll need to
	// examine the witness script to generate accurate script info.
	case si.PkScriptClass == WitnessV0ScriptHashTy && segwit:
		// The witness script is the final element of the witness
		// stack.
		witnessScript := witness[len(witness)-1]
		pops, _ := parsescript.ParseScript(witnessScript)

		shInputs := expectedInputs(pops, typeOfScript(pops))
		if shInputs == -1 {
			si.ExpectedInputs = -1
		} else {
			si.ExpectedInputs += shInputs
		}

		si.SigOps = GetWitnessSigOpCount(sigScript, pkScript, witness)
		si.NumInputs = len(witness)

	default:
		si.SigOps = getSigOpCount(pkPops, true)

		// All entries pushed to stack (or are OP_RESERVED and exec
		// will fail).
		si.NumInputs = len(sigPops)
	}

	return si, nil
}

// CalcMultiSigStats returns the number of public keys and signatures from
// a multi-signature transaction script.  The passed script MUST already be
// known to be a multi-signature script.
func CalcMultiSigStats(script []byte) (int, int, er.R) {
	pops, err := parsescript.ParseScript(script)
	if err != nil {
		return 0, 0, err
	}

	// A multi-signature script is of the pattern:
	//  NUM_SIGS PUBKEY PUBKEY PUBKEY... NUM_PUBKEYS OP_CHECKMULTISIG
	// Therefore the number of signatures is the oldest item on the stack
	// and the number of pubkeys is the 2nd to last.  Also, the absolute
	// minimum for a multi-signature script is 1 pubkey, so at least 4
	// items must be on the stack per:
	//  OP_1 PUBKEY OP_1 OP_CHECKMULTISIG
	if len(pops) < 4 {
		str := fmt.Sprintf("script %x is not a multisig script", script)
		return 0, 0, txscripterr.ScriptError(txscripterr.ErrNotMultisigScript, str)
	}

	numSigs := asSmallInt(pops[0].Opcode)
	numPubKeys := asSmallInt(pops[len(pops)-2].Opcode)
	return numPubKeys, numSigs, nil
}

// payToPubKeyHashScriptBuilder creates a new script to pay a transaction
// output to a 20-byte pubkey hash. It is expected that the input is a valid
// hash.
func payToPubKeyHashScriptBuilder(pubKeyHash []byte) *scriptbuilder.ScriptBuilder {
	return scriptbuilder.NewScriptBuilder().AddOp(opcode.OP_DUP).AddOp(opcode.OP_HASH160).
		AddData(pubKeyHash).AddOp(opcode.OP_EQUALVERIFY).AddOp(opcode.OP_CHECKSIG)
}

func payToPubKeyHashScript(pubKeyHash []byte) ([]byte, er.R) {
	return payToPubKeyHashScriptBuilder(pubKeyHash).Script()
}

// payToWitnessPubKeyHashScriptBuilder creates a new script to pay to a version 0
// pubkey hash witness program. The passed hash is expected to be valid.
func payToWitnessPubKeyHashScriptBuilder(pubKeyHash []byte) *scriptbuilder.ScriptBuilder {
	return scriptbuilder.NewScriptBuilder().AddOp(opcode.OP_0).AddData(pubKeyHash)
}

func payToWitnessPubKeyHashScript(pubKeyHash []byte) ([]byte, er.R) {
	return payToWitnessPubKeyHashScriptBuilder(pubKeyHash).Script()
}

// payToScriptHashScriptBuilder creates a new script to pay a transaction output to a
// script hash. It is expected that the input is a valid hash.
func payToScriptHashScriptBuilder(scriptHash []byte) *scriptbuilder.ScriptBuilder {
	return scriptbuilder.NewScriptBuilder().AddOp(opcode.OP_HASH160).AddData(scriptHash).
		AddOp(opcode.OP_EQUAL)
}

func payToScriptHashScript(scriptHash []byte) ([]byte, er.R) {
	return payToScriptHashScriptBuilder(scriptHash).Script()
}

// payToWitnessScriptHashScriptBuilder creates a new script to pay to a version 0
// script hash witness program. The passed hash is expected to be valid.
func payToWitnessScriptHashScriptBuilder(scriptHash []byte) *scriptbuilder.ScriptBuilder {
	return scriptbuilder.NewScriptBuilder().AddOp(opcode.OP_0).AddData(scriptHash)
}

func payToWitnessScriptHashScript(scriptHash []byte) ([]byte, er.R) {
	return payToWitnessScriptHashScriptBuilder(scriptHash).Script()
}

// payToPubKeyScriptBuilder creates a new script to pay a transaction output to a
// public key. It is expected that the input is a valid pubkey.
func payToPubKeyScriptBuilder(serializedPubKey []byte) *scriptbuilder.ScriptBuilder {
	return scriptbuilder.NewScriptBuilder().AddData(serializedPubKey).
		AddOp(opcode.OP_CHECKSIG)
}

// appendVote adds a vote to the end of a ScriptBuilder if a voteFor and/or voteAgainst
// is specified. If neither is specified then it will not alter the ScriptBuilder at
// all. The result is the same ScriptBuilder which was passed in.
func appendVote(sb *scriptbuilder.ScriptBuilder, voteFor, voteAgainst []byte) *scriptbuilder.ScriptBuilder {
	if voteFor == nil && voteAgainst == nil {
		return sb
	}
	if voteFor != nil {
		sb = sb.AddData(voteFor)
	} else {
		sb = sb.AddOp(opcode.OP_0)
	}
	if voteAgainst != nil {
		sb = sb.AddData(voteAgainst)
	} else {
		sb = sb.AddOp(opcode.OP_0)
	}
	return sb.AddOp(opcode.OP_VOTE)
}

func payToNonStandardScriptBuilder(pkScript, voteFor, voteAgainst []byte) ([]byte, er.R) {
	pops, err := parsescript.ParseScript(pkScript)
	if err != nil {
		// There's an error parsing, lets not touch it
		return pkScript, nil
	}

	pops = stripVote(pops)
	scriptClass := typeOfScript(pops)
	if scriptClass.IsSegwit() {
		// It's not possible to append votes to a segwit script
		return pkScript, nil
	}

	vf, va := ElectionGetVotesForAgainst(pkScript)
	if vf != nil || va != nil {
		// Respect first the vote which is in the address itself
		return pkScript, nil
	}

	// Append our vote to the script address
	sb := scriptbuilder.ScriptBuilder{ScriptInt: util.CloneBytes(pkScript)}
	return appendVote(&sb, voteFor, voteAgainst).Script()
}

// PayToAddrScriptWithVote creates a new script to pay a transaction output to a the
// specified address and adds a vote for the specified network steward, if voteFor
// and/or voteAgainst are non-null and if the address type is non-segwit.
func PayToAddrScriptWithVote(addr btcutil.Address, voteFor, voteAgainst []byte) ([]byte, er.R) {
	if util.IsNil(addr) {
		return nil, txscripterr.ScriptError(txscripterr.ErrUnsupportedAddress,
			"unable to generate payment script for nil address")
	}

	switch addr := addr.(type) {
	case *btcutil.AddressPubKeyHash:
		return appendVote(payToPubKeyHashScriptBuilder(addr.ScriptAddress()),
			voteFor, voteAgainst).Script()

	case *btcutil.AddressScriptHash:
		return appendVote(payToScriptHashScriptBuilder(addr.ScriptAddress()),
			voteFor, voteAgainst).Script()

	case *btcutil.AddressPubKey:
		return appendVote(payToPubKeyScriptBuilder(addr.ScriptAddress()),
			voteFor, voteAgainst).Script()

	case *btcutil.AddressWitnessPubKeyHash:
		return payToWitnessPubKeyHashScript(addr.ScriptAddress())

	case *btcutil.AddressWitnessScriptHash:
		return payToWitnessScriptHashScript(addr.ScriptAddress())

	case *btcutil.AddressNonStandard:
		return payToNonStandardScriptBuilder(addr.ScriptAddress(), voteFor, voteAgainst)
	}

	str := fmt.Sprintf("unable to generate payment script for unsupported "+
		"address type %T", addr)
	return nil, txscripterr.ScriptError(txscripterr.ErrUnsupportedAddress, str)
}

// PayToAddrScript creates a new script to pay a transaction output to the
// specified address.
func PayToAddrScript(addr btcutil.Address) ([]byte, er.R) {
	return PayToAddrScriptWithVote(addr, nil, nil)
}

// stripVote removes any votes from a script so that it will appear as a canonical
// transaction.
func stripVote(pops []parsescript.ParsedOpcode) []parsescript.ParsedOpcode {
	out := make([]parsescript.ParsedOpcode, 0, len(pops))
	for _, pop := range pops {
		out = append(out, pop)
		if pop.Opcode.Value != opcode.OP_VOTE {
			continue
		}
		if len(out) < 3 {
			continue
		}
		last := out[len(out)-2]
		secondLast := out[len(out)-3]
		if last.Opcode.Value != opcode.OP_0 && !canonicalPush(last) {
			continue
		}
		if secondLast.Opcode.Value != opcode.OP_0 && !canonicalPush(secondLast) {
			continue
		}
		// pop second op, last op, vote
		out = out[:len(out)-3]
	}
	return out
}

// NullDataScript creates a provably-prunable script containing OP_RETURN
// followed by the passed data.  An Error with the error code ErrTooMuchNullData
// will be returned if the length of the passed data exceeds MaxDataCarrierSize.
func NullDataScript(data []byte) ([]byte, er.R) {
	if len(data) > MaxDataCarrierSize {
		str := fmt.Sprintf("data size %d is larger than max "+
			"allowed size %d", len(data), MaxDataCarrierSize)
		return nil, txscripterr.ScriptError(txscripterr.ErrTooMuchNullData, str)
	}

	return scriptbuilder.NewScriptBuilder().AddOp(opcode.OP_RETURN).AddData(data).Script()
}

// MultiSigScript returns a valid script for a multisignature redemption where
// nrequired of the keys in pubkeys are required to have signed the transaction
// for success.  An Error with the error code ErrTooManyRequiredSigs will be
// returned if nrequired is larger than the number of keys provided.
func MultiSigScript(pubkeys []*btcutil.AddressPubKey, nrequired int) ([]byte, er.R) {
	if len(pubkeys) < nrequired {
		str := fmt.Sprintf("unable to generate multisig script with "+
			"%d required signatures when there are only %d public "+
			"keys available", nrequired, len(pubkeys))
		return nil, txscripterr.ScriptError(txscripterr.ErrTooManyRequiredSigs, str)
	}

	builder := scriptbuilder.NewScriptBuilder().AddInt64(int64(nrequired))
	for _, key := range pubkeys {
		builder.AddData(key.ScriptAddress())
	}
	builder.AddInt64(int64(len(pubkeys)))
	builder.AddOp(opcode.OP_CHECKMULTISIG)

	return builder.Script()
}

// PushedData returns an array of byte slices containing any pushed data found
// in the passed script.  This includes OP_0, but not OP_1 - OP_16.
func PushedData(script []byte) ([][]byte, er.R) {
	pops, err := parsescript.ParseScript(script)
	if err != nil {
		return nil, err
	}

	var data [][]byte
	for _, pop := range pops {
		if pop.Data != nil {
			data = append(data, pop.Data)
		} else if pop.Opcode.Value == opcode.OP_0 {
			data = append(data, nil)
		}
	}
	return data, nil
}

// ExtractPkScriptAddrs returns the type of script, addresses and required
// signatures associated with the passed PkScript.  Note that it only works for
// 'standard' transaction script types.  Any data such as public keys which are
// invalid are omitted from the results.
func ExtractPkScriptAddrs(pkScript []byte, chainParams *chaincfg.Params) (ScriptClass, []btcutil.Address, int, er.R) {
	var addrs []btcutil.Address
	var requiredSigs int

	// No valid addresses or required signatures if the script doesn't
	// parse.
	pops, err := parsescript.ParseScript(pkScript)
	if err != nil {
		return NonStandardTy, nil, 0, err
	}

	pops = stripVote(pops)

	scriptClass := typeOfScript(pops)
	switch scriptClass {
	case PubKeyHashTy:
		// A pay-to-pubkey-hash script is of the form:
		//  OP_DUP OP_HASH160 <hash> OP_EQUALVERIFY OP_CHECKSIG
		// Therefore the pubkey hash is the 3rd item on the stack.
		// Skip the pubkey hash if it's invalid for some reason.
		requiredSigs = 1
		addr, err := btcutil.NewAddressPubKeyHash(pops[2].Data,
			chainParams)
		if err == nil {
			addrs = append(addrs, addr)
		}

	case WitnessV0PubKeyHashTy:
		// A pay-to-witness-pubkey-hash script is of thw form:
		//  OP_0 <20-byte hash>
		// Therefore, the pubkey hash is the second item on the stack.
		// Skip the pubkey hash if it's invalid for some reason.
		requiredSigs = 1
		addr, err := btcutil.NewAddressWitnessPubKeyHash(pops[1].Data,
			chainParams)
		if err == nil {
			addrs = append(addrs, addr)
		}

	case PubKeyTy:
		// A pay-to-pubkey script is of the form:
		//  <pubkey> OP_CHECKSIG
		// Therefore the pubkey is the first item on the stack.
		// Skip the pubkey if it's invalid for some reason.
		requiredSigs = 1
		addr, err := btcutil.NewAddressPubKey(pops[0].Data, chainParams)
		if err == nil {
			addrs = append(addrs, addr)
		}

	case ScriptHashTy:
		// A pay-to-script-hash script is of the form:
		//  OP_HASH160 <scripthash> OP_EQUAL
		// Therefore the script hash is the 2nd item on the stack.
		// Skip the script hash if it's invalid for some reason.
		requiredSigs = 1
		addr, err := btcutil.NewAddressScriptHashFromHash(pops[1].Data,
			chainParams)
		if err == nil {
			addrs = append(addrs, addr)
		}

	case WitnessV0ScriptHashTy:
		// A pay-to-witness-script-hash script is of the form:
		//  OP_0 <32-byte hash>
		// Therefore, the script hash is the second item on the stack.
		// Skip the script hash if it's invalid for some reason.
		requiredSigs = 1
		addr, err := btcutil.NewAddressWitnessScriptHash(pops[1].Data,
			chainParams)
		if err == nil {
			addrs = append(addrs, addr)
		}

	case MultiSigTy:
		// A multi-signature script is of the form:
		//  <numsigs> <pubkey> <pubkey> <pubkey>... <numpubkeys> OP_CHECKMULTISIG
		// Therefore the number of required signatures is the 1st item
		// on the stack and the number of public keys is the 2nd to last
		// item on the stack.
		requiredSigs = asSmallInt(pops[0].Opcode)
		numPubKeys := asSmallInt(pops[len(pops)-2].Opcode)

		// Extract the public keys while skipping any that are invalid.
		addrs = make([]btcutil.Address, 0, numPubKeys)
		for i := 0; i < numPubKeys; i++ {
			addr, err := btcutil.NewAddressPubKey(pops[i+1].Data,
				chainParams)
			if err == nil {
				addrs = append(addrs, addr)
			}
		}

	case NullDataTy:
		// Null data transactions have no addresses or required
		// signatures.

	case NonStandardTy:
		// Don't attempt to extract addresses or required signatures for
		// nonstandard transactions.
	}

	return scriptClass, addrs, requiredSigs, nil
}

// PkScriptToAddress returns the address corresponding to a script.
// Because most multi-signature scripts are segwit and are thus able to be represented as
// addresses, most scripts are able to be represented directly as addresses, but if there
// is a script which is not directly parsable, this function will return "script:" followed
// by a base-64 representation of the pkScript itself such that it can be decoded later.
func PkScriptToAddress(pkScript []byte, chainParams *chaincfg.Params) btcutil.Address {
	_, addrs, requiredSigs, err := ExtractPkScriptAddrs(pkScript, chainParams)
	if err != nil || len(addrs) != 1 || requiredSigs != 1 {
		return btcutil.NewAddressNonStandard(pkScript)
	}
	return addrs[0]
}

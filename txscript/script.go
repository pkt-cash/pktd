// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"bytes"
	"encoding/binary"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/txscript/opcode"
	"github.com/pkt-cash/pktd/txscript/params"
	"github.com/pkt-cash/pktd/txscript/parsescript"

	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/wire"
)

// isSmallInt returns whether or not the opcode is considered a small integer,
// which is an OP_0, or OP_1 through OP_16.
func isSmallInt(op opcode.Opcode) bool {
	if op.Value == opcode.OP_0 || (op.Value >= opcode.OP_1 && op.Value <= opcode.OP_16) {
		return true
	}
	return false
}

// isScriptHash returns true if the script passed is a pay-to-script-hash
// transaction, false otherwise.
func isScriptHash(pops []parsescript.ParsedOpcode) bool {
	return len(pops) == 3 &&
		pops[0].Opcode.Value == opcode.OP_HASH160 &&
		pops[1].Opcode.Value == opcode.OP_DATA_20 &&
		pops[2].Opcode.Value == opcode.OP_EQUAL
}

// IsPayToScriptHash returns true if the script is in the standard
// pay-to-script-hash (P2SH) format, false otherwise.
func IsPayToScriptHash(script []byte) bool {
	pops, err := parsescript.ParseScript(script)
	if err != nil {
		return false
	}
	return isScriptHash(pops)
}

// isWitnessScriptHash returns true if the passed script is a
// pay-to-witness-script-hash transaction, false otherwise.
func isWitnessScriptHash(pops []parsescript.ParsedOpcode) bool {
	return len(pops) == 2 &&
		pops[0].Opcode.Value == opcode.OP_0 &&
		pops[1].Opcode.Value == opcode.OP_DATA_32
}

// ElectionGetVotesForAgainst gets the candidates who are voted for and voted against
// by the provided pkScript
func ElectionGetVotesForAgainst(pkScript []byte) (voteFor []byte, voteAgainst []byte) {
	pops, err := parsescript.ParseScript(pkScript)
	if err != nil {
		return nil, nil
	}
	for i, op := range pops {
		if op.Opcode.Value != opcode.OP_VOTE {
			continue
		}
		if i < 2 {
			// invalid, too short
			continue
		}
		if pops[i-1].Opcode.Value == opcode.OP_0 {
		} else if canonicalPush(pops[i-1]) && len(pops[i-1].Data) > 0 && len(pops[i-1].Data) < 80 {
			voteAgainst = pops[i-1].Data
		} else {
			continue
		}
		if pops[i-2].Opcode.Value == opcode.OP_0 {
		} else if canonicalPush(pops[i-2]) && len(pops[i-2].Data) > 0 && len(pops[i-2].Data) < 80 {
			voteFor = pops[i-2].Data
		} else {
			voteAgainst = nil
			continue
		}
		break
	}
	return
}

// IsPayToWitnessScriptHash returns true if the is in the standard
// pay-to-witness-script-hash (P2WSH) format, false otherwise.
func IsPayToWitnessScriptHash(script []byte) bool {
	pops, err := parsescript.ParseScript(script)
	if err != nil {
		return false
	}
	return isWitnessScriptHash(pops)
}

// IsPayToWitnessPubKeyHash returns true if the is in the standard
// pay-to-witness-pubkey-hash (P2WKH) format, false otherwise.
func IsPayToWitnessPubKeyHash(script []byte) bool {
	pops, err := parsescript.ParseScript(script)
	if err != nil {
		return false
	}
	return isWitnessPubKeyHash(pops)
}

// isWitnessPubKeyHash returns true if the passed script is a
// pay-to-witness-pubkey-hash, and false otherwise.
func isWitnessPubKeyHash(pops []parsescript.ParsedOpcode) bool {
	return len(pops) == 2 &&
		pops[0].Opcode.Value == opcode.OP_0 &&
		pops[1].Opcode.Value == opcode.OP_DATA_20
}

// IsWitnessProgram returns true if the passed script is a valid witness
// program which is encoded according to the passed witness program version. A
// witness program must be a small integer (from 0-16), followed by 2-40 bytes
// of pushed data.
func IsWitnessProgram(script []byte) bool {
	// The length of the script must be between 4 and 42 bytes. The
	// smallest program is the witness version, followed by a data push of
	// 2 bytes.  The largest allowed witness program has a data push of
	// 40-bytes.
	if len(script) < 4 || len(script) > 42 {
		return false
	}

	pops, err := parsescript.ParseScript(script)
	if err != nil {
		return false
	}

	return isWitnessProgram(pops)
}

// isWitnessProgram returns true if the passed script is a witness program, and
// false otherwise. A witness program MUST adhere to the following constraints:
// there must be exactly two pops (program version and the program itself), the
// first opcode MUST be a small integer (0-16), the push data MUST be
// canonical, and finally the size of the push data must be between 2 and 40
// bytes.
func isWitnessProgram(pops []parsescript.ParsedOpcode) bool {
	return len(pops) == 2 &&
		isSmallInt(pops[0].Opcode) &&
		canonicalPush(pops[1]) &&
		(len(pops[1].Data) >= 2 && len(pops[1].Data) <= 40)
}

// ExtractWitnessProgramInfo attempts to extract the witness program version,
// as well as the witness program itself from the passed script.
func ExtractWitnessProgramInfo(script []byte) (int, []byte, er.R) {
	pops, err := parsescript.ParseScript(script)
	if err != nil {
		return 0, nil, err
	}

	// If at this point, the scripts doesn't resemble a witness program,
	// then we'll exit early as there isn't a valid version or program to
	// extract.
	if !isWitnessProgram(pops) {
		return 0, nil, er.Errorf("script is not a witness program, " +
			"unable to extract version or witness program")
	}

	witnessVersion := asSmallInt(pops[0].Opcode)
	witnessProgram := pops[1].Data

	return witnessVersion, witnessProgram, nil
}

// IsPushOnlyScript returns whether or not the passed script only pushes data.
//
// False will be returned when the script does not parse.
func IsPushOnlyScript(script []byte) bool {
	pops, err := parsescript.ParseScript(script)
	if err != nil {
		return false
	}
	return parsescript.IsPushOnly(pops)
}

// unparseScript reversed the action of parseScript and returns the
// parsedOpcodes as a list of bytes
func unparseScript(pops []parsescript.ParsedOpcode) ([]byte, er.R) {
	script := make([]byte, 0, len(pops))
	for _, pop := range pops {
		b, err := popBytes(&pop)
		if err != nil {
			return nil, err
		}
		script = append(script, b...)
	}
	return script, nil
}

// DisasmString formats a disassembled script for one line printing.  When the
// script fails to parse, the returned string will contain the disassembled
// script up to the point the failure occurred along with the string '[error]'
// appended.  In addition, the reason the script failed to parse is returned
// if the caller wants more information about the failure.
func DisasmString(buf []byte) (string, er.R) {
	var disbuf bytes.Buffer
	opcodes, err := parsescript.ParseScript(buf)
	for _, pop := range opcodes {
		disbuf.WriteString(popPrint(&pop, true))
		disbuf.WriteByte(' ')
	}
	if disbuf.Len() > 0 {
		disbuf.Truncate(disbuf.Len() - 1)
	}
	if err != nil {
		disbuf.WriteString("[error]")
	}
	return disbuf.String(), err
}

// removeOpcode will remove any opcode matching ``opcode'' from the opcode
// stream in pkscript
func removeOpcode(pkscript []parsescript.ParsedOpcode, opcode byte) []parsescript.ParsedOpcode {
	retScript := make([]parsescript.ParsedOpcode, 0, len(pkscript))
	for _, pop := range pkscript {
		if pop.Opcode.Value != opcode {
			retScript = append(retScript, pop)
		}
	}
	return retScript
}

// canonicalPush returns true if the object is either not a push instruction
// or the push instruction contained wherein is matches the canonical form
// or using the smallest instruction to do the job. False otherwise.
func canonicalPush(pop parsescript.ParsedOpcode) bool {
	op := pop.Opcode.Value
	data := pop.Data
	dataLen := len(pop.Data)
	if op > opcode.OP_16 {
		return true
	}

	if op < opcode.OP_PUSHDATA1 && op > opcode.OP_0 && (dataLen == 1 && data[0] <= 16) {
		return false
	}
	if op == opcode.OP_PUSHDATA1 && dataLen < opcode.OP_PUSHDATA1 {
		return false
	}
	if op == opcode.OP_PUSHDATA2 && dataLen <= 0xff {
		return false
	}
	if op == opcode.OP_PUSHDATA4 && dataLen <= 0xffff {
		return false
	}
	return true
}

// removeOpcodeByData will return the script minus any opcodes that would push
// the passed data to the stack.
func removeOpcodeByData(pkscript []parsescript.ParsedOpcode, data []byte) []parsescript.ParsedOpcode {
	retScript := make([]parsescript.ParsedOpcode, 0, len(pkscript))
	for _, pop := range pkscript {
		if !canonicalPush(pop) || !bytes.Contains(pop.Data, data) {
			retScript = append(retScript, pop)
		}
	}
	return retScript
}

// calcHashPrevOuts calculates a single hash of all the previous outputs
// (txid:index) referenced within the passed transaction. This calculated hash
// can be re-used when validating all inputs spending segwit outputs, with a
// signature hash type of SigHashAll. This allows validation to re-use previous
// hashing computation, reducing the complexity of validating SigHashAll inputs
// from  O(N^2) to O(N).
func calcHashPrevOuts(tx *wire.MsgTx) chainhash.Hash {
	var b bytes.Buffer
	for _, in := range tx.TxIn {
		// First write out the 32-byte transaction ID one of whose
		// outputs are being referenced by this input.
		b.Write(in.PreviousOutPoint.Hash[:])

		// Next, we'll encode the index of the referenced output as a
		// little endian integer.
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], in.PreviousOutPoint.Index)
		b.Write(buf[:])
	}

	return chainhash.DoubleHashH(b.Bytes())
}

// calcHashSequence computes an aggregated hash of each of the sequence numbers
// within the inputs of the passed transaction. This single hash can be re-used
// when validating all inputs spending segwit outputs, which include signatures
// using the SigHashAll sighash type. This allows validation to re-use previous
// hashing computation, reducing the complexity of validating SigHashAll inputs
// from O(N^2) to O(N).
func calcHashSequence(tx *wire.MsgTx) chainhash.Hash {
	var b bytes.Buffer
	for _, in := range tx.TxIn {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], in.Sequence)
		b.Write(buf[:])
	}

	return chainhash.DoubleHashH(b.Bytes())
}

// calcHashOutputs computes a hash digest of all outputs created by the
// transaction encoded using the wire format. This single hash can be re-used
// when validating all inputs spending witness programs, which include
// signatures using the SigHashAll sighash type. This allows computation to be
// cached, reducing the total hashing complexity from O(N^2) to O(N).
func calcHashOutputs(tx *wire.MsgTx) chainhash.Hash {
	var b bytes.Buffer
	for _, out := range tx.TxOut {
		wire.WriteTxOut(&b, 0, 0, out)
	}

	return chainhash.DoubleHashH(b.Bytes())
}

// CalcWitnessSigHash computes the sighash digest for the specified input of
// the target transaction observing the desired sig hash type.
func CalcWitnessSigHash(script []byte, sigHashes *TxSigHashes, hType params.SigHashType,
	tx *wire.MsgTx, idx int, amt int64) ([]byte, er.R) {
	parsedScript, err := parsescript.ParseScript(script)
	if err != nil {
		return nil, er.Errorf("cannot parse output script: %v", err)
	}

	return calcWitnessSignatureHash(parsedScript, sigHashes, hType, tx, idx,
		amt)
}

// calcWitnessSignatureHash computes the sighash digest of a transaction's
// segwit input using the new, optimized digest calculation algorithm defined
// in BIP0143: https://github.com/bitcoin/bips/blob/master/bip-0143.mediawiki.
// This function makes use of pre-calculated sighash fragments stored within
// the passed HashCache to eliminate duplicate hashing computations when
// calculating the final digest, reducing the complexity from O(N^2) to O(N).
// Additionally, signatures now cover the input value of the referenced unspent
// output. This allows offline, or hardware wallets to compute the exact amount
// being spent, in addition to the final transaction fee. In the case the
// wallet if fed an invalid input amount, the real sighash will differ causing
// the produced signature to be invalid.
func calcWitnessSignatureHash(subScript []parsescript.ParsedOpcode, sigHashes *TxSigHashes,
	hashType params.SigHashType, tx *wire.MsgTx, idx int, amt int64) ([]byte, er.R) {
	// As a sanity check, ensure the passed input index for the transaction
	// is valid.
	if idx > len(tx.TxIn)-1 {
		return nil, er.Errorf("idx %d but %d txins", idx, len(tx.TxIn))
	}

	// We'll utilize this buffer throughout to incrementally calculate
	// the signature hash for this transaction.
	var sigHash bytes.Buffer

	// First write out, then encode the transaction's version number.
	var bVersion [4]byte
	binary.LittleEndian.PutUint32(bVersion[:], uint32(tx.Version))
	sigHash.Write(bVersion[:])

	// Next write out the possibly pre-calculated hashes for the sequence
	// numbers of all inputs, and the hashes of the previous outs for all
	// outputs.
	var zeroHash chainhash.Hash

	// If anyone can pay isn't active, then we can use the cached
	// hashPrevOuts, otherwise we just write zeroes for the prev outs.
	if hashType&params.SigHashAnyOneCanPay == 0 {
		sigHash.Write(sigHashes.HashPrevOuts[:])
	} else {
		sigHash.Write(zeroHash[:])
	}

	// If the sighash isn't anyone can pay, single, or none, the use the
	// cached hash sequences, otherwise write all zeroes for the
	// hashSequence.
	if hashType&params.SigHashAnyOneCanPay == 0 &&
		hashType&params.SigHashMask != params.SigHashSingle &&
		hashType&params.SigHashMask != params.SigHashNone {
		sigHash.Write(sigHashes.HashSequence[:])
	} else {
		sigHash.Write(zeroHash[:])
	}

	txIn := tx.TxIn[idx]

	// Next, write the outpoint being spent.
	sigHash.Write(txIn.PreviousOutPoint.Hash[:])
	var bIndex [4]byte
	binary.LittleEndian.PutUint32(bIndex[:], txIn.PreviousOutPoint.Index)
	sigHash.Write(bIndex[:])

	if isWitnessPubKeyHash(subScript) {
		// The script code for a p2wkh is a length prefix varint for
		// the next 25 bytes, followed by a re-creation of the original
		// p2pkh pk script.
		sigHash.Write([]byte{0x19})
		sigHash.Write([]byte{opcode.OP_DUP})
		sigHash.Write([]byte{opcode.OP_HASH160})
		sigHash.Write([]byte{opcode.OP_DATA_20})
		sigHash.Write(subScript[1].Data)
		sigHash.Write([]byte{opcode.OP_EQUALVERIFY})
		sigHash.Write([]byte{opcode.OP_CHECKSIG})
	} else {
		// For p2wsh outputs, and future outputs, the script code is
		// the original script, with all code separators removed,
		// serialized with a var int length prefix.
		rawScript, _ := unparseScript(subScript)
		wire.WriteVarBytes(&sigHash, 0, rawScript)
	}

	// Next, add the input amount, and sequence number of the input being
	// signed.
	var bAmount [8]byte
	binary.LittleEndian.PutUint64(bAmount[:], uint64(amt))
	sigHash.Write(bAmount[:])
	var bSequence [4]byte
	binary.LittleEndian.PutUint32(bSequence[:], txIn.Sequence)
	sigHash.Write(bSequence[:])

	// If the current signature mode isn't single, or none, then we can
	// re-use the pre-generated hashoutputs sighash fragment. Otherwise,
	// we'll serialize and add only the target output index to the signature
	// pre-image.
	if hashType&params.SigHashSingle != params.SigHashSingle &&
		hashType&params.SigHashNone != params.SigHashNone {
		sigHash.Write(sigHashes.HashOutputs[:])
	} else if hashType&params.SigHashMask == params.SigHashSingle && idx < len(tx.TxOut) {
		var b bytes.Buffer
		wire.WriteTxOut(&b, 0, 0, tx.TxOut[idx])
		sigHash.Write(chainhash.DoubleHashB(b.Bytes()))
	} else {
		sigHash.Write(zeroHash[:])
	}

	// Finally, write out the transaction's locktime, and the sig hash
	// type.
	var bLockTime [4]byte
	binary.LittleEndian.PutUint32(bLockTime[:], tx.LockTime)
	sigHash.Write(bLockTime[:])
	var bHashType [4]byte
	binary.LittleEndian.PutUint32(bHashType[:], uint32(hashType))
	sigHash.Write(bHashType[:])

	return chainhash.DoubleHashB(sigHash.Bytes()), nil
}

// shallowCopyTx creates a shallow copy of the transaction for use when
// calculating the signature hash.  It is used over the Copy method on the
// transaction itself since that is a deep copy and therefore does more work and
// allocates much more space than needed.
func shallowCopyTx(tx *wire.MsgTx) wire.MsgTx {
	// As an additional memory optimization, use contiguous backing arrays
	// for the copied inputs and outputs and point the final slice of
	// pointers into the contiguous arrays.  This avoids a lot of small
	// allocations.
	txCopy := wire.MsgTx{
		Version:  tx.Version,
		TxIn:     make([]*wire.TxIn, len(tx.TxIn)),
		TxOut:    make([]*wire.TxOut, len(tx.TxOut)),
		LockTime: tx.LockTime,
	}
	txIns := make([]wire.TxIn, len(tx.TxIn))
	for i, oldTxIn := range tx.TxIn {
		txIns[i] = *oldTxIn
		txCopy.TxIn[i] = &txIns[i]
	}
	txOuts := make([]wire.TxOut, len(tx.TxOut))
	for i, oldTxOut := range tx.TxOut {
		txOuts[i] = *oldTxOut
		txCopy.TxOut[i] = &txOuts[i]
	}
	return txCopy
}

// CalcSignatureHash will, given a script and hash type for the current script
// engine instance, calculate the signature hash to be used for signing and
// verification.
func CalcSignatureHash(script []byte, hashType params.SigHashType, tx *wire.MsgTx, idx int) ([]byte, er.R) {
	parsedScript, err := parsescript.ParseScript(script)
	if err != nil {
		return nil, er.Errorf("cannot parse output script: %v", err)
	}
	return calcSignatureHash(parsedScript, hashType, tx, idx), nil
}

// calcSignatureHash will, given a script and hash type for the current script
// engine instance, calculate the signature hash to be used for signing and
// verification.
func calcSignatureHash(script []parsescript.ParsedOpcode, hashType params.SigHashType, tx *wire.MsgTx, idx int) []byte {
	// The SigHashSingle signature type signs only the corresponding input
	// and output (the output with the same index number as the input).
	//
	// Since transactions can have more inputs than outputs, this means it
	// is improper to use SigHashSingle on input indices that don't have a
	// corresponding output.
	//
	// A bug in the original Satoshi client implementation means specifying
	// an index that is out of range results in a signature hash of 1 (as a
	// uint256 little endian).  The original intent appeared to be to
	// indicate failure, but unfortunately, it was never checked and thus is
	// treated as the actual signature hash.  This buggy behavior is now
	// part of the consensus and a hard fork would be required to fix it.
	//
	// Due to this, care must be taken by software that creates transactions
	// which make use of SigHashSingle because it can lead to an extremely
	// dangerous situation where the invalid inputs will end up signing a
	// hash of 1.  This in turn presents an opportunity for attackers to
	// cleverly construct transactions which can steal those coins provided
	// they can reuse signatures.
	if hashType&params.SigHashMask == params.SigHashSingle && idx >= len(tx.TxOut) {
		var hash chainhash.Hash
		hash[0] = 0x01
		return hash[:]
	}

	// Remove all instances of OP_CODESEPARATOR from the script.
	script = removeOpcode(script, opcode.OP_CODESEPARATOR)

	// Make a shallow copy of the transaction, zeroing out the script for
	// all inputs that are not currently being processed.
	txCopy := shallowCopyTx(tx)
	for i := range txCopy.TxIn {
		if i == idx {
			// UnparseScript cannot fail here because removeOpcode
			// above only returns a valid script.
			sigScript, _ := unparseScript(script)
			txCopy.TxIn[idx].SignatureScript = sigScript
		} else {
			txCopy.TxIn[i].SignatureScript = nil
		}
	}

	switch hashType & params.SigHashMask {
	case params.SigHashNone:
		txCopy.TxOut = txCopy.TxOut[0:0] // Empty slice.
		for i := range txCopy.TxIn {
			if i != idx {
				txCopy.TxIn[i].Sequence = 0
			}
		}

	case params.SigHashSingle:
		// Resize output array to up to and including requested index.
		txCopy.TxOut = txCopy.TxOut[:idx+1]

		// All but current output get zeroed out.
		for i := 0; i < idx; i++ {
			txCopy.TxOut[i].Value = -1
			txCopy.TxOut[i].PkScript = nil
		}

		// Sequence on all other inputs is 0, too.
		for i := range txCopy.TxIn {
			if i != idx {
				txCopy.TxIn[i].Sequence = 0
			}
		}

	case params.SigHashOld:
		fallthrough
	case params.SigHashAll:
		fallthrough
	default:
		// Consensus treats undefined hashtypes like normal SigHashAll
		// for purposes of hash generation.
	}
	if hashType&params.SigHashAnyOneCanPay != 0 {
		txCopy.TxIn = txCopy.TxIn[idx : idx+1]
	}

	// The final hash is the double sha256 of both the serialized modified
	// transaction and the hash type (encoded as a 4-byte little-endian
	// value) appended.
	wbuf := bytes.NewBuffer(make([]byte, 0, txCopy.SerializeSizeStripped()+4))
	txCopy.SerializeNoWitness(wbuf)
	errr := binary.Write(wbuf, binary.LittleEndian, hashType)
	if errr != nil {
		panic("calcSignatureHash: binary.Write failed")
	}
	return chainhash.DoubleHashB(wbuf.Bytes())
}

// asSmallInt returns the passed opcode, which must be true according to
// isSmallInt(), as an integer.
func asSmallInt(op opcode.Opcode) int {
	if op.Value == opcode.OP_0 {
		return 0
	}

	return int(op.Value - (opcode.OP_1 - 1))
}

// getSigOpCount is the implementation function for counting the number of
// signature operations in the script provided by pops. If precise mode is
// requested then we attempt to count the number of operations for a multisig
// op. Otherwise we use the maximum.
func getSigOpCount(pops []parsescript.ParsedOpcode, precise bool) int {
	nSigs := 0
	for i, pop := range pops {
		switch pop.Opcode.Value {
		case opcode.OP_CHECKSIG:
			fallthrough
		case opcode.OP_CHECKSIGVERIFY:
			nSigs++
		case opcode.OP_CHECKMULTISIG:
			fallthrough
		case opcode.OP_CHECKMULTISIGVERIFY:
			// If we are being precise then look for familiar
			// patterns for multisig, for now all we recognize is
			// OP_1 - OP_16 to signify the number of pubkeys.
			// Otherwise, we use the max of 20.
			if precise && i > 0 &&
				pops[i-1].Opcode.Value >= opcode.OP_1 &&
				pops[i-1].Opcode.Value <= opcode.OP_16 {
				nSigs += asSmallInt(pops[i-1].Opcode)
			} else {
				nSigs += params.MaxPubKeysPerMultiSig
			}
		default:
			// Not a sigop.
		}
	}

	return nSigs
}

// GetSigOpCount provides a quick count of the number of signature operations
// in a script. a CHECKSIG operations counts for 1, and a CHECK_MULTISIG for 20.
// If the script fails to parse, then the count up to the point of failure is
// returned.
func GetSigOpCount(script []byte) int {
	// Don't check error since parseScript returns the parsed-up-to-error
	// list of pops.
	pops, _ := parsescript.ParseScript(script)
	return getSigOpCount(pops, false)
}

// GetPreciseSigOpCount returns the number of signature operations in
// scriptPubKey.  If bip16 is true then scriptSig may be searched for the
// Pay-To-Script-Hash script in order to find the precise number of signature
// operations in the transaction.  If the script fails to parse, then the count
// up to the point of failure is returned.
func GetPreciseSigOpCount(scriptSig, scriptPubKey []byte, bip16 bool) int {
	// Don't check error since parseScript returns the parsed-up-to-error
	// list of pops.
	pops, _ := parsescript.ParseScript(scriptPubKey)

	// Treat non P2SH transactions as normal.
	if !(bip16 && isScriptHash(pops)) {
		return getSigOpCount(pops, true)
	}

	// The public key script is a pay-to-script-hash, so parse the signature
	// script to get the final item.  Scripts that fail to fully parse count
	// as 0 signature operations.
	sigPops, err := parsescript.ParseScript(scriptSig)
	if err != nil {
		return 0
	}

	// The signature script must only push data to the stack for P2SH to be
	// a valid pair, so the signature operation count is 0 when that is not
	// the case.
	if !parsescript.IsPushOnly(sigPops) || len(sigPops) == 0 {
		return 0
	}

	// The P2SH script is the last item the signature script pushes to the
	// stack.  When the script is empty, there are no signature operations.
	shScript := sigPops[len(sigPops)-1].Data
	if len(shScript) == 0 {
		return 0
	}

	// Parse the P2SH script and don't check the error since parseScript
	// returns the parsed-up-to-error list of pops and the consensus rules
	// dictate signature operations are counted up to the first parse
	// failure.
	shPops, _ := parsescript.ParseScript(shScript)
	return getSigOpCount(shPops, true)
}

// GetWitnessSigOpCount returns the number of signature operations generated by
// spending the passed pkScript with the specified witness, or sigScript.
// Unlike GetPreciseSigOpCount, this function is able to accurately count the
// number of signature operations generated by spending witness programs, and
// nested p2sh witness programs. If the script fails to parse, then the count
// up to the point of failure is returned.
func GetWitnessSigOpCount(sigScript, pkScript []byte, witness wire.TxWitness) int {
	// If this is a regular witness program, then we can proceed directly
	// to counting its signature operations without any further processing.
	if IsWitnessProgram(pkScript) {
		return getWitnessSigOps(pkScript, witness)
	}

	// Next, we'll check the sigScript to see if this is a nested p2sh
	// witness program. This is a case wherein the sigScript is actually a
	// datapush of a p2wsh witness program.
	sigPops, err := parsescript.ParseScript(sigScript)
	if err != nil {
		return 0
	}
	if IsPayToScriptHash(pkScript) && parsescript.IsPushOnly(sigPops) &&
		IsWitnessProgram(sigScript[1:]) {
		return getWitnessSigOps(sigScript[1:], witness)
	}

	return 0
}

// getWitnessSigOps returns the number of signature operations generated by
// spending the passed witness program wit the passed witness. The exact
// signature counting heuristic is modified by the version of the passed
// witness program. If the version of the witness program is unable to be
// extracted, then 0 is returned for the sig op count.
func getWitnessSigOps(pkScript []byte, witness wire.TxWitness) int {
	// Attempt to extract the witness program version.
	witnessVersion, witnessProgram, err := ExtractWitnessProgramInfo(
		pkScript,
	)
	if err != nil {
		return 0
	}

	switch witnessVersion {
	case 0:
		switch {
		case len(witnessProgram) == params.PayToWitnessPubKeyHashDataSize:
			return 1
		case len(witnessProgram) == params.PayToWitnessScriptHashDataSize &&
			len(witness) > 0:

			witnessScript := witness[len(witness)-1]
			pops, _ := parsescript.ParseScript(witnessScript)
			return getSigOpCount(pops, true)
		}
	}

	return 0
}

// IsUnspendable returns whether the passed public key script is unspendable, or
// guaranteed to fail at execution.  This allows inputs to be pruned instantly
// when entering the UTXO set.
func IsUnspendable(pkScript []byte) bool {
	pops, err := parsescript.ParseScript(pkScript)
	if err != nil {
		return true
	}

	return len(pops) > 0 && pops[0].Opcode.Value == opcode.OP_RETURN
}

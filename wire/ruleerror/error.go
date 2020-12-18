package ruleerror

import (
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/wire"
)

// Err identifies a rule violation.  It is used to indicate that
// processing of a block or transaction failed due to one of the many validation
// rules.  The caller can use type assertions to determine if a failure was
// specifically due to a rule violation and access the ErrorCode field to
// ascertain the specific reason for the rule violation.
var Err er.ErrorType = er.NewErrorType("ruleerror.Err")

var errorStrings map[*er.ErrorCode]string = make(map[*er.ErrorCode]string)

func mkError(code *er.ErrorCode, str string) *er.ErrorCode {
	errorStrings[code] = str
	return code
}

// These constants are used to identify a specific Err.
var (
	// ErrDuplicateBlock indicates a block with the same hash already
	// exists.
	ErrDuplicateBlock = mkError(Err.Code("ErrDuplicateBlock"),
		"duplicate")

	// ErrBlockTooBig indicates the serialized block size exceeds the
	// maximum allowed size.
	ErrBlockTooBig = mkError(Err.Code("ErrBlockTooBig"),
		"bad-blk-length")

	// ErrBlockWeightTooHigh indicates that the block's computed weight
	// metric exceeds the maximum allowed value.
	ErrBlockWeightTooHigh = mkError(Err.Code("ErrBlockWeightTooHigh"),
		"bad-blk-weight")

	// ErrBlockVersionTooOld indicates the block version is too old and is
	// no longer accepted since the majority of the network has upgraded
	// to a newer version.
	// bitcoind returns "bad-version(0x%08x)", nVersion
	ErrBlockVersionTooOld = mkError(Err.Code("ErrBlockVersionTooOld"),
		"bad-version")

	// ErrTimeTooOld indicates the time is either before the median time of
	// the last several blocks per the chain consensus rules or prior to the
	// most recent checkpoint.
	ErrTimeTooOld = mkError(Err.Code("ErrTimeTooOld"),
		"time-too-old")

	// ErrTimeTooNew indicates the time is too far in the future as compared
	// the current time.
	ErrTimeTooNew = mkError(Err.Code("ErrTimeTooNew"),
		"time-too-new")

	// ErrDifficultyTooLow indicates the difficulty for the block is lower
	// than the difficulty required by the most recent checkpoint.
	ErrDifficultyTooLow = mkError(Err.Code("ErrDifficultyTooLow"),
		"bad-diffbits")

	// ErrUnexpectedDifficulty indicates specified bits do not align with
	// the expected value either because it doesn't match the calculated
	// valued based on difficulty regarted rules or it is out of the valid
	// range.
	ErrUnexpectedDifficulty = mkError(Err.Code("ErrUnexpectedDifficulty"),
		"bad-diffbits")

	// ErrHighHash indicates the block does not hash to a value which is
	// lower than the required target difficultly.
	ErrHighHash = mkError(Err.Code("ErrHighHash"),
		"high-hash")

	// ErrBadMerkleRoot indicates the calculated merkle root does not match
	// the expected value.
	ErrBadMerkleRoot = mkError(Err.Code("ErrBadMerkleRoot"),
		"bad-txnmrklroot")

	// ErrBadCheckpoint indicates a block that is expected to be at a
	// checkpoint height does not match the expected one.
	// This is more strict than bitcoind
	ErrBadCheckpoint = mkError(Err.Code("ErrBadCheckpoint"),
		"bad-fork-prior-to-checkpoint")

	// ErrForkTooOld indicates a block is attempting to fork the block chain
	// before the most recent checkpoint.
	ErrForkTooOld = mkError(Err.Code("ErrForkTooOld"),
		"bad-fork-prior-to-checkpoint")

	// ErrCheckpointTimeTooOld indicates a block has a timestamp before the
	// most recent checkpoint.
	// This is more strict than bitcoind
	ErrCheckpointTimeTooOld = mkError(Err.Code("ErrCheckpointTimeTooOld"),
		"bad-fork-prior-to-checkpoint")

	// ErrNoTransactions indicates the block does not have a least one
	// transaction.  A valid block must have at least the coinbase
	// transaction.
	ErrNoTransactions = mkError(Err.Code("ErrNoTransactions"),
		"bad-cb-missing")

	// ErrNoTxInputs indicates a transaction does not have any inputs.  A
	// valid transaction must have at least one input.
	ErrNoTxInputs = mkError(Err.Code("ErrNoTxInputs"),
		"bad-txns-vin-empty")

	// ErrNoTxOutputs indicates a transaction does not have any outputs.  A
	// valid transaction must have at least one output.
	ErrNoTxOutputs = mkError(Err.Code("ErrNoTxOutputs"),
		"bad-txns-vout-empty")

	// ErrTxTooBig indicates a transaction exceeds the maximum allowed size
	// when serialized.
	ErrTxTooBig = mkError(Err.Code("ErrTxTooBig"),
		"bad-txns-oversize")

	// ErrNegativeTxOutValue indicates an output value for a transaction is
	// negative.
	ErrNegativeTxOutValue = mkError(Err.Code("ErrNegativeTxOutValue"),
		"bad-txns-vout-negative")

	// ErrNegativeTxOutValue indicates an output value for a transaction is
	// negative.
	ErrOversizeTxOutValue = mkError(Err.Code("ErrOversizeTxOutValue"),
		"bad-txns-vout-toolarge")

	// ErrNegativeTxOutValue indicates an output value for a transaction is
	// negative.
	ErrOversizeTxOutSum = mkError(Err.Code("ErrOversizeTxOutSum"),
		"bad-txns-txouttotal-toolarge")

	// ErrDuplicateTxInputs indicates a transaction references the same
	// input more than once.
	ErrDuplicateTxInputs = mkError(Err.Code("ErrDuplicateTxInputs"),
		"bad-txns-inputs-duplicate")

	// ErrNullPrevOut indicates a transaction input is invalid in some way
	// such as referencing a previous transaction outpoint which is out of
	// range or not referencing one at all.
	ErrNullPrevOut = mkError(Err.CodeWithDetail("ErrNullPrevOut",
		"transaction input refers to previous output that is null"),
		"bad-txns-prevout-null")

	// ErrMissingTxOut indicates a transaction output referenced by an input
	// either does not exist or has already been spent. It is misleadingly
	// called MissingTxOut because the previous output is what is missing.
	ErrMissingTxOut = mkError(Err.Code("ErrMissingTxOut"),
		"bad-txns-inputs-missingorspent")

	// ErrUnfinalizedTx indicates a transaction has not been finalized.
	// A valid block may only contain finalized transactions.
	ErrUnfinalizedTx = mkError(Err.Code("ErrUnfinalizedTx"),
		"bad-txns-nonfinal")

	// ErrDuplicateTx indicates a block contains an identical transaction
	// (or at least two transactions which hash to the same value).  A
	// valid block may only contain unique transactions.
	ErrDuplicateTx = mkError(Err.Code("ErrDuplicateTx"),
		"bad-txns-duplicate")

	// ErrOverwriteTx indicates a block contains a transaction that has
	// the same hash as a previous transaction which has not been fully
	// spent.
	ErrOverwriteTx = mkError(Err.Code("ErrOverwriteTx"),
		"bad-txns-BIP30")

	// ErrImmatureSpend indicates a transaction is attempting to spend a
	// coinbase that has not yet reached the required maturity.
	ErrImmatureSpend = mkError(Err.Code("ErrImmatureSpend"),
		"bad-txns-premature-spend-of-coinbase")

	// ErrSpendTooHigh indicates a transaction is attempting to spend more
	// value than the sum of all of its inputs.
	ErrSpendTooHigh = mkError(Err.Code("ErrSpendTooHigh"),
		"bad-txns-in-belowout")

	// ErrBadFees indicates the total fees for a block are invalid due to
	// exceeding the maximum possible value.
	ErrBadFees = mkError(Err.Code("ErrBadFees"),
		"bad-txns-accumulated-fee-outofrange")

	// ErrTooManySigOps indicates the total number of signature operations
	// for a transaction or block exceed the maximum allowed limits.
	// NOTE: We're returning this even for checking of a free transaction.
	ErrTooManySigOps = mkError(Err.Code("ErrTooManySigOps"),
		"bad-blk-sigops")

	// ErrFirstTxNotCoinbase indicates the first transaction in a block
	// is not a coinbase transaction.
	ErrFirstTxNotCoinbase = mkError(Err.Code("ErrFirstTxNotCoinbase"),
		"bad-cb-missing")

	// ErrMultipleCoinbases indicates a block contains more than one
	// coinbase transaction.
	ErrMultipleCoinbases = mkError(Err.Code("ErrMultipleCoinbases"),
		"bad-cb-multiple")

	// ErrBadCoinbaseScriptLen indicates the length of the signature script
	// for a coinbase transaction is not within the valid range.
	ErrBadCoinbaseScriptLen = mkError(Err.Code("ErrBadCoinbaseScriptLen"),
		"bad-cb-length")

	// ErrBadCoinbaseValue indicates the amount of a coinbase value does
	// not match the expected value of the subsidy plus the sum of all fees.
	ErrBadCoinbaseValue = mkError(Err.Code("ErrBadCoinbaseValue"),
		"bad-cb-amount")

	// ErrMissingCoinbaseHeight indicates the coinbase transaction for a
	// block does not start with the serialized block block height as
	// required for version 2 and higher blocks.
	ErrMissingCoinbaseHeight = mkError(Err.Code("ErrMissingCoinbaseHeight"),
		"bad-cb-height")

	// ErrBadCoinbaseHeight indicates the serialized block height in the
	// coinbase transaction for version 2 and higher blocks does not match
	// the expected value.
	ErrBadCoinbaseHeight = mkError(Err.Code("ErrBadCoinbaseHeight"),
		"bad-cb-height")

	// ErrBadCoinbaseTax indicates that the tax was paid to the wrong
	// network steward key.
	ErrBadCoinbaseNetworkSteward = mkError(Err.Code("ErrBadCoinbaseNetworkSteward"),
		"ErrBadCoinbaseNetworkSteward")

	// ErrScriptMalformed indicates a transaction script is malformed in
	// some way.  For example, it might be longer than the maximum allowed
	// length or fail to parse.
	ErrScriptMalformed = mkError(Err.Code("ErrScriptMalformed"),
		"mandatory-script-verify-flag-failed (ErrScriptMalformed)")

	// ErrScriptValidation indicates the result of executing transaction
	// script failed.  The error covers any failure when executing scripts
	// such signature verification failures and execution past the end of
	// the stack.
	ErrScriptValidation = mkError(Err.Code("ErrScriptValidation"),
		"mandatory-script-verify-flag-failed (ErrScriptValidation)")

	// ErrUnexpectedWitness indicates that a block includes transactions
	// with witness data, but doesn't also have a witness commitment within
	// the coinbase transaction.
	ErrUnexpectedWitness = mkError(Err.Code("ErrUnexpectedWitness"),
		"unexpected-witness")

	// ErrInvalidWitnessCommitment indicates that a block's witness
	// commitment is not well formed.
	ErrInvalidWitnessCommitment = mkError(Err.Code("ErrInvalidWitnessCommitment"),
		"bad-witness-nonce-size")

	// ErrWitnessCommitmentMismatch indicates that the witness commitment
	// included in the block's coinbase transaction doesn't match the
	// manually computed witness commitment.
	ErrWitnessCommitmentMismatch = mkError(Err.Code("ErrWitnessCommitmentMismatch"),
		"bad-witness-merkle-match")

	// ErrPreviousBlockUnknown indicates that the previous block is not known.
	ErrPreviousBlockUnknown = mkError(Err.Code("ErrPreviousBlockUnknown"),
		"prev-blk-not-found")

	// ErrInvalidAncestorBlock indicates that an ancestor of this block has
	// already failed validation.
	ErrInvalidAncestorBlock = mkError(Err.Code("ErrInvalidAncestorBlock"),
		"bad-prevblk")
)

// The following errors are not aligned with bitcoind, they are unique to pktd
var (
	// ErrPrevBlockNotBest indicates that the block's previous block is not the
	// current chain tip. This is not a block validation rule, but is required
	// for block proposals submitted via getblocktemplate RPC.
	// This error message is not aligned with bitcoind
	ErrPrevBlockNotBest = mkError(Err.Code("ErrPrevBlockNotBest"),
		"ErrPrevBlockNotBest")

	// ErrBadPow indicates that the proof of work was somehow malformed
	// This error message is not aligned with bitcoind
	ErrBadPow = mkError(Err.Code("ErrBadPow"),
		"ErrBadPow")

	// ErrPowCannotVerify indicates that the pow cannot be verified because there
	// is a missing block header required to check one of the announcements.
	// This error message is not aligned with bitcoind
	ErrPowCannotVerify = mkError(Err.Code("ErrPowCannotVerify"),
		"ErrPowCannotVerify")

	// ErrNetworkStewardOldSpend indicates that a transaction tried to spend
	// coins in the network steward wallet which are older than the age limit
	// for network steward payouts.
	// This error message is not aligned with bitcoind
	ErrNetworkStewardOldSpend = mkError(Err.Code("ErrNetworkStewardOldSpend"),
		"ErrNetworkStewardOldSpend")
)

var (
	// ErrOrphanTransactionDisallowed orphan transaction but allowOrphan is not set
	ErrOrphanTransactionDisallowed = Err.CodeWithDetail("ErrOrphanTransactionDisallowed",
		"orphan transaction but allowOrphan is not set")

	// ErrOrphanTransactionTooBig orphan transaction is larger than MaxOrphanTxSize
	ErrOrphanTransactionTooBig = Err.CodeWithDetail("ErrOrphanTransactionTooBig",
		"orphan transaction is larger than MaxOrphanTxSize")

	// ErrTxExistsInMempool transaction already exists in mempool
	ErrTxExistsInMempool = mkError(Err.CodeWithDetail("ErrTxExistsInMempool",
		"already have transaction (in mempool)"),
		"txn-already-in-mempool")

	// ErrTxExistsInChain transaction already exists in chain (confirmed)
	ErrTxExistsInChain = mkError(Err.CodeWithDetail("ErrTxExistsInChain",
		"transaction already exists (in chain)"),
		"txn-already-known")
)

// These errors map directly to the rejection codes and are used by the
// mempool when rejecting invalid transactions
var (
	ErrRejectMalformed = Err.CodeWithNumber("ErrRejectMalformed",
		int(wire.RejectMalformed))
	ErrRejectInvalid = Err.CodeWithNumber("ErrRejectInvalid",
		int(wire.RejectInvalid))
	ErrRejectObsolete = Err.CodeWithNumber("ErrRejectObsolete",
		int(wire.RejectObsolete))
	ErrRejectDuplicate = Err.CodeWithNumber("ErrRejectDuplicate",
		int(wire.RejectDuplicate))
	ErrRejectNonstandard = Err.CodeWithNumber("ErrRejectNonstandard",
		int(wire.RejectNonstandard))
	ErrRejectDust = Err.CodeWithNumber("ErrRejectDust",
		int(wire.RejectDust))
	ErrRejectInsufficientFee = Err.CodeWithNumber("ErrRejectInsufficientFee",
		int(wire.RejectInsufficientFee))
	ErrRejectCheckpoint = Err.CodeWithNumber("ErrRejectCheckpoint",
		int(wire.RejectCheckpoint))
)

var ErrUnknown = Err.CodeWithDetail("ErrUnknown", "Rejected with unexpected code")

func IsTxRuleErrorCode(ec *er.ErrorCode) bool {
	switch ec {
	case ErrRejectMalformed:
		return true
	case ErrRejectInvalid:
		return true
	case ErrRejectObsolete:
		return true
	case ErrRejectDuplicate:
		return true
	case ErrRejectNonstandard:
		return true
	case ErrRejectDust:
		return true
	case ErrRejectInsufficientFee:
		return true
	case ErrRejectCheckpoint:
		return true
	}
	return false
}

// ExtractRejectCode attempts to return a relevant reject code for a given error
// by examining the error for known types.  It will return true if a code
// was successfully extracted.
func ExtractRejectCode(e er.R) (rejCode wire.RejectCode, errString string, found bool) {
	errCode := Err.Decode(e)
	errString = errorStrings[errCode]
	if errString == "" && e != nil {
		s := e.String()
		l := len(s)
		if l > 32 {
			l = 32
		}
		errString = s[0:l]
	}
	found = true

	switch errCode {

	case ErrOrphanTransactionDisallowed:
		fallthrough
	case ErrDuplicateBlock:
		rejCode = wire.RejectDuplicate
		return

	case ErrBlockVersionTooOld:
		rejCode = wire.RejectObsolete
		return

	case ErrTxExistsInMempool:
		fallthrough
	case ErrTxExistsInChain:
		rejCode = wire.RejectDuplicate
		return

	case ErrCheckpointTimeTooOld:
		fallthrough
	case ErrDifficultyTooLow:
		fallthrough
	case ErrBadCheckpoint:
		fallthrough
	case ErrForkTooOld:
		rejCode = wire.RejectCheckpoint
		return

	case ErrOrphanTransactionTooBig:
		rejCode = wire.RejectNonstandard
		return

	case ErrUnknown:
		fallthrough
	case nil:
		// Not one of our errors
		found = false
		rejCode = wire.RejectInvalid
		return

	default:
		if IsTxRuleErrorCode(errCode) {
			rejCode = wire.RejectCode(errCode.Number)
		} else {
			// One of the other errors above
			rejCode = wire.RejectInvalid
		}
		return
	}
}

// ErrToRejectErr examines the underlying type of the error and returns a reject
// code and string appropriate to be sent in a wire.MsgReject message.
func ErrToRejectErr(err er.R) (wire.RejectCode, string) {
	// Return a generic rejected string if there is no error.  This really
	// should not happen unless the code elsewhere is not setting an error
	// as it should be, but it's best to be safe and simply return a generic
	// string rather than allowing the following code that dereferences the
	// err to panic.
	if err == nil {
		return wire.RejectInvalid, "rejected"
	}

	// Return the reject code along with the error text if it can be
	// extracted from the error.
	rejectCode, str, found := ExtractRejectCode(err)
	if found {
		return rejectCode, str
	}

	// When the underlying error is not one of the above cases, just return
	// wire.RejectInvalid with a generic rejected string plus the error
	// text.
	return wire.RejectInvalid, "rejected: " + str
}

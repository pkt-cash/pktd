package pushtx

import (
	"fmt"
	"strings"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/wire"
)

// BroadcastErrorCode uniquely identifies the broadcast error.
// type BroadcastErrorCode uint8

var Err er.ErrorType = er.NewErrorType("pushtx.Err")

var (
	// RejUnknown is the code used when a transaction has been rejected by some
	// unknown reason by a peer.
	RejUnknown = Err.Code("RejUnknown")

	// RejInvalid is the code used when a transaction has been deemed invalid
	// by a peer.
	RejInvalid = Err.Code("RejInvalid")

	// RejInsufficientFee is the code used when a transaction has been deemed
	// as having an insufficient fee by a peer.
	RejInsufficientFee = Err.Code("RejInsufficientFee")

	// RejMempool is the code used when a transaction already exists in a
	// peer's mempool.
	RejMempool = Err.Code("RejMempool")

	// RejConfirmed is the code used when a transaction has been deemed as
	// already existing in the chain by a peer.
	RejConfirmed = Err.Code("RejConfirmed")
)

// ParseBroadcastError maps a peer's reject message for a transaction to a
// BroadcastError.
// TODO(cjd): We're parsing reject messages here which is a bad idea.
//            The text is not guaranteed to be relevant and bitcoind has
//            already gotten rid of the reject message because nodes cannot
//            be trusted to send a reject honestly. Furthermore we are parsing
//            text of the messages because the error codes are not clear enough
//            alone.
func ParseBroadcastError(msg *wire.MsgReject, peerAddr string) er.R {
	// We'll determine the appropriate broadcast error code by looking at
	// the reject's message code and reason. The only reject codes returned
	// from peers (bitcoind and btcd) when attempting to accept a
	// transaction into their mempool are:
	//   RejectInvalid, RejectNonstandard, RejectInsufficientFee,
	//   RejectDuplicate
	var code *er.ErrorCode
	switch {
	case msg.Code == wire.RejectDust:
		return nil

	// The cases below apply for reject messages sent from any kind of peer.
	case msg.Code == wire.RejectInvalid || msg.Code == wire.RejectNonstandard:
		code = RejInvalid

	case msg.Code == wire.RejectInsufficientFee:
		code = RejInsufficientFee

	// The cases below apply for reject messages sent from bitcoind peers.
	//
	// If the transaction double spends an unconfirmed transaction in the
	// peer's mempool, then we'll deem it as invalid.
	case msg.Code == wire.RejectDuplicate &&
		strings.Contains(msg.Reason, "txn-mempool-conflict"):
		code = RejInvalid

	// If the transaction was rejected due to it already existing in the
	// peer's mempool, then return an error signaling so.
	case msg.Code == wire.RejectDuplicate &&
		strings.Contains(msg.Reason, "already have transaction"):
		fallthrough
	case msg.Code == wire.RejectDuplicate &&
		strings.Contains(msg.Reason, "txn-already-in-mempool"):
		code = RejMempool

	// If the transaction was rejected due to it already existing in the
	// chain according to our peer, then we'll return an error signaling so.
	case msg.Code == wire.RejectDuplicate &&
		strings.Contains(msg.Reason, "txn-already-known"):
		code = RejConfirmed

	// The cases below apply for reject messages sent from btcd peers.
	//
	// If the transaction double spends an unconfirmed transaction in the
	// peer's mempool, then we'll deem it as invalid.
	case msg.Code == wire.RejectDuplicate &&
		strings.Contains(msg.Reason, "already spent"):
		code = RejInvalid

	// If the transaction was rejected due to it already existing in the
	// chain according to our peer, then we'll return an error signaling so.
	case msg.Code == wire.RejectDuplicate &&
		strings.Contains(msg.Reason, "transaction already exists"):
		code = RejConfirmed

	// Any other reject messages will use the unknown code.
	default:
		code = RejUnknown
	}

	reason := fmt.Sprintf("rejected by %v: %v", peerAddr, msg.Reason)
	return code.New(reason, nil)
}

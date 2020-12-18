package pushtx_test

import (
	"testing"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/neutrino/pushtx"
	"github.com/pkt-cash/pktd/wire"
)

// TestParseBroadcastErrorCode ensures that we properly construct a
// BroadcastError with the appropriate error code from a wire.MsgReject.
func TestParseBroadcastErrorCode(t *testing.T) {
	testCases := []struct {
		name string
		msg  *wire.MsgReject
		code er.R
	}{
		{
			name: "dust transaction",
			msg: &wire.MsgReject{
				Code: wire.RejectDust,
			},
		},
		{
			name: "invalid transaction",
			msg: &wire.MsgReject{
				Code:   wire.RejectInvalid,
				Reason: "spends inexistent output",
			},
			code: pushtx.RejInvalid.Default(),
		},
		{
			name: "nonstandard transaction",
			msg: &wire.MsgReject{
				Code:   wire.RejectNonstandard,
				Reason: "",
			},
			code: pushtx.RejInvalid.Default(),
		},
		{
			name: "insufficient fee transaction",
			msg: &wire.MsgReject{
				Code:   wire.RejectInsufficientFee,
				Reason: "",
			},
			code: pushtx.RejInsufficientFee.Default(),
		},
		{
			name: "bitcoind mempool double spend",
			msg: &wire.MsgReject{
				Code:   wire.RejectDuplicate,
				Reason: "txn-mempool-conflict",
			},
			code: pushtx.RejInvalid.Default(),
		},
		{
			name: "bitcoind transaction in mempool",
			msg: &wire.MsgReject{
				Code:   wire.RejectDuplicate,
				Reason: "txn-already-in-mempool",
			},
			code: pushtx.RejMempool.Default(),
		},
		{
			name: "bitcoind transaction in chain",
			msg: &wire.MsgReject{
				Code:   wire.RejectDuplicate,
				Reason: "txn-already-known",
			},
			code: pushtx.RejConfirmed.Default(),
		},
		{
			name: "btcd mempool double spend",
			msg: &wire.MsgReject{
				Code:   wire.RejectDuplicate,
				Reason: "already spent",
			},
			code: pushtx.RejInvalid.Default(),
		},
		{
			name: "btcd transaction in mempool",
			msg: &wire.MsgReject{
				Code:   wire.RejectDuplicate,
				Reason: "already have transaction",
			},
			code: pushtx.RejMempool.Default(),
		},
		{
			name: "btcd transaction in chain",
			msg: &wire.MsgReject{
				Code:   wire.RejectDuplicate,
				Reason: "transaction already exists",
			},
			code: pushtx.RejConfirmed.Default(),
		},
	}

	for _, testCase := range testCases {
		test := testCase
		t.Run(test.name, func(t *testing.T) {
			broadcastErr := pushtx.ParseBroadcastError(
				test.msg, "127.0.0.1:8333",
			)
			if !er.Equals(broadcastErr, test.code) {
				t.Fatalf("expected BroadcastErrorCode %v, got "+
					"%v", test.code, broadcastErr)
			}
		})
	}
}

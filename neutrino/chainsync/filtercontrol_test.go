package chainsync

import (
	"testing"

	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/wire"
	"github.com/pkt-cash/pktd/wire/protocol"
	"github.com/pkt-cash/pktd/wire/ruleerror"
)

func TestControlCFHeader(t *testing.T) {
	// We'll modify our backing list of checkpoints for this test.
	height := uint32(999)
	header := hashFromStr(
		"4a242283a406a7c089f671bb8df7671e5d5e9ba577cea1047d30a7f4919df193",
	)
	filterHeaderCheckpoints = map[protocol.BitcoinNet]map[uint32]*chainhash.Hash{
		chaincfg.MainNetParams.Net: {
			height: header,
		},
	}

	// Expect the control at height to succeed.
	err := ControlCFHeader(
		chaincfg.MainNetParams, wire.GCSFilterRegular, height, header,
	)
	if err != nil {
		t.Fatalf("error checking height: %v", err)
	}

	// Pass an invalid header, this should return an error.
	header = hashFromStr(
		"000000000006a7c089f671bb8df7671e5d5e9ba577cea1047d30a7f4919df193",
	)
	err = ControlCFHeader(
		chaincfg.MainNetParams, wire.GCSFilterRegular, height, header,
	)
	if !ruleerror.ErrBadCheckpoint.Is(err) {
		t.Fatalf("expected ruleerror.ErrBadCheckpoint, got %v", err)
	}

	// Finally, control an unknown height. This should also pass since we
	// don't have the checkpoint stored.
	err = ControlCFHeader(
		chaincfg.MainNetParams, wire.GCSFilterRegular, 99, header,
	)
	if err != nil {
		t.Fatalf("error checking height: %v", err)
	}
}

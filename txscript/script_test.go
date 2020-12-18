// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/txscript/opcode"
	"github.com/pkt-cash/pktd/txscript/params"
	"github.com/pkt-cash/pktd/txscript/parsescript"
	"github.com/pkt-cash/pktd/txscript/scriptbuilder"
	"github.com/pkt-cash/pktd/txscript/txscripterr"

	"github.com/pkt-cash/pktd/wire"
)

// TestParseOpcode tests for opcode parsing with bad data templates.
func TestParseOpcode(t *testing.T) {
	// Deep copy the array and make one of the opcodes invalid by setting it
	// to the wrong length.
	fakeArray := make(map[byte]opcode.Opcode)
	fakeArray[opcode.OP_PUSHDATA4] = opcode.Opcode{Value: opcode.OP_PUSHDATA4, Length: -8}

	// This script would be fine if -8 was a valid length.
	_, err := parsescript.ParseScriptTemplate([]byte{
		opcode.OP_PUSHDATA4, 0x1, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
	}, fakeArray)
	if err == nil {
		t.Errorf("no error with dodgy opcode array!")
	}
}

// TestUnparsingInvalidOpcodes tests for errors when unparsing invalid parsed
// opcodes.
func TestUnparsingInvalidOpcodes(t *testing.T) {
	tests := []struct {
		name        string
		pop         *parsescript.ParsedOpcode
		expectedErr er.R
	}{
		{
			name: "OP_FALSE",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_FALSE),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_FALSE long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_FALSE),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_1 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_1),
				Data:   nil,
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_1",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_1),
				Data:   make([]byte, 1),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_1 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_1),
				Data:   make([]byte, 2),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_2 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_2),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_2",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_2),
				Data:   make([]byte, 2),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_2 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_2),
				Data:   make([]byte, 3),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_3 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_3),
				Data:   make([]byte, 2),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_3",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_3),
				Data:   make([]byte, 3),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_3 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_3),
				Data:   make([]byte, 4),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_4 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_4),
				Data:   make([]byte, 3),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_4",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_4),
				Data:   make([]byte, 4),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_4 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_4),
				Data:   make([]byte, 5),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_5 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_5),
				Data:   make([]byte, 4),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_5",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_5),
				Data:   make([]byte, 5),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_5 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_5),
				Data:   make([]byte, 6),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_6 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_6),
				Data:   make([]byte, 5),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_6",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_6),
				Data:   make([]byte, 6),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_6 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_6),
				Data:   make([]byte, 7),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_7 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_7),
				Data:   make([]byte, 6),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_7",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_7),
				Data:   make([]byte, 7),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_7 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_7),
				Data:   make([]byte, 8),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_8 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_8),
				Data:   make([]byte, 7),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_8",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_8),
				Data:   make([]byte, 8),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_8 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_8),
				Data:   make([]byte, 9),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_9 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_9),
				Data:   make([]byte, 8),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_9",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_9),
				Data:   make([]byte, 9),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_9 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_9),
				Data:   make([]byte, 10),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_10 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_10),
				Data:   make([]byte, 9),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_10",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_10),
				Data:   make([]byte, 10),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_10 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_10),
				Data:   make([]byte, 11),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_11 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_11),
				Data:   make([]byte, 10),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_11",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_11),
				Data:   make([]byte, 11),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_11 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_11),
				Data:   make([]byte, 12),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_12 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_12),
				Data:   make([]byte, 11),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_12",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_12),
				Data:   make([]byte, 12),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_12 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_12),
				Data:   make([]byte, 13),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_13 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_13),
				Data:   make([]byte, 12),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_13",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_13),
				Data:   make([]byte, 13),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_13 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_13),
				Data:   make([]byte, 14),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_14 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_14),
				Data:   make([]byte, 13),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_14",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_14),
				Data:   make([]byte, 14),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_14 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_14),
				Data:   make([]byte, 15),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_15 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_15),
				Data:   make([]byte, 14),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_15",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_15),
				Data:   make([]byte, 15),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_15 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_15),
				Data:   make([]byte, 16),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_16 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_16),
				Data:   make([]byte, 15),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_16",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_16),
				Data:   make([]byte, 16),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_16 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_16),
				Data:   make([]byte, 17),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_17 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_17),
				Data:   make([]byte, 16),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_17",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_17),
				Data:   make([]byte, 17),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_17 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_17),
				Data:   make([]byte, 18),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_18 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_18),
				Data:   make([]byte, 17),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_18",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_18),
				Data:   make([]byte, 18),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_18 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_18),
				Data:   make([]byte, 19),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_19 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_19),
				Data:   make([]byte, 18),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_19",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_19),
				Data:   make([]byte, 19),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_19 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_19),
				Data:   make([]byte, 20),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_20 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_20),
				Data:   make([]byte, 19),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_20",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_20),
				Data:   make([]byte, 20),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_20 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_20),
				Data:   make([]byte, 21),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_21 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_21),
				Data:   make([]byte, 20),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_21",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_21),
				Data:   make([]byte, 21),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_21 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_21),
				Data:   make([]byte, 22),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_22 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_22),
				Data:   make([]byte, 21),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_22",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_22),
				Data:   make([]byte, 22),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_22 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_22),
				Data:   make([]byte, 23),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_23 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_23),
				Data:   make([]byte, 22),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_23",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_23),
				Data:   make([]byte, 23),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_23 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_23),
				Data:   make([]byte, 24),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_24 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_24),
				Data:   make([]byte, 23),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_24",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_24),
				Data:   make([]byte, 24),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_24 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_24),
				Data:   make([]byte, 25),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_25 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_25),
				Data:   make([]byte, 24),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_25",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_25),
				Data:   make([]byte, 25),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_25 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_25),
				Data:   make([]byte, 26),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_26 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_26),
				Data:   make([]byte, 25),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_26",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_26),
				Data:   make([]byte, 26),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_26 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_26),
				Data:   make([]byte, 27),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_27 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_27),
				Data:   make([]byte, 26),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_27",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_27),
				Data:   make([]byte, 27),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_27 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_27),
				Data:   make([]byte, 28),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_28 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_28),
				Data:   make([]byte, 27),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_28",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_28),
				Data:   make([]byte, 28),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_28 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_28),
				Data:   make([]byte, 29),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_29 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_29),
				Data:   make([]byte, 28),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_29",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_29),
				Data:   make([]byte, 29),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_29 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_29),
				Data:   make([]byte, 30),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_30 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_30),
				Data:   make([]byte, 29),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_30",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_30),
				Data:   make([]byte, 30),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_30 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_30),
				Data:   make([]byte, 31),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_31 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_31),
				Data:   make([]byte, 30),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_31",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_31),
				Data:   make([]byte, 31),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_31 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_31),
				Data:   make([]byte, 32),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_32 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_32),
				Data:   make([]byte, 31),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_32",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_32),
				Data:   make([]byte, 32),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_32 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_32),
				Data:   make([]byte, 33),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_33 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_33),
				Data:   make([]byte, 32),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_33",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_33),
				Data:   make([]byte, 33),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_33 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_33),
				Data:   make([]byte, 34),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_34 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_34),
				Data:   make([]byte, 33),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_34",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_34),
				Data:   make([]byte, 34),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_34 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_34),
				Data:   make([]byte, 35),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_35 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_35),
				Data:   make([]byte, 34),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_35",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_35),
				Data:   make([]byte, 35),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_35 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_35),
				Data:   make([]byte, 36),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_36 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_36),
				Data:   make([]byte, 35),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_36",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_36),
				Data:   make([]byte, 36),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_36 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_36),
				Data:   make([]byte, 37),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_37 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_37),
				Data:   make([]byte, 36),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_37",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_37),
				Data:   make([]byte, 37),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_37 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_37),
				Data:   make([]byte, 38),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_38 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_38),
				Data:   make([]byte, 37),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_38",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_38),
				Data:   make([]byte, 38),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_38 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_38),
				Data:   make([]byte, 39),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_39 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_39),
				Data:   make([]byte, 38),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_39",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_39),
				Data:   make([]byte, 39),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_39 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_39),
				Data:   make([]byte, 40),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_40 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_40),
				Data:   make([]byte, 39),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_40",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_40),
				Data:   make([]byte, 40),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_40 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_40),
				Data:   make([]byte, 41),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_41 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_41),
				Data:   make([]byte, 40),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_41",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_41),
				Data:   make([]byte, 41),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_41 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_41),
				Data:   make([]byte, 42),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_42 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_42),
				Data:   make([]byte, 41),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_42",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_42),
				Data:   make([]byte, 42),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_42 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_42),
				Data:   make([]byte, 43),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_43 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_43),
				Data:   make([]byte, 42),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_43",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_43),
				Data:   make([]byte, 43),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_43 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_43),
				Data:   make([]byte, 44),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_44 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_44),
				Data:   make([]byte, 43),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_44",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_44),
				Data:   make([]byte, 44),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_44 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_44),
				Data:   make([]byte, 45),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_45 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_45),
				Data:   make([]byte, 44),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_45",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_45),
				Data:   make([]byte, 45),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_45 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_45),
				Data:   make([]byte, 46),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_46 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_46),
				Data:   make([]byte, 45),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_46",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_46),
				Data:   make([]byte, 46),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_46 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_46),
				Data:   make([]byte, 47),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_47 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_47),
				Data:   make([]byte, 46),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_47",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_47),
				Data:   make([]byte, 47),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_47 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_47),
				Data:   make([]byte, 48),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_48 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_48),
				Data:   make([]byte, 47),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_48",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_48),
				Data:   make([]byte, 48),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_48 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_48),
				Data:   make([]byte, 49),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_49 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_49),
				Data:   make([]byte, 48),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_49",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_49),
				Data:   make([]byte, 49),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_49 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_49),
				Data:   make([]byte, 50),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_50 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_50),
				Data:   make([]byte, 49),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_50",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_50),
				Data:   make([]byte, 50),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_50 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_50),
				Data:   make([]byte, 51),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_51 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_51),
				Data:   make([]byte, 50),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_51",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_51),
				Data:   make([]byte, 51),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_51 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_51),
				Data:   make([]byte, 52),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_52 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_52),
				Data:   make([]byte, 51),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_52",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_52),
				Data:   make([]byte, 52),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_52 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_52),
				Data:   make([]byte, 53),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_53 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_53),
				Data:   make([]byte, 52),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_53",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_53),
				Data:   make([]byte, 53),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_53 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_53),
				Data:   make([]byte, 54),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_54 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_54),
				Data:   make([]byte, 53),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_54",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_54),
				Data:   make([]byte, 54),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_54 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_54),
				Data:   make([]byte, 55),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_55 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_55),
				Data:   make([]byte, 54),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_55",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_55),
				Data:   make([]byte, 55),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_55 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_55),
				Data:   make([]byte, 56),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_56 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_56),
				Data:   make([]byte, 55),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_56",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_56),
				Data:   make([]byte, 56),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_56 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_56),
				Data:   make([]byte, 57),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_57 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_57),
				Data:   make([]byte, 56),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_57",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_57),
				Data:   make([]byte, 57),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_57 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_57),
				Data:   make([]byte, 58),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_58 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_58),
				Data:   make([]byte, 57),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_58",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_58),
				Data:   make([]byte, 58),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_58 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_58),
				Data:   make([]byte, 59),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_59 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_59),
				Data:   make([]byte, 58),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_59",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_59),
				Data:   make([]byte, 59),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_59 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_59),
				Data:   make([]byte, 60),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_60 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_60),
				Data:   make([]byte, 59),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_60",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_60),
				Data:   make([]byte, 60),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_60 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_60),
				Data:   make([]byte, 61),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_61 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_61),
				Data:   make([]byte, 60),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_61",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_61),
				Data:   make([]byte, 61),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_61 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_61),
				Data:   make([]byte, 62),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_62 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_62),
				Data:   make([]byte, 61),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_62",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_62),
				Data:   make([]byte, 62),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_62 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_62),
				Data:   make([]byte, 63),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_63 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_63),
				Data:   make([]byte, 62),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_63",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_63),
				Data:   make([]byte, 63),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_63 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_63),
				Data:   make([]byte, 64),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_64 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_64),
				Data:   make([]byte, 63),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_64",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_64),
				Data:   make([]byte, 64),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_64 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_64),
				Data:   make([]byte, 65),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_65 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_65),
				Data:   make([]byte, 64),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_65",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_65),
				Data:   make([]byte, 65),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_65 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_65),
				Data:   make([]byte, 66),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_66 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_66),
				Data:   make([]byte, 65),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_66",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_66),
				Data:   make([]byte, 66),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_66 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_66),
				Data:   make([]byte, 67),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_67 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_67),
				Data:   make([]byte, 66),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_67",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_67),
				Data:   make([]byte, 67),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_67 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_67),
				Data:   make([]byte, 68),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_68 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_68),
				Data:   make([]byte, 67),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_68",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_68),
				Data:   make([]byte, 68),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_68 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_68),
				Data:   make([]byte, 69),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_69 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_69),
				Data:   make([]byte, 68),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_69",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_69),
				Data:   make([]byte, 69),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_69 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_69),
				Data:   make([]byte, 70),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_70 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_70),
				Data:   make([]byte, 69),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_70",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_70),
				Data:   make([]byte, 70),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_70 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_70),
				Data:   make([]byte, 71),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_71 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_71),
				Data:   make([]byte, 70),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_71",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_71),
				Data:   make([]byte, 71),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_71 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_71),
				Data:   make([]byte, 72),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_72 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_72),
				Data:   make([]byte, 71),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_72",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_72),
				Data:   make([]byte, 72),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_72 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_72),
				Data:   make([]byte, 73),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_73 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_73),
				Data:   make([]byte, 72),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_73",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_73),
				Data:   make([]byte, 73),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_73 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_73),
				Data:   make([]byte, 74),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_74 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_74),
				Data:   make([]byte, 73),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_74",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_74),
				Data:   make([]byte, 74),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_74 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_74),
				Data:   make([]byte, 75),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_75 short",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_75),
				Data:   make([]byte, 74),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DATA_75",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_75),
				Data:   make([]byte, 75),
			},
			expectedErr: nil,
		},
		{
			name: "OP_DATA_75 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DATA_75),
				Data:   make([]byte, 76),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_PUSHDATA1",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_PUSHDATA1),
				Data:   []byte{0, 1, 2, 3, 4},
			},
			expectedErr: nil,
		},
		{
			name: "OP_PUSHDATA2",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_PUSHDATA2),
				Data:   []byte{0, 1, 2, 3, 4},
			},
			expectedErr: nil,
		},
		{
			name: "OP_PUSHDATA4",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_PUSHDATA1),
				Data:   []byte{0, 1, 2, 3, 4},
			},
			expectedErr: nil,
		},
		{
			name: "OP_1NEGATE",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_1NEGATE),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_1NEGATE long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_1NEGATE),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_RESERVED",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_RESERVED),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_RESERVED long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_RESERVED),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_TRUE",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_TRUE),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_TRUE long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_TRUE),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_2",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_2 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_2",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_2 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_3",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_3),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_3 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_3),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_4",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_4),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_4 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_4),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_5",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_5),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_5 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_5),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_6",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_6),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_6 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_6),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_7",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_7),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_7 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_7),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_8",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_8),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_8 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_8),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_9",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_9),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_9 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_9),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_10",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_10),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_10 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_10),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_11",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_11),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_11 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_11),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_12",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_12),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_12 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_12),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_13",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_13),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_13 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_13),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_14",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_14),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_14 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_14),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_15",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_15),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_15 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_15),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_16",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_16),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_16 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_16),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NOP",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NOP long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_VER",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_VER),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_VER long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_VER),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_IF",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_IF),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_IF long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_IF),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NOTIF",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOTIF),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NOTIF long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOTIF),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_VERIF",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_VERIF),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_VERIF long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_VERIF),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_VOTE",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_VOTE),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_VOTE long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_VOTE),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_ELSE",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_ELSE),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_ELSE long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_ELSE),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_ENDIF",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_ENDIF),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_ENDIF long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_ENDIF),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_VERIFY",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_VERIFY),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_VERIFY long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_VERIFY),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_RETURN",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_RETURN),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_RETURN long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_RETURN),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_TOALTSTACK",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_TOALTSTACK),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_TOALTSTACK long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_TOALTSTACK),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_FROMALTSTACK",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_FROMALTSTACK),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_FROMALTSTACK long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_FROMALTSTACK),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_2DROP",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2DROP),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_2DROP long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2DROP),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_2DUP",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2DUP),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_2DUP long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2DUP),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_3DUP",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_3DUP),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_3DUP long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_3DUP),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_2OVER",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2OVER),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_2OVER long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2OVER),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_2ROT",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2ROT),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_2ROT long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2ROT),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_2SWAP",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2SWAP),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_2SWAP long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2SWAP),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_IFDUP",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_IFDUP),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_IFDUP long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_IFDUP),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DEPTH",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DEPTH),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_DEPTH long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DEPTH),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DROP",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DROP),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_DROP long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DROP),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DUP",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DUP),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_DUP long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DUP),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NIP",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NIP),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NIP long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NIP),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_OVER",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_OVER),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_OVER long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_OVER),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_PICK",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_PICK),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_PICK long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_PICK),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_ROLL",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_ROLL),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_ROLL long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_ROLL),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_ROT",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_ROT),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_ROT long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_ROT),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_SWAP",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_SWAP),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_SWAP long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_SWAP),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_TUCK",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_TUCK),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_TUCK long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_TUCK),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_CAT",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_CAT),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_CAT long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_CAT),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_SUBSTR",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_SUBSTR),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_SUBSTR long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_SUBSTR),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_LEFT",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_LEFT),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_LEFT long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_LEFT),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_LEFT",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_LEFT),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_LEFT long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_LEFT),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_RIGHT",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_RIGHT),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_RIGHT long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_RIGHT),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_SIZE",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_SIZE),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_SIZE long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_SIZE),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_INVERT",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_INVERT),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_INVERT long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_INVERT),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_AND",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_AND),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_AND long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_AND),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_OR",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_OR),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_OR long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_OR),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_XOR",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_XOR),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_XOR long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_XOR),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_EQUAL",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_EQUAL),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_EQUAL long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_EQUAL),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_EQUALVERIFY",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_EQUALVERIFY),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_EQUALVERIFY long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_EQUALVERIFY),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_RESERVED1",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_RESERVED1),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_RESERVED1 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_RESERVED1),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_RESERVED2",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_RESERVED2),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_RESERVED2 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_RESERVED2),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_1ADD",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_1ADD),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_1ADD long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_1ADD),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_1SUB",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_1SUB),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_1SUB long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_1SUB),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_2MUL",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2MUL),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_2MUL long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2MUL),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_2DIV",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2DIV),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_2DIV long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_2DIV),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NEGATE",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NEGATE),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NEGATE long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NEGATE),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_ABS",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_ABS),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_ABS long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_ABS),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NOT",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOT),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NOT long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOT),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_0NOTEQUAL",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_0NOTEQUAL),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_0NOTEQUAL long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_0NOTEQUAL),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_ADD",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_ADD),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_ADD long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_ADD),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_SUB",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_SUB),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_SUB long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_SUB),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_MUL",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_MUL),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_MUL long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_MUL),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_DIV",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DIV),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_DIV long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_DIV),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_MOD",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_MOD),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_MOD long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_MOD),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_LSHIFT",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_LSHIFT),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_LSHIFT long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_LSHIFT),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_RSHIFT",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_RSHIFT),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_RSHIFT long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_RSHIFT),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_BOOLAND",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_BOOLAND),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_BOOLAND long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_BOOLAND),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_BOOLOR",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_BOOLOR),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_BOOLOR long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_BOOLOR),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NUMEQUAL",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NUMEQUAL),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NUMEQUAL long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NUMEQUAL),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NUMEQUALVERIFY",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NUMEQUALVERIFY),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NUMEQUALVERIFY long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NUMEQUALVERIFY),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NUMNOTEQUAL",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NUMNOTEQUAL),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NUMNOTEQUAL long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NUMNOTEQUAL),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_LESSTHAN",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_LESSTHAN),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_LESSTHAN long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_LESSTHAN),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_GREATERTHAN",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_GREATERTHAN),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_GREATERTHAN long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_GREATERTHAN),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_LESSTHANOREQUAL",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_LESSTHANOREQUAL),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_LESSTHANOREQUAL long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_LESSTHANOREQUAL),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_GREATERTHANOREQUAL",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_GREATERTHANOREQUAL),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_GREATERTHANOREQUAL long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_GREATERTHANOREQUAL),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_MIN",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_MIN),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_MIN long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_MIN),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_MAX",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_MAX),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_MAX long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_MAX),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_WITHIN",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_WITHIN),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_WITHIN long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_WITHIN),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_RIPEMD160",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_RIPEMD160),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_RIPEMD160 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_RIPEMD160),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_SHA1",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_SHA1),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_SHA1 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_SHA1),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_SHA256",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_SHA256),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_SHA256 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_SHA256),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_HASH160",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_HASH160),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_HASH160 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_HASH160),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_HASH256",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_HASH256),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_HASH256 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_HASH256),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_CODESAPERATOR",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_CODESEPARATOR),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_CODESEPARATOR long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_CODESEPARATOR),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_CHECKSIG",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_CHECKSIG),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_CHECKSIG long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_CHECKSIG),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_CHECKSIGVERIFY",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_CHECKSIGVERIFY),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_CHECKSIGVERIFY long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_CHECKSIGVERIFY),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_CHECKMULTISIG",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_CHECKMULTISIG),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_CHECKMULTISIG long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_CHECKMULTISIG),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_CHECKMULTISIGVERIFY",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_CHECKMULTISIGVERIFY),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_CHECKMULTISIGVERIFY long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_CHECKMULTISIGVERIFY),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NOP1",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP1),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NOP1 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP1),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NOP2",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP2),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NOP2 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP2),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NOP3",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP3),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NOP3 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP3),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NOP4",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP4),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NOP4 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP4),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NOP5",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP5),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NOP5 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP5),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NOP6",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP6),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NOP6 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP6),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NOP7",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP7),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NOP7 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP7),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NOP8",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP8),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NOP8 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP8),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NOP9",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP9),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NOP9 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP9),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_NOP10",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP10),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_NOP10 long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_NOP10),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_PUBKEYHASH",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_PUBKEYHASH),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_PUBKEYHASH long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_PUBKEYHASH),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_PUBKEY",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_PUBKEY),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_PUBKEY long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_PUBKEY),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
		{
			name: "OP_INVALIDOPCODE",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_INVALIDOPCODE),
				Data:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "OP_INVALIDOPCODE long",
			pop: &parsescript.ParsedOpcode{
				Opcode: opcode.MkOpcode(opcode.OP_INVALIDOPCODE),
				Data:   make([]byte, 1),
			},
			expectedErr: txscripterr.ScriptError(txscripterr.ErrInternal, ""),
		},
	}

	for _, test := range tests {
		_, err := popBytes(test.pop)
		if e := tstCheckScriptError(err, test.expectedErr); e != nil {
			t.Errorf("Parsed opcode test '%s': %v", test.name, e)
			continue
		}
	}
}

// TestPushedData ensured the PushedData function extracts the expected data out
// of various scripts.
func TestPushedData(t *testing.T) {
	tests := []struct {
		script string
		out    [][]byte
		valid  bool
	}{
		{
			"0 IF 0 ELSE 2 ENDIF",
			[][]byte{nil, nil},
			true,
		},
		{
			"16777216 10000000",
			[][]byte{
				{0x00, 0x00, 0x00, 0x01}, // 16777216
				{0x80, 0x96, 0x98, 0x00}, // 10000000
			},
			true,
		},
		{
			"DUP HASH160 '17VZNX1SN5NtKa8UQFxwQbFeFc3iqRYhem' EQUALVERIFY CHECKSIG",
			[][]byte{
				// 17VZNX1SN5NtKa8UQFxwQbFeFc3iqRYhem
				{
					0x31, 0x37, 0x56, 0x5a, 0x4e, 0x58, 0x31, 0x53, 0x4e, 0x35,
					0x4e, 0x74, 0x4b, 0x61, 0x38, 0x55, 0x51, 0x46, 0x78, 0x77,
					0x51, 0x62, 0x46, 0x65, 0x46, 0x63, 0x33, 0x69, 0x71, 0x52,
					0x59, 0x68, 0x65, 0x6d,
				},
			},
			true,
		},
		{
			"PUSHDATA4 1000 EQUAL",
			nil,
			false,
		},
	}

	for i, test := range tests {
		script := mustParseShortForm(test.script)
		data, err := PushedData(script)
		if test.valid && err != nil {
			t.Errorf("TestPushedData failed test #%d: %v\n", i, err)
			continue
		} else if !test.valid && err == nil {
			t.Errorf("TestPushedData failed test #%d: test should "+
				"be invalid\n", i)
			continue
		}
		if !reflect.DeepEqual(data, test.out) {
			t.Errorf("TestPushedData failed test #%d: want: %x "+
				"got: %x\n", i, test.out, data)
		}
	}
}

// TestHasCanonicalPush ensures the canonicalPush function works as expected.
func TestHasCanonicalPush(t *testing.T) {
	for i := 0; i < 65535; i++ {
		script, err := scriptbuilder.NewScriptBuilder().AddInt64(int64(i)).Script()
		if err != nil {
			t.Errorf("Script: test #%d unexpected error: %v\n", i,
				err)
			continue
		}
		if result := IsPushOnlyScript(script); !result {
			t.Errorf("IsPushOnlyScript: test #%d failed: %x\n", i,
				script)
			continue
		}
		pops, err := parsescript.ParseScript(script)
		if err != nil {
			t.Errorf("parseScript: #%d failed: %v", i, err)
			continue
		}
		for _, pop := range pops {
			if result := canonicalPush(pop); !result {
				t.Errorf("canonicalPush: test #%d failed: %x\n",
					i, script)
				break
			}
		}
	}
	for i := 0; i <= params.MaxScriptElementSize; i++ {
		builder := scriptbuilder.NewScriptBuilder()
		builder.AddData(bytes.Repeat([]byte{0x49}, i))
		script, err := builder.Script()
		if err != nil {
			t.Errorf("StandardPushesTests test #%d unexpected error: %v\n", i, err)
			continue
		}
		if result := IsPushOnlyScript(script); !result {
			t.Errorf("StandardPushesTests IsPushOnlyScript test #%d failed: %x\n", i, script)
			continue
		}
		pops, err := parsescript.ParseScript(script)
		if err != nil {
			t.Errorf("StandardPushesTests #%d failed to TstParseScript: %v", i, err)
			continue
		}
		for _, pop := range pops {
			if result := canonicalPush(pop); !result {
				t.Errorf("StandardPushesTests TstHasCanonicalPushes test #%d failed: %x\n", i, script)
				break
			}
		}
	}
}

// TestGetPreciseSigOps ensures the more precise signature operation counting
// mechanism which includes signatures in P2SH scripts works as expected.
func TestGetPreciseSigOps(t *testing.T) {
	tests := []struct {
		name      string
		scriptSig []byte
		nSigOps   int
	}{
		{
			name:      "scriptSig doesn't parse",
			scriptSig: mustParseShortForm("PUSHDATA1 0x02"),
		},
		{
			name:      "scriptSig isn't push only",
			scriptSig: mustParseShortForm("1 DUP"),
			nSigOps:   0,
		},
		{
			name:      "scriptSig length 0",
			scriptSig: nil,
			nSigOps:   0,
		},
		{
			name: "No script at the end",
			// No script at end but still push only.
			scriptSig: mustParseShortForm("1 1"),
			nSigOps:   0,
		},
		{
			name:      "pushed script doesn't parse",
			scriptSig: mustParseShortForm("DATA_2 PUSHDATA1 0x02"),
		},
	}

	// The signature in the p2sh script is nonsensical for the tests since
	// this script will never be executed.  What matters is that it matches
	// the right pattern.
	pkScript := mustParseShortForm("HASH160 DATA_20 0x433ec2ac1ffa1b7b7d0" +
		"27f564529c57197f9ae88 EQUAL")
	for _, test := range tests {
		count := GetPreciseSigOpCount(test.scriptSig, pkScript, true)
		if count != test.nSigOps {
			t.Errorf("%s: expected count of %d, got %d", test.name,
				test.nSigOps, count)
		}
	}
}

// TestGetWitnessSigOpCount tests that the sig op counting for p2wkh, p2wsh,
// nested p2sh, and invalid variants are counted properly.
func TestGetWitnessSigOpCount(t *testing.T) {
	tests := []struct {
		name string

		sigScript []byte
		pkScript  []byte
		witness   wire.TxWitness

		numSigOps int
	}{
		// A regualr p2wkh witness program. The output being spent
		// should only have a single sig-op counted.
		{
			name: "p2wkh",
			pkScript: mustParseShortForm("OP_0 DATA_20 " +
				"0x365ab47888e150ff46f8d51bce36dcd680f1283f"),
			witness: wire.TxWitness{
				hexToBytes("3045022100ee9fe8f9487afa977" +
					"6647ebcf0883ce0cd37454d7ce19889d34ba2c9" +
					"9ce5a9f402200341cb469d0efd3955acb9e46" +
					"f568d7e2cc10f9084aaff94ced6dc50a59134ad01"),
				hexToBytes("03f0000d0639a22bfaf217e4c9428" +
					"9c2b0cc7fa1036f7fd5d9f61a9d6ec153100e"),
			},
			numSigOps: 1,
		},
		// A p2wkh witness program nested within a p2sh output script.
		// The pattern should be recognized properly and attribute only
		// a single sig op.
		{
			name: "nested p2sh",
			sigScript: hexToBytes("160014ad0ffa2e387f07" +
				"e7ead14dc56d5a97dbd6ff5a23"),
			pkScript: mustParseShortForm("HASH160 DATA_20 " +
				"0xb3a84b564602a9d68b4c9f19c2ea61458ff7826c EQUAL"),
			witness: wire.TxWitness{
				hexToBytes("3045022100cb1c2ac1ff1d57d" +
					"db98f7bdead905f8bf5bcc8641b029ce8eef25" +
					"c75a9e22a4702203be621b5c86b771288706be5" +
					"a7eee1db4fceabf9afb7583c1cc6ee3f8297b21201"),
				hexToBytes("03f0000d0639a22bfaf217e4c9" +
					"4289c2b0cc7fa1036f7fd5d9f61a9d6ec153100e"),
			},
			numSigOps: 1,
		},
		// A p2sh script that spends a 2-of-2 multi-sig output.
		{
			name:      "p2wsh multi-sig spend",
			numSigOps: 2,
			pkScript: hexToBytes("0020e112b88a0cd87ba387f" +
				"449d443ee2596eb353beb1f0351ab2cba8909d875db23"),
			witness: wire.TxWitness{
				hexToBytes("522103b05faca7ceda92b493" +
					"3f7acdf874a93de0dc7edc461832031cd69cbb1d1e" +
					"6fae2102e39092e031c1621c902e3704424e8d8" +
					"3ca481d4d4eeae1b7970f51c78231207e52ae"),
			},
		},
		// A p2wsh witness program. However, the witness script fails
		// to parse after the valid portion of the script. As a result,
		// the valid portion of the script should still be counted.
		{
			name:      "witness script doesn't parse",
			numSigOps: 1,
			pkScript: hexToBytes("0020e112b88a0cd87ba387f44" +
				"9d443ee2596eb353beb1f0351ab2cba8909d875db23"),
			witness: wire.TxWitness{
				mustParseShortForm("DUP HASH160 " +
					"'17VZNX1SN5NtKa8UQFxwQbFeFc3iqRYhem'" +
					" EQUALVERIFY CHECKSIG DATA_20 0x91"),
			},
		},
	}

	for _, test := range tests {
		count := GetWitnessSigOpCount(test.sigScript, test.pkScript,
			test.witness)
		if count != test.numSigOps {
			t.Errorf("%s: expected count of %d, got %d", test.name,
				test.numSigOps, count)
		}
	}
}

// TestRemoveOpcodes ensures that removing opcodes from scripts behaves as
// expected.
func TestRemoveOpcodes(t *testing.T) {
	tests := []struct {
		name   string
		before string
		remove byte
		err    er.R
		after  string
	}{
		{
			// Nothing to remove.
			name:   "nothing to remove",
			before: "NOP",
			remove: opcode.OP_CODESEPARATOR,
			after:  "NOP",
		},
		{
			// Test basic opcode removal.
			name:   "codeseparator 1",
			before: "NOP CODESEPARATOR TRUE",
			remove: opcode.OP_CODESEPARATOR,
			after:  "NOP TRUE",
		},
		{
			// The opcode in question is actually part of the data
			// in a previous opcode.
			name:   "codeseparator by coincidence",
			before: "NOP DATA_1 CODESEPARATOR TRUE",
			remove: opcode.OP_CODESEPARATOR,
			after:  "NOP DATA_1 CODESEPARATOR TRUE",
		},
		{
			name:   "invalid opcode",
			before: "CAT",
			remove: opcode.OP_CODESEPARATOR,
			after:  "CAT",
		},
		{
			name:   "invalid length (instruction)",
			before: "PUSHDATA1",
			remove: opcode.OP_CODESEPARATOR,
			err:    txscripterr.ScriptError(txscripterr.ErrMalformedPush, ""),
		},
		{
			name:   "invalid length (data)",
			before: "PUSHDATA1 0xff 0xfe",
			remove: opcode.OP_CODESEPARATOR,
			err:    txscripterr.ScriptError(txscripterr.ErrMalformedPush, ""),
		},
	}

	// tstRemoveOpcode is a convenience function to parse the provided
	// raw script, remove the passed opcode, then unparse the result back
	// into a raw script.
	tstRemoveOpcode := func(script []byte, opcode byte) ([]byte, er.R) {
		pops, err := parsescript.ParseScript(script)
		if err != nil {
			return nil, err
		}
		pops = removeOpcode(pops, opcode)
		return unparseScript(pops)
	}

	for _, test := range tests {
		before := mustParseShortForm(test.before)
		after := mustParseShortForm(test.after)
		result, err := tstRemoveOpcode(before, test.remove)
		if e := tstCheckScriptError(err, test.err); e != nil {
			t.Errorf("%s: %v", test.name, e)
			continue
		}

		if !bytes.Equal(after, result) {
			t.Errorf("%s: value does not equal expected: exp: %q"+
				" got: %q", test.name, after, result)
		}
	}
}

// TestRemoveOpcodeByData ensures that removing data carrying opcodes based on
// the data they contain works as expected.
func TestRemoveOpcodeByData(t *testing.T) {
	tests := []struct {
		name   string
		before []byte
		remove []byte
		err    er.R
		after  []byte
	}{
		{
			name:   "nothing to do",
			before: []byte{opcode.OP_NOP},
			remove: []byte{1, 2, 3, 4},
			after:  []byte{opcode.OP_NOP},
		},
		{
			name:   "simple case",
			before: []byte{opcode.OP_DATA_4, 1, 2, 3, 4},
			remove: []byte{1, 2, 3, 4},
			after:  nil,
		},
		{
			name:   "simple case (miss)",
			before: []byte{opcode.OP_DATA_4, 1, 2, 3, 4},
			remove: []byte{1, 2, 3, 5},
			after:  []byte{opcode.OP_DATA_4, 1, 2, 3, 4},
		},
		{
			// padded to keep it canonical.
			name: "simple case (pushdata1)",
			before: append(append([]byte{opcode.OP_PUSHDATA1, 76},
				bytes.Repeat([]byte{0}, 72)...),
				[]byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 4},
			after:  nil,
		},
		{
			name: "simple case (pushdata1 miss)",
			before: append(append([]byte{opcode.OP_PUSHDATA1, 76},
				bytes.Repeat([]byte{0}, 72)...),
				[]byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 5},
			after: append(append([]byte{opcode.OP_PUSHDATA1, 76},
				bytes.Repeat([]byte{0}, 72)...),
				[]byte{1, 2, 3, 4}...),
		},
		{
			name:   "simple case (pushdata1 miss noncanonical)",
			before: []byte{opcode.OP_PUSHDATA1, 4, 1, 2, 3, 4},
			remove: []byte{1, 2, 3, 4},
			after:  []byte{opcode.OP_PUSHDATA1, 4, 1, 2, 3, 4},
		},
		{
			name: "simple case (pushdata2)",
			before: append(append([]byte{opcode.OP_PUSHDATA2, 0, 1},
				bytes.Repeat([]byte{0}, 252)...),
				[]byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 4},
			after:  nil,
		},
		{
			name: "simple case (pushdata2 miss)",
			before: append(append([]byte{opcode.OP_PUSHDATA2, 0, 1},
				bytes.Repeat([]byte{0}, 252)...),
				[]byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 4, 5},
			after: append(append([]byte{opcode.OP_PUSHDATA2, 0, 1},
				bytes.Repeat([]byte{0}, 252)...),
				[]byte{1, 2, 3, 4}...),
		},
		{
			name:   "simple case (pushdata2 miss noncanonical)",
			before: []byte{opcode.OP_PUSHDATA2, 4, 0, 1, 2, 3, 4},
			remove: []byte{1, 2, 3, 4},
			after:  []byte{opcode.OP_PUSHDATA2, 4, 0, 1, 2, 3, 4},
		},
		{
			// This is padded to make the push canonical.
			name: "simple case (pushdata4)",
			before: append(append([]byte{opcode.OP_PUSHDATA4, 0, 0, 1, 0},
				bytes.Repeat([]byte{0}, 65532)...),
				[]byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 4},
			after:  nil,
		},
		{
			name:   "simple case (pushdata4 miss noncanonical)",
			before: []byte{opcode.OP_PUSHDATA4, 4, 0, 0, 0, 1, 2, 3, 4},
			remove: []byte{1, 2, 3, 4},
			after:  []byte{opcode.OP_PUSHDATA4, 4, 0, 0, 0, 1, 2, 3, 4},
		},
		{
			// This is padded to make the push canonical.
			name: "simple case (pushdata4 miss)",
			before: append(append([]byte{opcode.OP_PUSHDATA4, 0, 0, 1, 0},
				bytes.Repeat([]byte{0}, 65532)...), []byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 4, 5},
			after: append(append([]byte{opcode.OP_PUSHDATA4, 0, 0, 1, 0},
				bytes.Repeat([]byte{0}, 65532)...), []byte{1, 2, 3, 4}...),
		},
		{
			name:   "invalid opcode ",
			before: []byte{opcode.OP_UNKNOWN187},
			remove: []byte{1, 2, 3, 4},
			after:  []byte{opcode.OP_UNKNOWN187},
		},
		{
			name:   "invalid length (instruction)",
			before: []byte{opcode.OP_PUSHDATA1},
			remove: []byte{1, 2, 3, 4},
			err:    txscripterr.ScriptError(txscripterr.ErrMalformedPush, ""),
		},
		{
			name:   "invalid length (data)",
			before: []byte{opcode.OP_PUSHDATA1, 255, 254},
			remove: []byte{1, 2, 3, 4},
			err:    txscripterr.ScriptError(txscripterr.ErrMalformedPush, ""),
		},
	}

	// tstRemoveOpcodeByData is a convenience function to parse the provided
	// raw script, remove the passed data, then unparse the result back
	// into a raw script.
	tstRemoveOpcodeByData := func(script []byte, data []byte) ([]byte, er.R) {
		pops, err := parsescript.ParseScript(script)
		if err != nil {
			return nil, err
		}
		pops = removeOpcodeByData(pops, data)
		return unparseScript(pops)
	}

	for _, test := range tests {
		result, err := tstRemoveOpcodeByData(test.before, test.remove)
		if e := tstCheckScriptError(err, test.err); e != nil {
			t.Errorf("%s: %v", test.name, e)
			continue
		}

		if !bytes.Equal(test.after, result) {
			t.Errorf("%s: value does not equal expected: exp: %q"+
				" got: %q", test.name, test.after, result)
		}
	}
}

// TestIsPayToScriptHash ensures the IsPayToScriptHash function returns the
// expected results for all the scripts in scriptClassTests.
func TestIsPayToScriptHash(t *testing.T) {
	for _, test := range scriptClassTests {
		script := mustParseShortForm(test.script)
		shouldBe := (test.class == ScriptHashTy)
		p2sh := IsPayToScriptHash(script)
		if p2sh != shouldBe {
			t.Errorf("%s: expected p2sh %v, got %v", test.name,
				shouldBe, p2sh)
		}
	}
}

// TestIsPayToWitnessScriptHash ensures the IsPayToWitnessScriptHash function
// returns the expected results for all the scripts in scriptClassTests.
func TestIsPayToWitnessScriptHash(t *testing.T) {
	for _, test := range scriptClassTests {
		script := mustParseShortForm(test.script)
		shouldBe := (test.class == WitnessV0ScriptHashTy)
		p2wsh := IsPayToWitnessScriptHash(script)
		if p2wsh != shouldBe {
			t.Errorf("%s: expected p2wsh %v, got %v", test.name,
				shouldBe, p2wsh)
		}
	}
}

// TestIsPayToWitnessPubKeyHash ensures the IsPayToWitnessPubKeyHash function
// returns the expected results for all the scripts in scriptClassTests.
func TestIsPayToWitnessPubKeyHash(t *testing.T) {
	for _, test := range scriptClassTests {
		script := mustParseShortForm(test.script)
		shouldBe := (test.class == WitnessV0PubKeyHashTy)
		p2wkh := IsPayToWitnessPubKeyHash(script)
		if p2wkh != shouldBe {
			t.Errorf("%s: expected p2wkh %v, got %v", test.name,
				shouldBe, p2wkh)
		}
	}
}

// TestHasCanonicalPushes ensures the canonicalPush function properly determines
// what is considered a canonical push for the purposes of removeOpcodeByData.
func TestHasCanonicalPushes(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		expected bool
	}{
		{
			name: "does not parse",
			script: "0x046708afdb0fe5548271967f1a67130b7105cd6a82" +
				"8e03909a67962e0ea1f61d",
			expected: false,
		},
		{
			name:     "non-canonical push",
			script:   "PUSHDATA1 0x04 0x01020304",
			expected: false,
		},
	}

	for i, test := range tests {
		script := mustParseShortForm(test.script)
		pops, err := parsescript.ParseScript(script)
		if err != nil {
			if test.expected {
				t.Errorf("TstParseScript #%d failed: %v", i, err)
			}
			continue
		}
		for _, pop := range pops {
			if canonicalPush(pop) != test.expected {
				t.Errorf("canonicalPush: #%d (%s) wrong result"+
					"\ngot: %v\nwant: %v", i, test.name,
					true, test.expected)
				break
			}
		}
	}
}

// TestIsPushOnlyScript ensures the IsPushOnlyScript function returns the
// expected results.
func TestIsPushOnlyScript(t *testing.T) {
	test := struct {
		name     string
		script   []byte
		expected bool
	}{
		name: "does not parse",
		script: mustParseShortForm("0x046708afdb0fe5548271967f1a67130" +
			"b7105cd6a828e03909a67962e0ea1f61d"),
		expected: false,
	}

	if IsPushOnlyScript(test.script) != test.expected {
		t.Errorf("IsPushOnlyScript (%s) wrong result\ngot: %v\nwant: "+
			"%v", test.name, true, test.expected)
	}
}

// TestIsUnspendable ensures the IsUnspendable function returns the expected
// results.
func TestIsUnspendable(t *testing.T) {
	tests := []struct {
		name     string
		pkScript []byte
		expected bool
	}{
		{
			// Unspendable
			pkScript: []byte{0x6a, 0x04, 0x74, 0x65, 0x73, 0x74},
			expected: true,
		},
		{
			// Spendable
			pkScript: []byte{
				0x76, 0xa9, 0x14, 0x29, 0x95, 0xa0,
				0xfe, 0x68, 0x43, 0xfa, 0x9b, 0x95, 0x45,
				0x97, 0xf0, 0xdc, 0xa7, 0xa4, 0x4d, 0xf6,
				0xfa, 0x0b, 0x5c, 0x88, 0xac,
			},
			expected: false,
		},
	}

	for i, test := range tests {
		res := IsUnspendable(test.pkScript)
		if res != test.expected {
			t.Errorf("TestIsUnspendable #%d failed: got %v want %v",
				i, res, test.expected)
			continue
		}
	}
}

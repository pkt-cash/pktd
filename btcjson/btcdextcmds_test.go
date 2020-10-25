// Copyright (c) 2014-2016 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcjson_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"github.com/json-iterator/go"
	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/btcjson"
)

// TestBtcdExtCmds tests all of the pktd extended commands marshal and unmarshal
// into valid results include handling of optional fields being omitted in the
// marshaled command, while optional fields with defaults have the default
// assigned on unmarshaled commands.
func TestBtcdExtCmds(t *testing.T) {
	t.Parallel()

	testID := int(1)
	tests := []struct {
		name         string
		newCmd       func() (interface{}, er.R)
		staticCmd    func() interface{}
		marshaled   string
		unmarshaled interface{}
	}{
		{
			name: "debuglevel",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("debuglevel", "trace")
			},
			staticCmd: func() interface{} {
				return btcjson.NewDebugLevelCmd("trace")
			},
			marshaled: `{"jsonrpc":"1.0","method":"debuglevel","params":["trace"],"id":1}`,
			unmarshaled: &btcjson.DebugLevelCmd{
				LevelSpec: "trace",
			},
		},
		{
			name: "node",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("node", btcjson.NRemove, "1.1.1.1")
			},
			staticCmd: func() interface{} {
				return btcjson.NewNodeCmd("remove", "1.1.1.1", nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"node","params":["remove","1.1.1.1"],"id":1}`,
			unmarshaled: &btcjson.NodeCmd{
				SubCmd: btcjson.NRemove,
				Target: "1.1.1.1",
			},
		},
		{
			name: "node",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("node", btcjson.NDisconnect, "1.1.1.1")
			},
			staticCmd: func() interface{} {
				return btcjson.NewNodeCmd("disconnect", "1.1.1.1", nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"node","params":["disconnect","1.1.1.1"],"id":1}`,
			unmarshaled: &btcjson.NodeCmd{
				SubCmd: btcjson.NDisconnect,
				Target: "1.1.1.1",
			},
		},
		{
			name: "node",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("node", btcjson.NConnect, "1.1.1.1", "perm")
			},
			staticCmd: func() interface{} {
				return btcjson.NewNodeCmd("connect", "1.1.1.1", btcjson.String("perm"))
			},
			marshaled: `{"jsonrpc":"1.0","method":"node","params":["connect","1.1.1.1","perm"],"id":1}`,
			unmarshaled: &btcjson.NodeCmd{
				SubCmd:        btcjson.NConnect,
				Target:        "1.1.1.1",
				ConnectSubCmd: btcjson.String("perm"),
			},
		},
		{
			name: "node",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("node", btcjson.NConnect, "1.1.1.1", "temp")
			},
			staticCmd: func() interface{} {
				return btcjson.NewNodeCmd("connect", "1.1.1.1", btcjson.String("temp"))
			},
			marshaled: `{"jsonrpc":"1.0","method":"node","params":["connect","1.1.1.1","temp"],"id":1}`,
			unmarshaled: &btcjson.NodeCmd{
				SubCmd:        btcjson.NConnect,
				Target:        "1.1.1.1",
				ConnectSubCmd: btcjson.String("temp"),
			},
		},
		{
			name: "generate",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("generate", 1)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGenerateCmd(1)
			},
			marshaled: `{"jsonrpc":"1.0","method":"generate","params":[1],"id":1}`,
			unmarshaled: &btcjson.GenerateCmd{
				NumBlocks: 1,
			},
		},
		{
			name: "getbestblock",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getbestblock")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetBestBlockCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getbestblock","params":[],"id":1}`,
			unmarshaled: &btcjson.GetBestBlockCmd{},
		},
		{
			name: "getcurrentnet",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getcurrentnet")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetCurrentNetCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getcurrentnet","params":[],"id":1}`,
			unmarshaled: &btcjson.GetCurrentNetCmd{},
		},
		{
			name: "getheaders",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getheaders", []string{}, "")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetHeadersCmd(
					[]string{},
					"",
				)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getheaders","params":[[],""],"id":1}`,
			unmarshaled: &btcjson.GetHeadersCmd{
				BlockLocators: []string{},
				HashStop:      "",
			},
		},
		{
			name: "getheaders - with arguments",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getheaders", []string{"000000000000000001f1739002418e2f9a84c47a4fd2a0eb7a787a6b7dc12f16", "0000000000000000026f4b7f56eef057b32167eb5ad9ff62006f1807b7336d10"}, "000000000000000000ba33b33e1fad70b69e234fc24414dd47113bff38f523f7")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetHeadersCmd(
					[]string{
						"000000000000000001f1739002418e2f9a84c47a4fd2a0eb7a787a6b7dc12f16",
						"0000000000000000026f4b7f56eef057b32167eb5ad9ff62006f1807b7336d10",
					},
					"000000000000000000ba33b33e1fad70b69e234fc24414dd47113bff38f523f7",
				)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getheaders","params":[["000000000000000001f1739002418e2f9a84c47a4fd2a0eb7a787a6b7dc12f16","0000000000000000026f4b7f56eef057b32167eb5ad9ff62006f1807b7336d10"],"000000000000000000ba33b33e1fad70b69e234fc24414dd47113bff38f523f7"],"id":1}`,
			unmarshaled: &btcjson.GetHeadersCmd{
				BlockLocators: []string{
					"000000000000000001f1739002418e2f9a84c47a4fd2a0eb7a787a6b7dc12f16",
					"0000000000000000026f4b7f56eef057b32167eb5ad9ff62006f1807b7336d10",
				},
				HashStop: "000000000000000000ba33b33e1fad70b69e234fc24414dd47113bff38f523f7",
			},
		},
		{
			name: "version",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("version")
			},
			staticCmd: func() interface{} {
				return btcjson.NewVersionCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"version","params":[],"id":1}`,
			unmarshaled: &btcjson.VersionCmd{},
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Marshal the command as created by the new static command
		// creation function.
		marshaled, err := btcjson.MarshalCmd(testID, test.staticCmd())
		if err != nil {
			t.Errorf("MarshalCmd #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !bytes.Equal(marshaled, []byte(test.marshaled)) {
			t.Errorf("Test #%d (%s) unexpected marshaled data - "+
				"got %s, want %s", i, test.name, marshaled,
				test.marshaled)
			continue
		}

		// Ensure the command is created without error via the generic
		// new command creation function.
		cmd, err := test.newCmd()
		if err != nil {
			t.Errorf("Test #%d (%s) unexpected NewCmd error: %v ",
				i, test.name, err)
		}

		// Marshal the command as created by the generic new command
		// creation function.
		marshaled, err = btcjson.MarshalCmd(testID, cmd)
		if err != nil {
			t.Errorf("MarshalCmd #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !bytes.Equal(marshaled, []byte(test.marshaled)) {
			t.Errorf("Test #%d (%s) unexpected marshaled data - "+
				"got %s, want %s", i, test.name, marshaled,
				test.marshaled)
			continue
		}

		var request btcjson.Request
		if err := jsoniter.Unmarshal(marshaled, &request); err != nil {
			t.Errorf("Test #%d (%s) unexpected error while "+
				"unmarshalling JSON-RPC request: %v", i,
				test.name, err)
			continue
		}

		cmd, err = btcjson.UnmarshalCmd(&request)
		if err != nil {
			t.Errorf("UnmarshalCmd #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !reflect.DeepEqual(cmd, test.unmarshaled) {
			t.Errorf("Test #%d (%s) unexpected unmarshaled command "+
				"- got %s, want %s", i, test.name,
				fmt.Sprintf("(%T) %+[1]v", cmd),
				fmt.Sprintf("(%T) %+[1]v\n", test.unmarshaled))
			continue
		}
	}
}

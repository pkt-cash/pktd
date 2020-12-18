// Copyright (c) 2014-2017 The btcsuite developers
// Copyright (c) 2015-2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcjson_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/btcjson"
)

// TestChainSvrWsCmds tests all of the chain server websocket-specific commands
// marshal and unmarshal into valid results include handling of optional fields
// being omitted in the marshaled command, while optional fields with defaults
// have the default assigned on unmarshaled commands.
func TestChainSvrWsCmds(t *testing.T) {
	testID := int(1)
	tests := []struct {
		name        string
		newCmd      func() (interface{}, er.R)
		staticCmd   func() interface{}
		marshaled   string
		unmarshaled interface{}
	}{
		{
			name: "authenticate",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("authenticate", "user", "pass")
			},
			staticCmd: func() interface{} {
				return btcjson.NewAuthenticateCmd("user", "pass")
			},
			marshaled:   `{"jsonrpc":"1.0","method":"authenticate","params":["user","pass"],"id":1}`,
			unmarshaled: &btcjson.AuthenticateCmd{Username: "user", Passphrase: "pass"},
		},
		{
			name: "notifyblocks",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("notifyblocks")
			},
			staticCmd: func() interface{} {
				return btcjson.NewNotifyBlocksCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"notifyblocks","params":[],"id":1}`,
			unmarshaled: &btcjson.NotifyBlocksCmd{},
		},
		{
			name: "stopnotifyblocks",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("stopnotifyblocks")
			},
			staticCmd: func() interface{} {
				return btcjson.NewStopNotifyBlocksCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"stopnotifyblocks","params":[],"id":1}`,
			unmarshaled: &btcjson.StopNotifyBlocksCmd{},
		},
		{
			name: "notifynewtransactions",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("notifynewtransactions")
			},
			staticCmd: func() interface{} {
				return btcjson.NewNotifyNewTransactionsCmd(nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"notifynewtransactions","params":[],"id":1}`,
			unmarshaled: &btcjson.NotifyNewTransactionsCmd{
				Verbose: btcjson.Bool(false),
			},
		},
		{
			name: "notifynewtransactions optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("notifynewtransactions", true)
			},
			staticCmd: func() interface{} {
				return btcjson.NewNotifyNewTransactionsCmd(btcjson.Bool(true))
			},
			marshaled: `{"jsonrpc":"1.0","method":"notifynewtransactions","params":[true],"id":1}`,
			unmarshaled: &btcjson.NotifyNewTransactionsCmd{
				Verbose: btcjson.Bool(true),
			},
		},
		{
			name: "stopnotifynewtransactions",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("stopnotifynewtransactions")
			},
			staticCmd: func() interface{} {
				return btcjson.NewStopNotifyNewTransactionsCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"stopnotifynewtransactions","params":[],"id":1}`,
			unmarshaled: &btcjson.StopNotifyNewTransactionsCmd{},
		},
		{
			name: "notifyreceived",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("notifyreceived", []string{"1Address"})
			},
			staticCmd: func() interface{} {
				return btcjson.NewNotifyReceivedCmd([]string{"1Address"})
			},
			marshaled: `{"jsonrpc":"1.0","method":"notifyreceived","params":[["1Address"]],"id":1}`,
			unmarshaled: &btcjson.NotifyReceivedCmd{
				Addresses: []string{"1Address"},
			},
		},
		{
			name: "stopnotifyreceived",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("stopnotifyreceived", []string{"1Address"})
			},
			staticCmd: func() interface{} {
				return btcjson.NewStopNotifyReceivedCmd([]string{"1Address"})
			},
			marshaled: `{"jsonrpc":"1.0","method":"stopnotifyreceived","params":[["1Address"]],"id":1}`,
			unmarshaled: &btcjson.StopNotifyReceivedCmd{
				Addresses: []string{"1Address"},
			},
		},
		{
			name: "notifyspent",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("notifyspent", `[{"hash":"123","index":0}]`)
			},
			staticCmd: func() interface{} {
				ops := []btcjson.OutPoint{{Hash: "123", Index: 0}}
				return btcjson.NewNotifySpentCmd(ops)
			},
			marshaled: `{"jsonrpc":"1.0","method":"notifyspent","params":[[{"hash":"123","index":0}]],"id":1}`,
			unmarshaled: &btcjson.NotifySpentCmd{
				OutPoints: []btcjson.OutPoint{{Hash: "123", Index: 0}},
			},
		},
		{
			name: "stopnotifyspent",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("stopnotifyspent", `[{"hash":"123","index":0}]`)
			},
			staticCmd: func() interface{} {
				ops := []btcjson.OutPoint{{Hash: "123", Index: 0}}
				return btcjson.NewStopNotifySpentCmd(ops)
			},
			marshaled: `{"jsonrpc":"1.0","method":"stopnotifyspent","params":[[{"hash":"123","index":0}]],"id":1}`,
			unmarshaled: &btcjson.StopNotifySpentCmd{
				OutPoints: []btcjson.OutPoint{{Hash: "123", Index: 0}},
			},
		},
		{
			name: "rescan",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("rescan", "123", `["1Address"]`, `[{"hash":"0000000000000000000000000000000000000000000000000000000000000123","index":0}]`)
			},
			staticCmd: func() interface{} {
				addrs := []string{"1Address"}
				ops := []btcjson.OutPoint{{
					Hash:  "0000000000000000000000000000000000000000000000000000000000000123",
					Index: 0,
				}}
				return btcjson.NewRescanCmd("123", addrs, ops, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"rescan","params":["123",["1Address"],[{"hash":"0000000000000000000000000000000000000000000000000000000000000123","index":0}]],"id":1}`,
			unmarshaled: &btcjson.RescanCmd{
				BeginBlock: "123",
				Addresses:  []string{"1Address"},
				OutPoints:  []btcjson.OutPoint{{Hash: "0000000000000000000000000000000000000000000000000000000000000123", Index: 0}},
				EndBlock:   nil,
			},
		},
		{
			name: "rescan optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("rescan", "123", `["1Address"]`, `[{"hash":"123","index":0}]`, "456")
			},
			staticCmd: func() interface{} {
				addrs := []string{"1Address"}
				ops := []btcjson.OutPoint{{Hash: "123", Index: 0}}
				return btcjson.NewRescanCmd("123", addrs, ops, btcjson.String("456"))
			},
			marshaled: `{"jsonrpc":"1.0","method":"rescan","params":["123",["1Address"],[{"hash":"123","index":0}],"456"],"id":1}`,
			unmarshaled: &btcjson.RescanCmd{
				BeginBlock: "123",
				Addresses:  []string{"1Address"},
				OutPoints:  []btcjson.OutPoint{{Hash: "123", Index: 0}},
				EndBlock:   btcjson.String("456"),
			},
		},
		{
			name: "loadtxfilter",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("loadtxfilter", false, `["1Address"]`, `[{"hash":"0000000000000000000000000000000000000000000000000000000000000123","index":0}]`)
			},
			staticCmd: func() interface{} {
				addrs := []string{"1Address"}
				ops := []btcjson.OutPoint{{
					Hash:  "0000000000000000000000000000000000000000000000000000000000000123",
					Index: 0,
				}}
				return btcjson.NewLoadTxFilterCmd(false, addrs, ops)
			},
			marshaled: `{"jsonrpc":"1.0","method":"loadtxfilter","params":[false,["1Address"],[{"hash":"0000000000000000000000000000000000000000000000000000000000000123","index":0}]],"id":1}`,
			unmarshaled: &btcjson.LoadTxFilterCmd{
				Reload:    false,
				Addresses: []string{"1Address"},
				OutPoints: []btcjson.OutPoint{{Hash: "0000000000000000000000000000000000000000000000000000000000000123", Index: 0}},
			},
		},
		{
			name: "rescanblocks",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("rescanblocks", `["0000000000000000000000000000000000000000000000000000000000000123"]`)
			},
			staticCmd: func() interface{} {
				blockhashes := []string{"0000000000000000000000000000000000000000000000000000000000000123"}
				return btcjson.NewRescanBlocksCmd(blockhashes)
			},
			marshaled: `{"jsonrpc":"1.0","method":"rescanblocks","params":[["0000000000000000000000000000000000000000000000000000000000000123"]],"id":1}`,
			unmarshaled: &btcjson.RescanBlocksCmd{
				BlockHashes: []string{"0000000000000000000000000000000000000000000000000000000000000123"},
			},
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
				"unmarshaling JSON-RPC request: %v", i,
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

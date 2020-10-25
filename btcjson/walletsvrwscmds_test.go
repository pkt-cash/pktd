// Copyright (c) 2014 The btcsuite developers
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

// TestWalletSvrWsCmds tests all of the wallet server websocket-specific
// commands marshal and unmarshal into valid results include handling of
// optional fields being omitted in the marshaled command, while optional
// fields with defaults have the default assigned on unmarshaled commands.
func TestWalletSvrWsCmds(t *testing.T) {
	t.Parallel()

	testID := int(1)
	tests := []struct {
		name        string
		newCmd      func() (interface{}, er.R)
		staticCmd   func() interface{}
		marshaled   string
		unmarshaled interface{}
	}{
		{
			name: "createencryptedwallet",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("createencryptedwallet", "pass")
			},
			staticCmd: func() interface{} {
				return btcjson.NewCreateEncryptedWalletCmd("pass")
			},
			marshaled:   `{"jsonrpc":"1.0","method":"createencryptedwallet","params":["pass"],"id":1}`,
			unmarshaled: &btcjson.CreateEncryptedWalletCmd{Passphrase: "pass"},
		},
		{
			name: "exportwatchingwallet",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("exportwatchingwallet")
			},
			staticCmd: func() interface{} {
				return btcjson.NewExportWatchingWalletCmd(nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"exportwatchingwallet","params":[],"id":1}`,
			unmarshaled: &btcjson.ExportWatchingWalletCmd{
				Account:  nil,
				Download: btcjson.Bool(false),
			},
		},
		{
			name: "exportwatchingwallet optional1",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("exportwatchingwallet", "acct")
			},
			staticCmd: func() interface{} {
				return btcjson.NewExportWatchingWalletCmd(btcjson.String("acct"), nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"exportwatchingwallet","params":["acct"],"id":1}`,
			unmarshaled: &btcjson.ExportWatchingWalletCmd{
				Account:  btcjson.String("acct"),
				Download: btcjson.Bool(false),
			},
		},
		{
			name: "exportwatchingwallet optional2",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("exportwatchingwallet", "acct", true)
			},
			staticCmd: func() interface{} {
				return btcjson.NewExportWatchingWalletCmd(btcjson.String("acct"),
					btcjson.Bool(true))
			},
			marshaled: `{"jsonrpc":"1.0","method":"exportwatchingwallet","params":["acct",true],"id":1}`,
			unmarshaled: &btcjson.ExportWatchingWalletCmd{
				Account:  btcjson.String("acct"),
				Download: btcjson.Bool(true),
			},
		},
		{
			name: "getunconfirmedbalance",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getunconfirmedbalance")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetUnconfirmedBalanceCmd(nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getunconfirmedbalance","params":[],"id":1}`,
			unmarshaled: &btcjson.GetUnconfirmedBalanceCmd{
				Account: nil,
			},
		},
		{
			name: "getunconfirmedbalance optional1",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getunconfirmedbalance", "acct")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetUnconfirmedBalanceCmd(btcjson.String("acct"))
			},
			marshaled: `{"jsonrpc":"1.0","method":"getunconfirmedbalance","params":["acct"],"id":1}`,
			unmarshaled: &btcjson.GetUnconfirmedBalanceCmd{
				Account: btcjson.String("acct"),
			},
		},
		{
			name: "listaddresstransactions",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listaddresstransactions", `["1Address"]`)
			},
			staticCmd: func() interface{} {
				return btcjson.NewListAddressTransactionsCmd([]string{"1Address"}, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"listaddresstransactions","params":[["1Address"]],"id":1}`,
			unmarshaled: &btcjson.ListAddressTransactionsCmd{
				Addresses: []string{"1Address"},
				Account:   nil,
			},
		},
		{
			name: "listaddresstransactions optional1",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listaddresstransactions", `["1Address"]`, "acct")
			},
			staticCmd: func() interface{} {
				return btcjson.NewListAddressTransactionsCmd([]string{"1Address"},
					btcjson.String("acct"))
			},
			marshaled: `{"jsonrpc":"1.0","method":"listaddresstransactions","params":[["1Address"],"acct"],"id":1}`,
			unmarshaled: &btcjson.ListAddressTransactionsCmd{
				Addresses: []string{"1Address"},
				Account:   btcjson.String("acct"),
			},
		},
		{
			name: "listalltransactions",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listalltransactions")
			},
			staticCmd: func() interface{} {
				return btcjson.NewListAllTransactionsCmd(nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"listalltransactions","params":[],"id":1}`,
			unmarshaled: &btcjson.ListAllTransactionsCmd{
				Account: nil,
			},
		},
		{
			name: "listalltransactions optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listalltransactions", "acct")
			},
			staticCmd: func() interface{} {
				return btcjson.NewListAllTransactionsCmd(btcjson.String("acct"))
			},
			marshaled: `{"jsonrpc":"1.0","method":"listalltransactions","params":["acct"],"id":1}`,
			unmarshaled: &btcjson.ListAllTransactionsCmd{
				Account: btcjson.String("acct"),
			},
		},
		{
			name: "recoveraddresses",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("recoveraddresses", "acct", 10)
			},
			staticCmd: func() interface{} {
				return btcjson.NewRecoverAddressesCmd("acct", 10)
			},
			marshaled: `{"jsonrpc":"1.0","method":"recoveraddresses","params":["acct",10],"id":1}`,
			unmarshaled: &btcjson.RecoverAddressesCmd{
				Account: "acct",
				N:       10,
			},
		},
		{
			name: "walletislocked",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("walletislocked")
			},
			staticCmd: func() interface{} {
				return btcjson.NewWalletIsLockedCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"walletislocked","params":[],"id":1}`,
			unmarshaled: &btcjson.WalletIsLockedCmd{},
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

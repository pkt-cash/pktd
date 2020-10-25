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

// TestWalletSvrWsNtfns tests all of the chain server websocket-specific
// notifications marshal and unmarshal into valid results include handling of
// optional fields being omitted in the marshaled command, while optional
// fields with defaults have the default assigned on unmarshaled commands.
func TestWalletSvrWsNtfns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		newNtfn     func() (interface{}, er.R)
		staticNtfn  func() interface{}
		marshaled   string
		unmarshaled interface{}
	}{
		{
			name: "accountbalance",
			newNtfn: func() (interface{}, er.R) {
				return btcjson.NewCmd("accountbalance", "acct", 1.25, true)
			},
			staticNtfn: func() interface{} {
				return btcjson.NewAccountBalanceNtfn("acct", 1.25, true)
			},
			marshaled: `{"jsonrpc":"1.0","method":"accountbalance","params":["acct",1.25,true],"id":null}`,
			unmarshaled: &btcjson.AccountBalanceNtfn{
				Account:   "acct",
				Balance:   1.25,
				Confirmed: true,
			},
		},
		{
			name: "pktdconnected",
			newNtfn: func() (interface{}, er.R) {
				return btcjson.NewCmd("pktdconnected", true)
			},
			staticNtfn: func() interface{} {
				return btcjson.NewBtcdConnectedNtfn(true)
			},
			marshaled: `{"jsonrpc":"1.0","method":"pktdconnected","params":[true],"id":null}`,
			unmarshaled: &btcjson.BtcdConnectedNtfn{
				Connected: true,
			},
		},
		{
			name: "walletlockstate",
			newNtfn: func() (interface{}, er.R) {
				return btcjson.NewCmd("walletlockstate", true)
			},
			staticNtfn: func() interface{} {
				return btcjson.NewWalletLockStateNtfn(true)
			},
			marshaled: `{"jsonrpc":"1.0","method":"walletlockstate","params":[true],"id":null}`,
			unmarshaled: &btcjson.WalletLockStateNtfn{
				Locked: true,
			},
		},
		{
			name: "newtx",
			newNtfn: func() (interface{}, er.R) {
				return btcjson.NewCmd("newtx", "acct", `{"account":"acct","address":"1Address","category":"send","amount":1.5,"bip125-replaceable":"unknown","fee":0.0001,"confirmations":1,"trusted":true,"txid":"456","walletconflicts":[],"time":12345678,"timereceived":12345876,"vout":789,"otheraccount":"otheracct"}`)
			},
			staticNtfn: func() interface{} {
				result := btcjson.ListTransactionsResult{
					Abandoned:         false,
					Account:           "acct",
					Address:           "1Address",
					BIP125Replaceable: "unknown",
					Category:          "send",
					Amount:            1.5,
					Fee:               btcjson.Float64(0.0001),
					Confirmations:     1,
					TxID:              "456",
					WalletConflicts:   []string{},
					Time:              12345678,
					TimeReceived:      12345876,
					Trusted:           true,
					Vout:              789,
					OtherAccount:      "otheracct",
				}
				return btcjson.NewNewTxNtfn("acct", result)
			},
			marshaled: `{"jsonrpc":"1.0","method":"newtx","params":["acct",{"abandoned":false,"account":"acct","address":"1Address","amount":1.5,"bip125-replaceable":"unknown","category":"send","confirmations":1,"fee":0.0001,"time":12345678,"timereceived":12345876,"trusted":true,"txid":"456","vout":789,"walletconflicts":[],"otheraccount":"otheracct"}],"id":null}`,
			unmarshaled: &btcjson.NewTxNtfn{
				Account: "acct",
				Details: btcjson.ListTransactionsResult{
					Abandoned:         false,
					Account:           "acct",
					Address:           "1Address",
					BIP125Replaceable: "unknown",
					Category:          "send",
					Amount:            1.5,
					Fee:               btcjson.Float64(0.0001),
					Confirmations:     1,
					TxID:              "456",
					WalletConflicts:   []string{},
					Time:              12345678,
					TimeReceived:      12345876,
					Trusted:           true,
					Vout:              789,
					OtherAccount:      "otheracct",
				},
			},
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Marshal the notification as created by the new static
		// creation function.  The ID is nil for notifications.
		marshaled, err := btcjson.MarshalCmd(nil, test.staticNtfn())
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

		// Ensure the notification is created without error via the
		// generic new notification creation function.
		cmd, err := test.newNtfn()
		if err != nil {
			t.Errorf("Test #%d (%s) unexpected NewCmd error: %v ",
				i, test.name, err)
		}

		// Marshal the notification as created by the generic new
		// notification creation function.    The ID is nil for
		// notifications.
		marshaled, err = btcjson.MarshalCmd(nil, cmd)
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

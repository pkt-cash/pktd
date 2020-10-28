// Copyright (c) 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcjson_test

import (
	"bytes"
	"github.com/json-iterator/go"
	"fmt"
	"reflect"
	"testing"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/btcjson"
)

// TestWalletSvrCmds tests all of the wallet server commands marshal and
// unmarshal into valid results include handling of optional fields being
// omitted in the marshaled command, while optional fields with defaults have
// the default assigned on unmarhsaled commands.
func TestWalletSvrCmds(t *testing.T) {
	t.Parallel()

	testID := int(1)
	tests := []struct {
		name         string
		newCmd       func() (interface{}, er.R)
		staticCmd    func() interface{}
		marshaled   string
		unmarhsaled interface{}
	}{
		{
			name: "addmultisigaddress",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("addmultisigaddress", 2, []string{"031234", "035678"})
			},
			marshaled: `{"jsonrpc":"1.0","method":"addmultisigaddress","params":[2,["031234","035678"]],"id":1}`,
			unmarhsaled: &btcjson.AddMultisigAddressCmd{
				NRequired: 2,
				Keys:      []string{"031234", "035678"},
			},
		},
		{
			name: "addwitnessaddress",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("addwitnessaddress", "1address")
			},
			staticCmd: func() interface{} {
				return btcjson.NewAddWitnessAddressCmd("1address")
			},
			marshaled: `{"jsonrpc":"1.0","method":"addwitnessaddress","params":["1address"],"id":1}`,
			unmarhsaled: &btcjson.AddWitnessAddressCmd{
				Address: "1address",
			},
		},
		{
			name: "createmultisig",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("createmultisig", 2, []string{"031234", "035678"})
			},
			staticCmd: func() interface{} {
				keys := []string{"031234", "035678"}
				return btcjson.NewCreateMultisigCmd(2, keys)
			},
			marshaled: `{"jsonrpc":"1.0","method":"createmultisig","params":[2,["031234","035678"]],"id":1}`,
			unmarhsaled: &btcjson.CreateMultisigCmd{
				NRequired: 2,
				Keys:      []string{"031234", "035678"},
			},
		},
		{
			name: "dumpprivkey",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("dumpprivkey", "1Address")
			},
			staticCmd: func() interface{} {
				return btcjson.NewDumpPrivKeyCmd("1Address")
			},
			marshaled: `{"jsonrpc":"1.0","method":"dumpprivkey","params":["1Address"],"id":1}`,
			unmarhsaled: &btcjson.DumpPrivKeyCmd{
				Address: "1Address",
			},
		},
		{
			name: "estimatefee",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("estimatefee", 6)
			},
			staticCmd: func() interface{} {
				return btcjson.NewEstimateFeeCmd(6)
			},
			marshaled: `{"jsonrpc":"1.0","method":"estimatefee","params":[6],"id":1}`,
			unmarhsaled: &btcjson.EstimateFeeCmd{
				NumBlocks: 6,
			},
		},
		{
			name: "getbalance",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getbalance")
			},
			marshaled: `{"jsonrpc":"1.0","method":"getbalance","params":[],"id":1}`,
			unmarhsaled: &btcjson.GetBalanceCmd{
				MinConf: btcjson.Int(1),
			},
		},
		{
			name: "getnewaddress",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getnewaddress")
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getnewaddress","params":[],"id":1}`,
			unmarhsaled: &btcjson.GetNewAddressCmd{},
		},
		{
			name: "getnewaddress optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getnewaddress", true)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getnewaddress","params":[true],"id":1}`,
			unmarhsaled: &btcjson.GetNewAddressCmd{
				Legacy: func() *bool { x := true; return &x }(),
			},
		},
		{
			name: "getreceivedbyaddress",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getreceivedbyaddress", "1Address")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetReceivedByAddressCmd("1Address", nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getreceivedbyaddress","params":["1Address"],"id":1}`,
			unmarhsaled: &btcjson.GetReceivedByAddressCmd{
				Address: "1Address",
				MinConf: btcjson.Int(1),
			},
		},
		{
			name: "getreceivedbyaddress optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getreceivedbyaddress", "1Address", 6)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetReceivedByAddressCmd("1Address", btcjson.Int(6))
			},
			marshaled: `{"jsonrpc":"1.0","method":"getreceivedbyaddress","params":["1Address",6],"id":1}`,
			unmarhsaled: &btcjson.GetReceivedByAddressCmd{
				Address: "1Address",
				MinConf: btcjson.Int(6),
			},
		},
		{
			name: "gettransaction",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("gettransaction", "123")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetTransactionCmd("123", nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"gettransaction","params":["123"],"id":1}`,
			unmarhsaled: &btcjson.GetTransactionCmd{
				Txid:             "123",
				IncludeWatchOnly: btcjson.Bool(false),
			},
		},
		{
			name: "gettransaction optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("gettransaction", "123", true)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetTransactionCmd("123", btcjson.Bool(true))
			},
			marshaled: `{"jsonrpc":"1.0","method":"gettransaction","params":["123",true],"id":1}`,
			unmarhsaled: &btcjson.GetTransactionCmd{
				Txid:             "123",
				IncludeWatchOnly: btcjson.Bool(true),
			},
		},
		{
			name: "importprivkey",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("importprivkey", "abc")
			},
			staticCmd: func() interface{} {
				return btcjson.NewImportPrivKeyCmd("abc", nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"importprivkey","params":["abc"],"id":1}`,
			unmarhsaled: &btcjson.ImportPrivKeyCmd{
				PrivKey: "abc",
				Label:   nil,
				Rescan:  btcjson.Bool(true),
			},
		},
		{
			name: "importprivkey optional1",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("importprivkey", "abc", "label")
			},
			staticCmd: func() interface{} {
				return btcjson.NewImportPrivKeyCmd("abc", btcjson.String("label"), nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"importprivkey","params":["abc","label"],"id":1}`,
			unmarhsaled: &btcjson.ImportPrivKeyCmd{
				PrivKey: "abc",
				Label:   btcjson.String("label"),
				Rescan:  btcjson.Bool(true),
			},
		},
		{
			name: "importprivkey optional2",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("importprivkey", "abc", "label", false)
			},
			staticCmd: func() interface{} {
				return btcjson.NewImportPrivKeyCmd("abc", btcjson.String("label"), btcjson.Bool(false))
			},
			marshaled: `{"jsonrpc":"1.0","method":"importprivkey","params":["abc","label",false],"id":1}`,
			unmarhsaled: &btcjson.ImportPrivKeyCmd{
				PrivKey: "abc",
				Label:   btcjson.String("label"),
				Rescan:  btcjson.Bool(false),
			},
		},
		{
			name: "listlockunspent",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listlockunspent")
			},
			staticCmd: func() interface{} {
				return btcjson.NewListLockUnspentCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"listlockunspent","params":[],"id":1}`,
			unmarhsaled: &btcjson.ListLockUnspentCmd{},
		},
		{
			name: "listreceivedbyaddress",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listreceivedbyaddress")
			},
			staticCmd: func() interface{} {
				return btcjson.NewListReceivedByAddressCmd(nil, nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"listreceivedbyaddress","params":[],"id":1}`,
			unmarhsaled: &btcjson.ListReceivedByAddressCmd{
				MinConf:          btcjson.Int(1),
				IncludeEmpty:     btcjson.Bool(false),
				IncludeWatchOnly: btcjson.Bool(false),
			},
		},
		{
			name: "listreceivedbyaddress optional1",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listreceivedbyaddress", 6)
			},
			staticCmd: func() interface{} {
				return btcjson.NewListReceivedByAddressCmd(btcjson.Int(6), nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"listreceivedbyaddress","params":[6],"id":1}`,
			unmarhsaled: &btcjson.ListReceivedByAddressCmd{
				MinConf:          btcjson.Int(6),
				IncludeEmpty:     btcjson.Bool(false),
				IncludeWatchOnly: btcjson.Bool(false),
			},
		},
		{
			name: "listreceivedbyaddress optional2",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listreceivedbyaddress", 6, true)
			},
			staticCmd: func() interface{} {
				return btcjson.NewListReceivedByAddressCmd(btcjson.Int(6), btcjson.Bool(true), nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"listreceivedbyaddress","params":[6,true],"id":1}`,
			unmarhsaled: &btcjson.ListReceivedByAddressCmd{
				MinConf:          btcjson.Int(6),
				IncludeEmpty:     btcjson.Bool(true),
				IncludeWatchOnly: btcjson.Bool(false),
			},
		},
		{
			name: "listreceivedbyaddress optional3",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listreceivedbyaddress", 6, true, false)
			},
			staticCmd: func() interface{} {
				return btcjson.NewListReceivedByAddressCmd(btcjson.Int(6), btcjson.Bool(true), btcjson.Bool(false))
			},
			marshaled: `{"jsonrpc":"1.0","method":"listreceivedbyaddress","params":[6,true,false],"id":1}`,
			unmarhsaled: &btcjson.ListReceivedByAddressCmd{
				MinConf:          btcjson.Int(6),
				IncludeEmpty:     btcjson.Bool(true),
				IncludeWatchOnly: btcjson.Bool(false),
			},
		},
		{
			name: "listsinceblock",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listsinceblock")
			},
			staticCmd: func() interface{} {
				return btcjson.NewListSinceBlockCmd(nil, nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"listsinceblock","params":[],"id":1}`,
			unmarhsaled: &btcjson.ListSinceBlockCmd{
				BlockHash:           nil,
				TargetConfirmations: btcjson.Int(1),
				IncludeWatchOnly:    btcjson.Bool(false),
			},
		},
		{
			name: "listsinceblock optional1",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listsinceblock", "123")
			},
			staticCmd: func() interface{} {
				return btcjson.NewListSinceBlockCmd(btcjson.String("123"), nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"listsinceblock","params":["123"],"id":1}`,
			unmarhsaled: &btcjson.ListSinceBlockCmd{
				BlockHash:           btcjson.String("123"),
				TargetConfirmations: btcjson.Int(1),
				IncludeWatchOnly:    btcjson.Bool(false),
			},
		},
		{
			name: "listsinceblock optional2",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listsinceblock", "123", 6)
			},
			staticCmd: func() interface{} {
				return btcjson.NewListSinceBlockCmd(btcjson.String("123"), btcjson.Int(6), nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"listsinceblock","params":["123",6],"id":1}`,
			unmarhsaled: &btcjson.ListSinceBlockCmd{
				BlockHash:           btcjson.String("123"),
				TargetConfirmations: btcjson.Int(6),
				IncludeWatchOnly:    btcjson.Bool(false),
			},
		},
		{
			name: "listsinceblock optional3",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listsinceblock", "123", 6, true)
			},
			staticCmd: func() interface{} {
				return btcjson.NewListSinceBlockCmd(btcjson.String("123"), btcjson.Int(6), btcjson.Bool(true))
			},
			marshaled: `{"jsonrpc":"1.0","method":"listsinceblock","params":["123",6,true],"id":1}`,
			unmarhsaled: &btcjson.ListSinceBlockCmd{
				BlockHash:           btcjson.String("123"),
				TargetConfirmations: btcjson.Int(6),
				IncludeWatchOnly:    btcjson.Bool(true),
			},
		},
		{
			name: "listtransactions",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listtransactions")
			},
			marshaled: `{"jsonrpc":"1.0","method":"listtransactions","params":[],"id":1}`,
			unmarhsaled: &btcjson.ListTransactionsCmd{
				Count: btcjson.Int(10),
				From:  btcjson.Int(0),
			},
		},
		{
			name: "listtransactions optional3",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listtransactions", 20, 1)
			},
			marshaled: `{"jsonrpc":"1.0","method":"listtransactions","params":[20,1],"id":1}`,
			unmarhsaled: &btcjson.ListTransactionsCmd{
				Count: btcjson.Int(20),
				From:  btcjson.Int(1),
			},
		},
		{
			name: "listunspent",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listunspent")
			},
			staticCmd: func() interface{} {
				return btcjson.NewListUnspentCmd(nil, nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"listunspent","params":[],"id":1}`,
			unmarhsaled: &btcjson.ListUnspentCmd{
				MinConf:   btcjson.Int(1),
				MaxConf:   btcjson.Int(9999999),
				Addresses: nil,
			},
		},
		{
			name: "listunspent optional1",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listunspent", 6)
			},
			staticCmd: func() interface{} {
				return btcjson.NewListUnspentCmd(btcjson.Int(6), nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"listunspent","params":[6],"id":1}`,
			unmarhsaled: &btcjson.ListUnspentCmd{
				MinConf:   btcjson.Int(6),
				MaxConf:   btcjson.Int(9999999),
				Addresses: nil,
			},
		},
		{
			name: "listunspent optional2",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listunspent", 6, 100)
			},
			staticCmd: func() interface{} {
				return btcjson.NewListUnspentCmd(btcjson.Int(6), btcjson.Int(100), nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"listunspent","params":[6,100],"id":1}`,
			unmarhsaled: &btcjson.ListUnspentCmd{
				MinConf:   btcjson.Int(6),
				MaxConf:   btcjson.Int(100),
				Addresses: nil,
			},
		},
		{
			name: "listunspent optional3",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("listunspent", 6, 100, []string{"1Address", "1Address2"})
			},
			staticCmd: func() interface{} {
				return btcjson.NewListUnspentCmd(btcjson.Int(6), btcjson.Int(100),
					&[]string{"1Address", "1Address2"})
			},
			marshaled: `{"jsonrpc":"1.0","method":"listunspent","params":[6,100,["1Address","1Address2"]],"id":1}`,
			unmarhsaled: &btcjson.ListUnspentCmd{
				MinConf:   btcjson.Int(6),
				MaxConf:   btcjson.Int(100),
				Addresses: &[]string{"1Address", "1Address2"},
			},
		},
		{
			name: "lockunspent",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("lockunspent", true, `[{"txid":"123","vout":1}]`)
			},
			staticCmd: func() interface{} {
				txInputs := []btcjson.TransactionInput{
					{Txid: "123", Vout: 1},
				}
				return btcjson.NewLockUnspentCmd(true, txInputs)
			},
			marshaled: `{"jsonrpc":"1.0","method":"lockunspent","params":[true,[{"txid":"123","vout":1}]],"id":1}`,
			unmarhsaled: &btcjson.LockUnspentCmd{
				Unlock: true,
				Transactions: []btcjson.TransactionInput{
					{Txid: "123", Vout: 1},
				},
			},
		},
		{
			name: "sendfrom",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("sendfrom", "1Address", 0.5, &[]string{"from"})
			},
			staticCmd: func() interface{} {
				return btcjson.NewSendFromCmd(&[]string{"from"}, "1Address", 0.5, nil, nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"sendfrom","params":["1Address",0.5,["from"]],"id":1}`,
			unmarhsaled: &btcjson.SendFromCmd{
				ToAddress:     "1Address",
				Amount:        0.5,
				FromAddresses: &[]string{"from"},
				MinConf:       btcjson.Int(1),
				Comment:       nil,
				CommentTo:     nil,
			},
		},
		{
			name: "sendfrom optional1",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("sendfrom", "1Address", 0.5, &[]string{"from"}, 6)
			},
			staticCmd: func() interface{} {
				return btcjson.NewSendFromCmd(&[]string{"from"}, "1Address", 0.5, btcjson.Int(6), nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"sendfrom","params":["1Address",0.5,["from"],6],"id":1}`,
			unmarhsaled: &btcjson.SendFromCmd{
				FromAddresses: &[]string{"from"},
				ToAddress:     "1Address",
				Amount:        0.5,
				MinConf:       btcjson.Int(6),
				Comment:       nil,
				CommentTo:     nil,
			},
		},
		{
			name: "sendfrom optional2",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("sendfrom", "1Address", 0.5, &[]string{"from"}, 6, "comment")
			},
			staticCmd: func() interface{} {
				return btcjson.NewSendFromCmd(&[]string{"from"}, "1Address", 0.5, btcjson.Int(6),
					btcjson.String("comment"), nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"sendfrom","params":["1Address",0.5,["from"],6,"comment"],"id":1}`,
			unmarhsaled: &btcjson.SendFromCmd{
				FromAddresses: &[]string{"from"},
				ToAddress:     "1Address",
				Amount:        0.5,
				MinConf:       btcjson.Int(6),
				Comment:       btcjson.String("comment"),
				CommentTo:     nil,
			},
		},
		{
			name: "sendfrom optional3",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("sendfrom", "1Address", 0.5, &[]string{"from"}, 6, "comment", "commentto")
			},
			staticCmd: func() interface{} {
				return btcjson.NewSendFromCmd(&[]string{"from"}, "1Address", 0.5, btcjson.Int(6),
					btcjson.String("comment"), btcjson.String("commentto"))
			},
			marshaled: `{"jsonrpc":"1.0","method":"sendfrom","params":["1Address",0.5,["from"],6,"comment","commentto"],"id":1}`,
			unmarhsaled: &btcjson.SendFromCmd{
				FromAddresses: &[]string{"from"},
				ToAddress:     "1Address",
				Amount:        0.5,
				MinConf:       btcjson.Int(6),
				Comment:       btcjson.String("comment"),
				CommentTo:     btcjson.String("commentto"),
			},
		},
		{
			name: "sendmany",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("sendmany", `{"1Address":0.5}`, &[]string{"from"})
			},
			staticCmd: func() interface{} {
				amounts := map[string]float64{"1Address": 0.5}
				return btcjson.NewSendManyCmd(&[]string{"from"}, amounts, nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"sendmany","params":[{"1Address":0.5},["from"]],"id":1}`,
			unmarhsaled: &btcjson.SendManyCmd{
				FromAddresses: &[]string{"from"},
				Amounts:       map[string]float64{"1Address": 0.5},
				MinConf:       btcjson.Int(1),
				Comment:       nil,
			},
		},
		{
			name: "sendmany optional1",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("sendmany", `{"1Address":0.5}`, &[]string{"from"}, 6)
			},
			staticCmd: func() interface{} {
				amounts := map[string]float64{"1Address": 0.5}
				return btcjson.NewSendManyCmd(&[]string{"from"}, amounts, btcjson.Int(6), nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"sendmany","params":[{"1Address":0.5},["from"],6],"id":1}`,
			unmarhsaled: &btcjson.SendManyCmd{
				FromAddresses: &[]string{"from"},
				Amounts:       map[string]float64{"1Address": 0.5},
				MinConf:       btcjson.Int(6),
				Comment:       nil,
			},
		},
		{
			name: "sendmany optional2",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("sendmany", `{"1Address":0.5}`, &[]string{"from"}, 6, "comment")
			},
			staticCmd: func() interface{} {
				amounts := map[string]float64{"1Address": 0.5}
				return btcjson.NewSendManyCmd(&[]string{"from"}, amounts, btcjson.Int(6), btcjson.String("comment"))
			},
			marshaled: `{"jsonrpc":"1.0","method":"sendmany","params":[{"1Address":0.5},["from"],6,"comment"],"id":1}`,
			unmarhsaled: &btcjson.SendManyCmd{
				FromAddresses: &[]string{"from"},
				Amounts:       map[string]float64{"1Address": 0.5},
				MinConf:       btcjson.Int(6),
				Comment:       btcjson.String("comment"),
			},
		},
		{
			name: "sendtoaddress",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("sendtoaddress", "1Address", 0.5)
			},
			staticCmd: func() interface{} {
				return btcjson.NewSendToAddressCmd("1Address", 0.5, nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"sendtoaddress","params":["1Address",0.5],"id":1}`,
			unmarhsaled: &btcjson.SendToAddressCmd{
				Address:   "1Address",
				Amount:    0.5,
				Comment:   nil,
				CommentTo: nil,
			},
		},
		{
			name: "sendtoaddress optional1",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("sendtoaddress", "1Address", 0.5, "comment", "commentto")
			},
			staticCmd: func() interface{} {
				return btcjson.NewSendToAddressCmd("1Address", 0.5, btcjson.String("comment"),
					btcjson.String("commentto"))
			},
			marshaled: `{"jsonrpc":"1.0","method":"sendtoaddress","params":["1Address",0.5,"comment","commentto"],"id":1}`,
			unmarhsaled: &btcjson.SendToAddressCmd{
				Address:   "1Address",
				Amount:    0.5,
				Comment:   btcjson.String("comment"),
				CommentTo: btcjson.String("commentto"),
			},
		},
		{
			name: "settxfee",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("settxfee", 0.0001)
			},
			staticCmd: func() interface{} {
				return btcjson.NewSetTxFeeCmd(0.0001)
			},
			marshaled: `{"jsonrpc":"1.0","method":"settxfee","params":[0.0001],"id":1}`,
			unmarhsaled: &btcjson.SetTxFeeCmd{
				Amount: 0.0001,
			},
		},
		{
			name: "signmessage",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("signmessage", "1Address", "message")
			},
			staticCmd: func() interface{} {
				return btcjson.NewSignMessageCmd("1Address", "message")
			},
			marshaled: `{"jsonrpc":"1.0","method":"signmessage","params":["1Address","message"],"id":1}`,
			unmarhsaled: &btcjson.SignMessageCmd{
				Address: "1Address",
				Message: "message",
			},
		},
		{
			name: "signrawtransaction",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("signrawtransaction", "001122")
			},
			staticCmd: func() interface{} {
				return btcjson.NewSignRawTransactionCmd("001122", nil, nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"signrawtransaction","params":["001122"],"id":1}`,
			unmarhsaled: &btcjson.SignRawTransactionCmd{
				RawTx:    "001122",
				Inputs:   nil,
				PrivKeys: nil,
				Flags:    btcjson.String("ALL"),
			},
		},
		{
			name: "signrawtransaction optional1",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("signrawtransaction", "001122", `[{"txid":"123","vout":1,"scriptPubKey":"00","redeemScript":"01"}]`)
			},
			staticCmd: func() interface{} {
				txInputs := []btcjson.RawTxInput{
					{
						Txid:         "123",
						Vout:         1,
						ScriptPubKey: "00",
						RedeemScript: "01",
					},
				}

				return btcjson.NewSignRawTransactionCmd("001122", &txInputs, nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"signrawtransaction","params":["001122",[{"txid":"123","vout":1,"scriptPubKey":"00","redeemScript":"01"}]],"id":1}`,
			unmarhsaled: &btcjson.SignRawTransactionCmd{
				RawTx: "001122",
				Inputs: &[]btcjson.RawTxInput{
					{
						Txid:         "123",
						Vout:         1,
						ScriptPubKey: "00",
						RedeemScript: "01",
					},
				},
				PrivKeys: nil,
				Flags:    btcjson.String("ALL"),
			},
		},
		{
			name: "signrawtransaction optional2",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("signrawtransaction", "001122", `[]`, `["abc"]`)
			},
			staticCmd: func() interface{} {
				txInputs := []btcjson.RawTxInput{}
				privKeys := []string{"abc"}
				return btcjson.NewSignRawTransactionCmd("001122", &txInputs, &privKeys, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"signrawtransaction","params":["001122",[],["abc"]],"id":1}`,
			unmarhsaled: &btcjson.SignRawTransactionCmd{
				RawTx:    "001122",
				Inputs:   &[]btcjson.RawTxInput{},
				PrivKeys: &[]string{"abc"},
				Flags:    btcjson.String("ALL"),
			},
		},
		{
			name: "signrawtransaction optional3",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("signrawtransaction", "001122", `[]`, `[]`, "ALL")
			},
			staticCmd: func() interface{} {
				txInputs := []btcjson.RawTxInput{}
				privKeys := []string{}
				return btcjson.NewSignRawTransactionCmd("001122", &txInputs, &privKeys,
					btcjson.String("ALL"))
			},
			marshaled: `{"jsonrpc":"1.0","method":"signrawtransaction","params":["001122",[],[],"ALL"],"id":1}`,
			unmarhsaled: &btcjson.SignRawTransactionCmd{
				RawTx:    "001122",
				Inputs:   &[]btcjson.RawTxInput{},
				PrivKeys: &[]string{},
				Flags:    btcjson.String("ALL"),
			},
		},
		{
			name: "walletlock",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("walletlock")
			},
			staticCmd: func() interface{} {
				return btcjson.NewWalletLockCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"walletlock","params":[],"id":1}`,
			unmarhsaled: &btcjson.WalletLockCmd{},
		},
		{
			name: "walletpassphrase",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("walletpassphrase", "pass", 60)
			},
			staticCmd: func() interface{} {
				return btcjson.NewWalletPassphraseCmd("pass", 60)
			},
			marshaled: `{"jsonrpc":"1.0","method":"walletpassphrase","params":["pass",60],"id":1}`,
			unmarhsaled: &btcjson.WalletPassphraseCmd{
				Passphrase: "pass",
				Timeout:    60,
			},
		},
		{
			name: "walletpassphrasechange",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("walletpassphrasechange", "old", "new")
			},
			staticCmd: func() interface{} {
				return btcjson.NewWalletPassphraseChangeCmd("old", "new")
			},
			marshaled: `{"jsonrpc":"1.0","method":"walletpassphrasechange","params":["old","new"],"id":1}`,
			unmarhsaled: &btcjson.WalletPassphraseChangeCmd{
				OldPassphrase: "old",
				NewPassphrase: "new",
			},
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Marshal the command as created by the new static command
		// creation function.
		if test.staticCmd != nil {
			marshaled, err := btcjson.MarshalCmd(testID, test.staticCmd())
			if err != nil {
				t.Errorf("MarshalCmd #%d (%s) unexpected error: %v", i,
					test.name, err)
				continue
			}

			if !bytes.Equal(marshaled, []byte(test.marhsaled)) {
				t.Errorf("Test #%d (%s) unexpected marshaled data - "+
					"got %s, want %s", i, test.name, marshaled,
					test.marhsaled)
				continue
			}
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
		marshaled, err := btcjson.MarshalCmd(testID, cmd)
		if err != nil {
			t.Errorf("MarshalCmd #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !bytes.Equal(marshaled, []byte(test.marhsaled)) {
			t.Errorf("Test #%d (%s) unexpected marshaled data - "+
				"got %s, want %s", i, test.name, marshaled,
				test.marhsaled)
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

		if !reflect.DeepEqual(cmd, test.unmarhsaled) {
			t.Errorf("Test #%d (%s) unexpected unmarhsaled command "+
				"- got %s, want %s", i, test.name,
				fmt.Sprintf("(%T) %+[1]v", cmd),
				fmt.Sprintf("(%T) %+[1]v\n", test.unmarhsaled))
			continue
		}
	}
}

// Copyright (c) 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcjson_test

import (
	"bytes"
	/*"encoding/json" XXX -trn */
	"github.com/json-iterator/go"
	"fmt"
	"reflect"
	"testing"

	"github.com/pkt-cash/pktd/btcjson"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/wire"
)

// TestChainSvrCmds tests all of the chain server commands marshal and unmarshal
// into valid results include handling of optional fields being omitted in the
// marshaled command, while optional fields with defaults have the default
// assigned on unmarshaled commands.
func TestChainSvrCmds(t *testing.T) {
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
			name: "addnode",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("addnode", "127.0.0.1", btcjson.ANRemove)
			},
			staticCmd: func() interface{} {
				return btcjson.NewAddNodeCmd("127.0.0.1", btcjson.ANRemove)
			},
			marshaled:   `{"jsonrpc":"1.0","method":"addnode","params":["127.0.0.1","remove"],"id":1}`,
			unmarshaled: &btcjson.AddNodeCmd{Addr: "127.0.0.1", SubCmd: btcjson.ANRemove},
		},
		{
			name: "createrawtransaction",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("createrawtransaction", `[{"txid":"123","vout":1}]`,
					`{"456":0.0123}`)
			},
			staticCmd: func() interface{} {
				txInputs := []btcjson.TransactionInput{
					{Txid: "123", Vout: 1},
				}
				amounts := map[string]float64{"456": .0123}
				return btcjson.NewCreateRawTransactionCmd(txInputs, amounts, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"createrawtransaction","params":[[{"txid":"123","vout":1}],{"456":0.0123}],"id":1}`,
			unmarshaled: &btcjson.CreateRawTransactionCmd{
				Inputs:  []btcjson.TransactionInput{{Txid: "123", Vout: 1}},
				Amounts: map[string]float64{"456": .0123},
			},
		},
		{
			name: "createrawtransaction optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("createrawtransaction", `[{"txid":"123","vout":1}]`,
					`{"456":0.0123}`, int64(12312333333))
			},
			staticCmd: func() interface{} {
				txInputs := []btcjson.TransactionInput{
					{Txid: "123", Vout: 1},
				}
				amounts := map[string]float64{"456": .0123}
				return btcjson.NewCreateRawTransactionCmd(txInputs, amounts, btcjson.Int64(12312333333))
			},
			marshaled: `{"jsonrpc":"1.0","method":"createrawtransaction","params":[[{"txid":"123","vout":1}],{"456":0.0123},12312333333],"id":1}`,
			unmarshaled: &btcjson.CreateRawTransactionCmd{
				Inputs:   []btcjson.TransactionInput{{Txid: "123", Vout: 1}},
				Amounts:  map[string]float64{"456": .0123},
				LockTime: btcjson.Int64(12312333333),
			},
		},

		{
			name: "decoderawtransaction",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("decoderawtransaction", "123")
			},
			staticCmd: func() interface{} {
				return btcjson.NewDecodeRawTransactionCmd("123")
			},
			marshaled:   `{"jsonrpc":"1.0","method":"decoderawtransaction","params":["123"],"id":1}`,
			unmarshaled: &btcjson.DecodeRawTransactionCmd{HexTx: "123"},
		},
		{
			name: "decodescript",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("decodescript", "00")
			},
			staticCmd: func() interface{} {
				return btcjson.NewDecodeScriptCmd("00")
			},
			marshaled:   `{"jsonrpc":"1.0","method":"decodescript","params":["00"],"id":1}`,
			unmarshaled: &btcjson.DecodeScriptCmd{HexScript: "00"},
		},
		{
			name: "getaddednodeinfo",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getaddednodeinfo", true)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetAddedNodeInfoCmd(true, nil)
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getaddednodeinfo","params":[true],"id":1}`,
			unmarshaled: &btcjson.GetAddedNodeInfoCmd{DNS: true, Node: nil},
		},
		{
			name: "getaddednodeinfo optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getaddednodeinfo", true, "127.0.0.1")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetAddedNodeInfoCmd(true, btcjson.String("127.0.0.1"))
			},
			marshaled: `{"jsonrpc":"1.0","method":"getaddednodeinfo","params":[true,"127.0.0.1"],"id":1}`,
			unmarshaled: &btcjson.GetAddedNodeInfoCmd{
				DNS:  true,
				Node: btcjson.String("127.0.0.1"),
			},
		},
		{
			name: "getbestblockhash",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getbestblockhash")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetBestBlockHashCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getbestblockhash","params":[],"id":1}`,
			unmarshaled: &btcjson.GetBestBlockHashCmd{},
		},
		{
			name: "getblock",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getblock", "123")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetBlockCmd("123", nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getblock","params":["123"],"id":1}`,
			unmarshaled: &btcjson.GetBlockCmd{
				Hash:       "123",
				Verbose:    btcjson.Bool(true),
				VerboseTx:  btcjson.Bool(false),
				VerbosePcp: btcjson.Bool(false),
			},
		},
		{
			name: "getblock required optional1",
			newCmd: func() (interface{}, er.R) {
				// Intentionally use a source param that is
				// more pointers than the destination to
				// exercise that path.
				verbosePtr := btcjson.Bool(true)
				return btcjson.NewCmd("getblock", "123", &verbosePtr)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetBlockCmd("123", btcjson.Bool(true), nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getblock","params":["123",true],"id":1}`,
			unmarshaled: &btcjson.GetBlockCmd{
				Hash:       "123",
				Verbose:    btcjson.Bool(true),
				VerboseTx:  btcjson.Bool(false),
				VerbosePcp: btcjson.Bool(false),
			},
		},
		{
			name: "getblock required optional2",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getblock", "123", true, true)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetBlockCmd("123", btcjson.Bool(true), btcjson.Bool(true))
			},
			marshaled: `{"jsonrpc":"1.0","method":"getblock","params":["123",true,true],"id":1}`,
			unmarshaled: &btcjson.GetBlockCmd{
				Hash:       "123",
				Verbose:    btcjson.Bool(true),
				VerboseTx:  btcjson.Bool(true),
				VerbosePcp: btcjson.Bool(false),
			},
		},
		{
			name: "getblockchaininfo",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getblockchaininfo")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetBlockChainInfoCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getblockchaininfo","params":[],"id":1}`,
			unmarshaled: &btcjson.GetBlockChainInfoCmd{},
		},
		{
			name: "getblockcount",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getblockcount")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetBlockCountCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getblockcount","params":[],"id":1}`,
			unmarshaled: &btcjson.GetBlockCountCmd{},
		},
		{
			name: "getblockhash",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getblockhash", 123)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetBlockHashCmd(123)
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getblockhash","params":[123],"id":1}`,
			unmarshaled: &btcjson.GetBlockHashCmd{Index: 123},
		},
		{
			name: "getblockheader",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getblockheader", "123")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetBlockHeaderCmd("123", nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getblockheader","params":["123"],"id":1}`,
			unmarshaled: &btcjson.GetBlockHeaderCmd{
				Hash:    "123",
				Verbose: btcjson.Bool(true),
			},
		},
		{
			name: "getblocktemplate",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getblocktemplate")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetBlockTemplateCmd(nil)
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getblocktemplate","params":[],"id":1}`,
			unmarshaled: &btcjson.GetBlockTemplateCmd{Request: nil},
		},
		{
			name: "getblocktemplate optional - template request",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getblocktemplate", `{"mode":"template","capabilities":["longpoll","coinbasetxn"]}`)
			},
			staticCmd: func() interface{} {
				template := btcjson.TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
				}
				return btcjson.NewGetBlockTemplateCmd(&template)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getblocktemplate","params":[{"mode":"template","capabilities":["longpoll","coinbasetxn"]}],"id":1}`,
			unmarshaled: &btcjson.GetBlockTemplateCmd{
				Request: &btcjson.TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
				},
			},
		},
		{
			name: "getblocktemplate optional - template request with tweaks",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getblocktemplate", `{"mode":"template","capabilities":["longpoll","coinbasetxn"],"sigoplimit":500,"sizelimit":100000000,"maxversion":2}`)
			},
			staticCmd: func() interface{} {
				template := btcjson.TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
					SigOpLimit:   500,
					SizeLimit:    100000000,
					MaxVersion:   2,
				}
				return btcjson.NewGetBlockTemplateCmd(&template)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getblocktemplate","params":[{"mode":"template","capabilities":["longpoll","coinbasetxn"],"sigoplimit":500,"sizelimit":100000000,"maxversion":2}],"id":1}`,
			unmarshaled: &btcjson.GetBlockTemplateCmd{
				Request: &btcjson.TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
					SigOpLimit:   int64(500),
					SizeLimit:    int64(100000000),
					MaxVersion:   2,
				},
			},
		},
		{
			name: "getblocktemplate optional - template request with tweaks 2",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getblocktemplate", `{"mode":"template","capabilities":["longpoll","coinbasetxn"],"sigoplimit":true,"sizelimit":100000000,"maxversion":2}`)
			},
			staticCmd: func() interface{} {
				template := btcjson.TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
					SigOpLimit:   true,
					SizeLimit:    100000000,
					MaxVersion:   2,
				}
				return btcjson.NewGetBlockTemplateCmd(&template)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getblocktemplate","params":[{"mode":"template","capabilities":["longpoll","coinbasetxn"],"sigoplimit":true,"sizelimit":100000000,"maxversion":2}],"id":1}`,
			unmarshaled: &btcjson.GetBlockTemplateCmd{
				Request: &btcjson.TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
					SigOpLimit:   true,
					SizeLimit:    int64(100000000),
					MaxVersion:   2,
				},
			},
		},
		{
			name: "getcfilter",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getcfilter", "123",
					wire.GCSFilterRegular)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetCFilterCmd("123",
					wire.GCSFilterRegular)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getcfilter","params":["123",0],"id":1}`,
			unmarshaled: &btcjson.GetCFilterCmd{
				Hash:       "123",
				FilterType: wire.GCSFilterRegular,
			},
		},
		{
			name: "getcfilterheader",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getcfilterheader", "123",
					wire.GCSFilterRegular)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetCFilterHeaderCmd("123",
					wire.GCSFilterRegular)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getcfilterheader","params":["123",0],"id":1}`,
			unmarshaled: &btcjson.GetCFilterHeaderCmd{
				Hash:       "123",
				FilterType: wire.GCSFilterRegular,
			},
		},
		{
			name: "getchaintips",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getchaintips")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetChainTipsCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getchaintips","params":[],"id":1}`,
			unmarshaled: &btcjson.GetChainTipsCmd{},
		},
		{
			name: "getconnectioncount",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getconnectioncount")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetConnectionCountCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getconnectioncount","params":[],"id":1}`,
			unmarshaled: &btcjson.GetConnectionCountCmd{},
		},
		{
			name: "getdifficulty",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getdifficulty")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetDifficultyCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getdifficulty","params":[],"id":1}`,
			unmarshaled: &btcjson.GetDifficultyCmd{},
		},
		{
			name: "getgenerate",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getgenerate")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetGenerateCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getgenerate","params":[],"id":1}`,
			unmarshaled: &btcjson.GetGenerateCmd{},
		},
		{
			name: "gethashespersec",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("gethashespersec")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetHashesPerSecCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"gethashespersec","params":[],"id":1}`,
			unmarshaled: &btcjson.GetHashesPerSecCmd{},
		},
		{
			name: "getinfo",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getinfo")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetInfoCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getinfo","params":[],"id":1}`,
			unmarshaled: &btcjson.GetInfoCmd{},
		},
		{
			name: "getmempoolentry",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getmempoolentry", "txhash")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetMempoolEntryCmd("txhash")
			},
			marshaled: `{"jsonrpc":"1.0","method":"getmempoolentry","params":["txhash"],"id":1}`,
			unmarshaled: &btcjson.GetMempoolEntryCmd{
				TxID: "txhash",
			},
		},
		{
			name: "getmempoolinfo",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getmempoolinfo")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetMempoolInfoCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getmempoolinfo","params":[],"id":1}`,
			unmarshaled: &btcjson.GetMempoolInfoCmd{},
		},
		{
			name: "getmininginfo",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getmininginfo")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetMiningInfoCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getmininginfo","params":[],"id":1}`,
			unmarshaled: &btcjson.GetMiningInfoCmd{},
		},
		{
			name: "getnetworkinfo",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getnetworkinfo")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetNetworkInfoCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getnetworkinfo","params":[],"id":1}`,
			unmarshaled: &btcjson.GetNetworkInfoCmd{},
		},
		{
			name: "getnettotals",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getnettotals")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetNetTotalsCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getnettotals","params":[],"id":1}`,
			unmarshaled: &btcjson.GetNetTotalsCmd{},
		},
		{
			name: "getnetworkhashps",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getnetworkhashps")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetNetworkHashPSCmd(nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getnetworkhashps","params":[],"id":1}`,
			unmarshaled: &btcjson.GetNetworkHashPSCmd{
				Blocks: btcjson.Int(120),
				Height: btcjson.Int(-1),
			},
		},
		{
			name: "getnetworkhashps optional1",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getnetworkhashps", 200)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetNetworkHashPSCmd(btcjson.Int(200), nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getnetworkhashps","params":[200],"id":1}`,
			unmarshaled: &btcjson.GetNetworkHashPSCmd{
				Blocks: btcjson.Int(200),
				Height: btcjson.Int(-1),
			},
		},
		{
			name: "getnetworkhashps optional2",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getnetworkhashps", 200, 123)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetNetworkHashPSCmd(btcjson.Int(200), btcjson.Int(123))
			},
			marshaled: `{"jsonrpc":"1.0","method":"getnetworkhashps","params":[200,123],"id":1}`,
			unmarshaled: &btcjson.GetNetworkHashPSCmd{
				Blocks: btcjson.Int(200),
				Height: btcjson.Int(123),
			},
		},
		{
			name: "getpeerinfo",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getpeerinfo")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetPeerInfoCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"getpeerinfo","params":[],"id":1}`,
			unmarshaled: &btcjson.GetPeerInfoCmd{},
		},
		{
			name: "getrawmempool",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getrawmempool")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetRawMempoolCmd(nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getrawmempool","params":[],"id":1}`,
			unmarshaled: &btcjson.GetRawMempoolCmd{
				Verbose: btcjson.Bool(false),
			},
		},
		{
			name: "getrawmempool optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getrawmempool", false)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetRawMempoolCmd(btcjson.Bool(false))
			},
			marshaled: `{"jsonrpc":"1.0","method":"getrawmempool","params":[false],"id":1}`,
			unmarshaled: &btcjson.GetRawMempoolCmd{
				Verbose: btcjson.Bool(false),
			},
		},
		{
			name: "getrawtransaction",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getrawtransaction", "123")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetRawTransactionCmd("123", nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getrawtransaction","params":["123"],"id":1}`,
			unmarshaled: &btcjson.GetRawTransactionCmd{
				Txid:    "123",
				Verbose: btcjson.Bool(false),
			},
		},
		{
			name: "getrawtransaction optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getrawtransaction", "123", true)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetRawTransactionCmd("123", btcjson.Bool(true))
			},
			marshaled: `{"jsonrpc":"1.0","method":"getrawtransaction","params":["123",true],"id":1}`,
			unmarshaled: &btcjson.GetRawTransactionCmd{
				Txid:    "123",
				Verbose: btcjson.Bool(true),
			},
		},
		{
			name: "gettxout",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("gettxout", "123", 1)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetTxOutCmd("123", 1, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"gettxout","params":["123",1],"id":1}`,
			unmarshaled: &btcjson.GetTxOutCmd{
				Txid:           "123",
				Vout:           1,
				IncludeMempool: btcjson.Bool(true),
			},
		},
		{
			name: "gettxout optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("gettxout", "123", 1, true)
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetTxOutCmd("123", 1, btcjson.Bool(true))
			},
			marshaled: `{"jsonrpc":"1.0","method":"gettxout","params":["123",1,true],"id":1}`,
			unmarshaled: &btcjson.GetTxOutCmd{
				Txid:           "123",
				Vout:           1,
				IncludeMempool: btcjson.Bool(true),
			},
		},
		{
			name: "gettxoutproof",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("gettxoutproof", []string{"123", "456"})
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetTxOutProofCmd([]string{"123", "456"}, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"gettxoutproof","params":[["123","456"]],"id":1}`,
			unmarshaled: &btcjson.GetTxOutProofCmd{
				TxIDs: []string{"123", "456"},
			},
		},
		{
			name: "gettxoutproof optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("gettxoutproof", []string{"123", "456"},
					btcjson.String("000000000000034a7dedef4a161fa058a2d67a173a90155f3a2fe6fc132e0ebf"))
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetTxOutProofCmd([]string{"123", "456"},
					btcjson.String("000000000000034a7dedef4a161fa058a2d67a173a90155f3a2fe6fc132e0ebf"))
			},
			marshaled: `{"jsonrpc":"1.0","method":"gettxoutproof","params":[["123","456"],` +
				`"000000000000034a7dedef4a161fa058a2d67a173a90155f3a2fe6fc132e0ebf"],"id":1}`,
			unmarshaled: &btcjson.GetTxOutProofCmd{
				TxIDs:     []string{"123", "456"},
				BlockHash: btcjson.String("000000000000034a7dedef4a161fa058a2d67a173a90155f3a2fe6fc132e0ebf"),
			},
		},
		{
			name: "gettxoutsetinfo",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("gettxoutsetinfo")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetTxOutSetInfoCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"gettxoutsetinfo","params":[],"id":1}`,
			unmarshaled: &btcjson.GetTxOutSetInfoCmd{},
		},
		{
			name: "getwork",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getwork")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetWorkCmd(nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"getwork","params":[],"id":1}`,
			unmarshaled: &btcjson.GetWorkCmd{
				Data: nil,
			},
		},
		{
			name: "getwork optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("getwork", "00112233")
			},
			staticCmd: func() interface{} {
				return btcjson.NewGetWorkCmd(btcjson.String("00112233"))
			},
			marshaled: `{"jsonrpc":"1.0","method":"getwork","params":["00112233"],"id":1}`,
			unmarshaled: &btcjson.GetWorkCmd{
				Data: btcjson.String("00112233"),
			},
		},
		{
			name: "help",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("help")
			},
			staticCmd: func() interface{} {
				return btcjson.NewHelpCmd(nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"help","params":[],"id":1}`,
			unmarshaled: &btcjson.HelpCmd{
				Command: nil,
			},
		},
		{
			name: "help optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("help", "getblock")
			},
			staticCmd: func() interface{} {
				return btcjson.NewHelpCmd(btcjson.String("getblock"))
			},
			marshaled: `{"jsonrpc":"1.0","method":"help","params":["getblock"],"id":1}`,
			unmarshaled: &btcjson.HelpCmd{
				Command: btcjson.String("getblock"),
			},
		},
		{
			name: "invalidateblock",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("invalidateblock", "123")
			},
			staticCmd: func() interface{} {
				return btcjson.NewInvalidateBlockCmd("123")
			},
			marshaled: `{"jsonrpc":"1.0","method":"invalidateblock","params":["123"],"id":1}`,
			unmarshaled: &btcjson.InvalidateBlockCmd{
				BlockHash: "123",
			},
		},
		{
			name: "ping",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("ping")
			},
			staticCmd: func() interface{} {
				return btcjson.NewPingCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"ping","params":[],"id":1}`,
			unmarshaled: &btcjson.PingCmd{},
		},
		{
			name: "preciousblock",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("preciousblock", "0123")
			},
			staticCmd: func() interface{} {
				return btcjson.NewPreciousBlockCmd("0123")
			},
			marshaled: `{"jsonrpc":"1.0","method":"preciousblock","params":["0123"],"id":1}`,
			unmarshaled: &btcjson.PreciousBlockCmd{
				BlockHash: "0123",
			},
		},
		{
			name: "reconsiderblock",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("reconsiderblock", "123")
			},
			staticCmd: func() interface{} {
				return btcjson.NewReconsiderBlockCmd("123")
			},
			marshaled: `{"jsonrpc":"1.0","method":"reconsiderblock","params":["123"],"id":1}`,
			unmarshaled: &btcjson.ReconsiderBlockCmd{
				BlockHash: "123",
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("searchrawtransactions", "1Address")
			},
			staticCmd: func() interface{} {
				return btcjson.NewSearchRawTransactionsCmd("1Address", nil, nil, nil, nil, nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address"],"id":1}`,
			unmarshaled: &btcjson.SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     btcjson.Int(1),
				Skip:        btcjson.Int(0),
				Count:       btcjson.Int(100),
				VinExtra:    btcjson.Int(0),
				Reverse:     btcjson.Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("searchrawtransactions", "1Address", 0)
			},
			staticCmd: func() interface{} {
				return btcjson.NewSearchRawTransactionsCmd("1Address",
					btcjson.Int(0), nil, nil, nil, nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0],"id":1}`,
			unmarshaled: &btcjson.SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     btcjson.Int(0),
				Skip:        btcjson.Int(0),
				Count:       btcjson.Int(100),
				VinExtra:    btcjson.Int(0),
				Reverse:     btcjson.Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("searchrawtransactions", "1Address", 0, 5)
			},
			staticCmd: func() interface{} {
				return btcjson.NewSearchRawTransactionsCmd("1Address",
					btcjson.Int(0), btcjson.Int(5), nil, nil, nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5],"id":1}`,
			unmarshaled: &btcjson.SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     btcjson.Int(0),
				Skip:        btcjson.Int(5),
				Count:       btcjson.Int(100),
				VinExtra:    btcjson.Int(0),
				Reverse:     btcjson.Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("searchrawtransactions", "1Address", 0, 5, 10)
			},
			staticCmd: func() interface{} {
				return btcjson.NewSearchRawTransactionsCmd("1Address",
					btcjson.Int(0), btcjson.Int(5), btcjson.Int(10), nil, nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5,10],"id":1}`,
			unmarshaled: &btcjson.SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     btcjson.Int(0),
				Skip:        btcjson.Int(5),
				Count:       btcjson.Int(10),
				VinExtra:    btcjson.Int(0),
				Reverse:     btcjson.Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("searchrawtransactions", "1Address", 0, 5, 10, 1)
			},
			staticCmd: func() interface{} {
				return btcjson.NewSearchRawTransactionsCmd("1Address",
					btcjson.Int(0), btcjson.Int(5), btcjson.Int(10), btcjson.Int(1), nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5,10,1],"id":1}`,
			unmarshaled: &btcjson.SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     btcjson.Int(0),
				Skip:        btcjson.Int(5),
				Count:       btcjson.Int(10),
				VinExtra:    btcjson.Int(1),
				Reverse:     btcjson.Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("searchrawtransactions", "1Address", 0, 5, 10, 1, true)
			},
			staticCmd: func() interface{} {
				return btcjson.NewSearchRawTransactionsCmd("1Address",
					btcjson.Int(0), btcjson.Int(5), btcjson.Int(10), btcjson.Int(1), btcjson.Bool(true), nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5,10,1,true],"id":1}`,
			unmarshaled: &btcjson.SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     btcjson.Int(0),
				Skip:        btcjson.Int(5),
				Count:       btcjson.Int(10),
				VinExtra:    btcjson.Int(1),
				Reverse:     btcjson.Bool(true),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("searchrawtransactions", "1Address", 0, 5, 10, 1, true, []string{"1Address"})
			},
			staticCmd: func() interface{} {
				return btcjson.NewSearchRawTransactionsCmd("1Address",
					btcjson.Int(0), btcjson.Int(5), btcjson.Int(10), btcjson.Int(1), btcjson.Bool(true), &[]string{"1Address"})
			},
			marshaled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5,10,1,true,["1Address"]],"id":1}`,
			unmarshaled: &btcjson.SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     btcjson.Int(0),
				Skip:        btcjson.Int(5),
				Count:       btcjson.Int(10),
				VinExtra:    btcjson.Int(1),
				Reverse:     btcjson.Bool(true),
				FilterAddrs: &[]string{"1Address"},
			},
		},
		{
			name: "sendrawtransaction",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("sendrawtransaction", "1122")
			},
			staticCmd: func() interface{} {
				return btcjson.NewSendRawTransactionCmd("1122", nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"sendrawtransaction","params":["1122"],"id":1}`,
			unmarshaled: &btcjson.SendRawTransactionCmd{
				HexTx:         "1122",
				AllowHighFees: btcjson.Bool(false),
			},
		},
		{
			name: "sendrawtransaction optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("sendrawtransaction", "1122", false)
			},
			staticCmd: func() interface{} {
				return btcjson.NewSendRawTransactionCmd("1122", btcjson.Bool(false))
			},
			marshaled: `{"jsonrpc":"1.0","method":"sendrawtransaction","params":["1122",false],"id":1}`,
			unmarshaled: &btcjson.SendRawTransactionCmd{
				HexTx:         "1122",
				AllowHighFees: btcjson.Bool(false),
			},
		},
		{
			name: "setgenerate",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("setgenerate", true)
			},
			staticCmd: func() interface{} {
				return btcjson.NewSetGenerateCmd(true, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"setgenerate","params":[true],"id":1}`,
			unmarshaled: &btcjson.SetGenerateCmd{
				Generate:     true,
				GenProcLimit: btcjson.Int(-1),
			},
		},
		{
			name: "setgenerate optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("setgenerate", true, 6)
			},
			staticCmd: func() interface{} {
				return btcjson.NewSetGenerateCmd(true, btcjson.Int(6))
			},
			marshaled: `{"jsonrpc":"1.0","method":"setgenerate","params":[true,6],"id":1}`,
			unmarshaled: &btcjson.SetGenerateCmd{
				Generate:     true,
				GenProcLimit: btcjson.Int(6),
			},
		},
		{
			name: "stop",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("stop")
			},
			staticCmd: func() interface{} {
				return btcjson.NewStopCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"stop","params":[],"id":1}`,
			unmarshaled: &btcjson.StopCmd{},
		},
		{
			name: "submitblock",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("submitblock", "112233")
			},
			staticCmd: func() interface{} {
				return btcjson.NewSubmitBlockCmd("112233", nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"submitblock","params":["112233"],"id":1}`,
			unmarshaled: &btcjson.SubmitBlockCmd{
				HexBlock: "112233",
				Options:  nil,
			},
		},
		{
			name: "submitblock optional",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("submitblock", "112233", `{"workid":"12345"}`)
			},
			staticCmd: func() interface{} {
				options := btcjson.SubmitBlockOptions{
					WorkID: "12345",
				}
				return btcjson.NewSubmitBlockCmd("112233", &options)
			},
			marshaled: `{"jsonrpc":"1.0","method":"submitblock","params":["112233",{"workid":"12345"}],"id":1}`,
			unmarshaled: &btcjson.SubmitBlockCmd{
				HexBlock: "112233",
				Options: &btcjson.SubmitBlockOptions{
					WorkID: "12345",
				},
			},
		},
		{
			name: "uptime",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("uptime")
			},
			staticCmd: func() interface{} {
				return btcjson.NewUptimeCmd()
			},
			marshaled:   `{"jsonrpc":"1.0","method":"uptime","params":[],"id":1}`,
			unmarshaled: &btcjson.UptimeCmd{},
		},
		{
			name: "validateaddress",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("validateaddress", "1Address")
			},
			staticCmd: func() interface{} {
				return btcjson.NewValidateAddressCmd("1Address")
			},
			marshaled: `{"jsonrpc":"1.0","method":"validateaddress","params":["1Address"],"id":1}`,
			unmarshaled: &btcjson.ValidateAddressCmd{
				Address: "1Address",
			},
		},
		{
			name: "verifychain",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("verifychain")
			},
			staticCmd: func() interface{} {
				return btcjson.NewVerifyChainCmd(nil, nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"verifychain","params":[],"id":1}`,
			unmarshaled: &btcjson.VerifyChainCmd{
				CheckLevel: btcjson.Int32(3),
				CheckDepth: btcjson.Int32(288),
			},
		},
		{
			name: "verifychain optional1",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("verifychain", 2)
			},
			staticCmd: func() interface{} {
				return btcjson.NewVerifyChainCmd(btcjson.Int32(2), nil)
			},
			marshaled: `{"jsonrpc":"1.0","method":"verifychain","params":[2],"id":1}`,
			unmarshaled: &btcjson.VerifyChainCmd{
				CheckLevel: btcjson.Int32(2),
				CheckDepth: btcjson.Int32(288),
			},
		},
		{
			name: "verifychain optional2",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("verifychain", 2, 500)
			},
			staticCmd: func() interface{} {
				return btcjson.NewVerifyChainCmd(btcjson.Int32(2), btcjson.Int32(500))
			},
			marshaled: `{"jsonrpc":"1.0","method":"verifychain","params":[2,500],"id":1}`,
			unmarshaled: &btcjson.VerifyChainCmd{
				CheckLevel: btcjson.Int32(2),
				CheckDepth: btcjson.Int32(500),
			},
		},
		{
			name: "verifymessage",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("verifymessage", "1Address", "301234", "test")
			},
			staticCmd: func() interface{} {
				return btcjson.NewVerifyMessageCmd("1Address", "301234", "test")
			},
			marshaled: `{"jsonrpc":"1.0","method":"verifymessage","params":["1Address","301234","test"],"id":1}`,
			unmarshaled: &btcjson.VerifyMessageCmd{
				Address:   "1Address",
				Signature: "301234",
				Message:   "test",
			},
		},
		{
			name: "verifytxoutproof",
			newCmd: func() (interface{}, er.R) {
				return btcjson.NewCmd("verifytxoutproof", "test")
			},
			staticCmd: func() interface{} {
				return btcjson.NewVerifyTxOutProofCmd("test")
			},
			marshaled: `{"jsonrpc":"1.0","method":"verifytxoutproof","params":["test"],"id":1}`,
			unmarshaled: &btcjson.VerifyTxOutProofCmd{
				Proof: "test",
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
			t.Errorf("\n%s\n%s", marshaled, test.marshaled)
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

// TestChainSvrCmdErrors ensures any errors that occur in the command during
// custom mashal and unmarshal are as expected.
/*func TestChainSvrCmdErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		result     interface{}
		marshaled string
		err        er.R
	}{
		{
			name:       "template request with invalid type",
			result:     &btcjson.TemplateRequest{},
			marshaled: `{"mode":1}`,
			err:        er.E(&json.UnmarshalTypeError{}),
		},
		{
			name:       "invalid template request sigoplimit field",
			result:     &btcjson.TemplateRequest{},
			marshaled: `{"sigoplimit":"invalid"}`,
			err:        btcjson.ErrInvalidType.Default(),
		},
		{
			name:       "invalid template request sizelimit field",
			result:     &btcjson.TemplateRequest{},
			marshaled: `{"sizelimit":"invalid"}`,
			err:        btcjson.ErrInvalidType.Default(),
		},
	} XXX -trn */

	/*t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		err := er.E(jsoniter.Unmarshal([]byte(test.marshaled), &test.result))
		if !er.FuzzyEquals(err, test.err) {
			t.Errorf("Test #%d (%s) wrong error - got %T (%v), want %v",
				i, test.name, err, err, test.err)
			continue
		}
	} XXX -trn 
}*/

// Copyright (c) 2014-2017 The btcsuite developers
// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpcclient

import (
	jsoniter "github.com/json-iterator/go"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/btcjson"
)

// ***********************
// Miscellaneous Functions
// ***********************

// NOTE: While getinfo is implemented here (in wallet.go), a pktd chain server
// will respond to getinfo requests as well, excluding any wallet information.

// FutureGetInfoResult is a future promise to deliver the result of a
// GetInfoAsync RPC invocation (or an applicable error).
type FutureGetInfoResult chan *response

// Receive waits for the response promised by the future and returns the info
// provided by the server.
func (r FutureGetInfoResult) Receive() (*btcjson.InfoWalletResult, er.R) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal result as a getinfo result object.
	var infoRes btcjson.InfoWalletResult
	err = er.E(jsoniter.Unmarshal(res, &infoRes))
	if err != nil {
		return nil, err
	}

	return &infoRes, nil
}

// GetInfoAsync returns an instance of a type that can be used to get the result
// of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See GetInfo for the blocking version and more details.
func (c *Client) GetInfoAsync() FutureGetInfoResult {
	cmd := btcjson.NewGetInfoCmd()
	return c.sendCmd(cmd)
}

// GetInfo returns miscellaneous info regarding the RPC server.  The returned
// info object may be void of wallet information if the remote server does
// not include wallet functionality.
func (c *Client) GetInfo() (*btcjson.InfoWalletResult, er.R) {
	return c.GetInfoAsync().Receive()
}

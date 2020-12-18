// Copyright (c) 2014-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpcclient

import (
	jsoniter "github.com/json-iterator/go"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/btcjson"
)

// FutureRawResult is a future promise to deliver the result of a RawRequest RPC
// invocation (or an applicable error).
type FutureRawResult chan *response

// Receive waits for the response promised by the future and returns the raw
// response, or an error if the request was unsuccessful.
func (r FutureRawResult) Receive() (jsoniter.RawMessage, er.R) {
	return receiveFuture(r)
}

// RawRequestAsync returns an instance of a type that can be used to get the
// result of a custom RPC request at some future time by invoking the Receive
// function on the returned instance.
//
// See RawRequest for the blocking version and more details.
func (c *Client) RawRequestAsync(method string, params []jsoniter.RawMessage) FutureRawResult {
	// Method may not be empty.
	if method == "" {
		return newFutureError(er.New("no method"))
	}

	// Marshal parameters as "[]" instead of "null" when no parameters
	// are passed.
	if params == nil {
		params = []jsoniter.RawMessage{}
	}

	// Create a raw JSON-RPC request using the provided method and params
	// and marshal it.  This is done rather than using the sendCmd function
	// since that relies on marshaling registered btcjson commands rather
	// than custom commands.
	id := c.NextID()
	rawRequest := &btcjson.Request{
		Jsonrpc: "1.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	marshaledJSON, errr := jsoniter.Marshal(rawRequest)
	if errr != nil {
		return newFutureError(er.E(errr))
	}

	// Generate the request and send it along with a channel to respond on.
	responseChan := make(chan *response, 1)
	jReq := &jsonRequest{
		id:            id,
		method:        method,
		cmd:           nil,
		marshaledJSON: marshaledJSON,
		responseChan:  responseChan,
	}
	c.sendRequest(jReq)

	return responseChan
}

// RawRequest allows the caller to send a raw or custom request to the server.
// This method may be used to send and receive requests and responses for
// requests that are not handled by this client package, or to proxy partially
// unmarshaled requests to another JSON-RPC server if a request cannot be
// handled directly.
func (c *Client) RawRequest(method string, params []jsoniter.RawMessage) (jsoniter.RawMessage, er.R) {
	return c.RawRequestAsync(method, params).Receive()
}

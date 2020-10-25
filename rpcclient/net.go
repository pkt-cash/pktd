// Copyright (c) 2014-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpcclient

import (
	jsoniter "github.com/json-iterator/go"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/btcjson"
)

// AddNodeCommand enumerates the available commands that the AddNode function
// accepts.
type AddNodeCommand string

// Constants used to indicate the command for the AddNode function.
const (
	// ANAdd indicates the specified host should be added as a persistent
	// peer.
	ANAdd AddNodeCommand = "add"
)

// String returns the AddNodeCommand in human-readable form.
func (cmd AddNodeCommand) String() string {
	return string(cmd)
}

// FutureAddNodeResult is a future promise to deliver the result of an
// AddNodeAsync RPC invocation (or an applicable error).
type FutureAddNodeResult chan *response

// Receive waits for the response promised by the future and returns an error if
// any occurred when performing the specified command.
func (r FutureAddNodeResult) Receive() er.R {
	_, err := receiveFuture(r)
	return err
}

// AddNodeAsync returns an instance of a type that can be used to get the result
// of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See AddNode for the blocking version and more details.
func (c *Client) AddNodeAsync(host string, command AddNodeCommand) FutureAddNodeResult {
	cmd := btcjson.NewAddNodeCmd(host, btcjson.AddNodeSubCmd(command))
	return c.sendCmd(cmd)
}

// AddNode attempts to perform the passed command on the passed persistent peer.
// For example, it can be used to add or a remove a persistent peer, or to do
// a one time connection to a peer.
//
// It may not be used to remove non-persistent peers.
func (c *Client) AddNode(host string, command AddNodeCommand) er.R {
	return c.AddNodeAsync(host, command).Receive()
}

// FutureGetPeerInfoResult is a future promise to deliver the result of a
// GetPeerInfoAsync RPC invocation (or an applicable error).
type FutureGetPeerInfoResult chan *response

// Receive waits for the response promised by the future and returns  data about
// each connected network peer.
func (r FutureGetPeerInfoResult) Receive() ([]btcjson.GetPeerInfoResult, er.R) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal result as an array of getpeerinfo result objects.
	var peerInfo []btcjson.GetPeerInfoResult
	err = er.E(jsoniter.Unmarshal(res, &peerInfo))
	if err != nil {
		return nil, err
	}

	return peerInfo, nil
}

// GetPeerInfoAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See GetPeerInfo for the blocking version and more details.
func (c *Client) GetPeerInfoAsync() FutureGetPeerInfoResult {
	cmd := btcjson.NewGetPeerInfoCmd()
	return c.sendCmd(cmd)
}

// GetPeerInfo returns data about each connected network peer.
func (c *Client) GetPeerInfo() ([]btcjson.GetPeerInfoResult, er.R) {
	return c.GetPeerInfoAsync().Receive()
}

// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package netparams

import "github.com/pkt-cash/pktd/chaincfg"

// Params is used to group parameters for various networks such as the main
// network and test networks.
type Params struct {
	*chaincfg.Params
	RPCClientPort string
	RPCServerPort string
}

// MainNetParams contains parameters specific running pktwallet and
// pktd on the BTC main network (wire.MainNet).
var MainNetParams = Params{
	Params:        &chaincfg.MainNetParams,
	RPCClientPort: "8334",
	RPCServerPort: "8332",
}

// TestNet3Params contains parameters specific running pktwallet and
// pktdd on the BTC test network (version 3) (wire.TestNet3).
var TestNet3Params = Params{
	Params:        &chaincfg.TestNet3Params,
	RPCClientPort: "18334",
	RPCServerPort: "18332",
}

// SimNetParams contains parameters specific to the simulation test network
// (wire.SimNet).
var SimNetParams = Params{
	Params:        &chaincfg.SimNetParams,
	RPCClientPort: "18556",
	RPCServerPort: "18554",
}

// PktTestNetParams contains parameters specific running pktwallet and
// pktd on the pkt.cash test network (wire.PktTestNet).
var PktTestNetParams = Params{
	Params:        &chaincfg.PktTestNetParams,
	RPCClientPort: "64513",
	RPCServerPort: "64511",
}

// PktMainNetParams contains parameters specific running pktwallet and
// pktd on the pkt.cash main network (wire.PktMainNet).
var PktMainNetParams = Params{
	Params:        &chaincfg.PktMainNetParams,
	RPCClientPort: "64765",
	RPCServerPort: "64763",
}

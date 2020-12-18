// Copyright (c) 2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package connmgr

import (
	"fmt"
	mrand "math/rand"
	"net"
	"strconv"
	"time"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/pktlog/log"
	"github.com/pkt-cash/pktd/wire/protocol"

	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/wire"
)

const (
	// These constants are used by the DNS seed code to pick a random last
	// seen time.
	secondsIn3Days int32 = 24 * 60 * 60 * 3
	secondsIn4Days int32 = 24 * 60 * 60 * 4
)

// OnSeed is the signature of the callback function which is invoked when DNS
// seeding is succesfull.
type OnSeed func(addrs []*wire.NetAddress)

// LookupFunc is the signature of the DNS lookup function.
type LookupFunc func(string) ([]net.IP, er.R)

// SeedFromDNS uses DNS seeding to populate the address manager with peers.
func SeedFromDNS(chainParams *chaincfg.Params, reqServices protocol.ServiceFlag,
	lookupFn LookupFunc, seedFn OnSeed) {
	for _, dnsseed := range chainParams.DNSSeeds {
		var host string
		if !dnsseed.HasFiltering || reqServices == protocol.SFNodeNetwork {
			host = dnsseed.Host
		} else {
			host = fmt.Sprintf("x%x.%s", uint64(reqServices), dnsseed.Host)
		}

		go func(host string) {
			randSource := mrand.New(mrand.NewSource(time.Now().UnixNano()))

			seedpeers, err := lookupFn(host)
			if err != nil {
				log.Infof("DNS discovery failed on seed %s: %v", host, err)
				return
			}
			numPeers := len(seedpeers)

			log.Infof("%d addresses found from DNS seed %s", numPeers, host)

			if numPeers == 0 {
				return
			}
			addresses := make([]*wire.NetAddress, len(seedpeers))
			intPort, _ := strconv.Atoi(chainParams.DefaultPort)
			if intPort == 0 {
				panic("SeedFromDNS: failed to set intPort")
			}
			for i, peer := range seedpeers {
				addresses[i] = wire.NewNetAddressTimestamp(
					// bitcoind seeds with addresses from
					// a time randomly selected between 3
					// and 7 days ago.
					time.Now().Add(-1*time.Second*time.Duration(secondsIn3Days+
						randSource.Int31n(secondsIn4Days))),
					0, peer, uint16(intPort))
			}

			seedFn(addresses)
		}(host)
	}
}

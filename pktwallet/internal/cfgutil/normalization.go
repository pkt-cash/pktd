// Copyright (c) 2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package cfgutil

import (
	"net"

	"github.com/pkt-cash/pktd/btcutil/er"
)

// NormalizeAddress returns the normalized form of the address, adding a default
// port if necessary.  An error is returned if the address, even without a port,
// is not valid.
func NormalizeAddress(addr, defaultPort string) (string, er.R) {
	// If the first SplitHostPort errors because of a missing port and not
	// for an invalid host, add the port.  If the second SplitHostPort
	// fails, then a port is not missing and the original error should be
	// returned.
	host, port, origErr := net.SplitHostPort(addr)
	if origErr == nil {
		return net.JoinHostPort(host, port), nil
	}
	addr = net.JoinHostPort(addr, defaultPort)
	_, _, errr := net.SplitHostPort(addr)
	if errr != nil {
		return "", er.E(origErr)
	}
	return addr, nil
}

// NormalizeAddresses returns a new slice with all the passed peer addresses
// normalized with the given default port, and all duplicates removed.
func NormalizeAddresses(addrs []string, defaultPort string) ([]string, er.R) {
	var (
		normalized = make([]string, 0, len(addrs))
		seenSet    = make(map[string]struct{})
	)

	for _, addr := range addrs {
		normalizedAddr, err := NormalizeAddress(addr, defaultPort)
		if err != nil {
			return nil, err
		}
		_, seen := seenSet[normalizedAddr]
		if !seen {
			normalized = append(normalized, normalizedAddr)
			seenSet[normalizedAddr] = struct{}{}
		}
	}

	return normalized, nil
}

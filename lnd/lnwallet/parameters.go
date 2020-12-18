package lnwallet

import (
	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/lnd/input"
	"github.com/pkt-cash/pktd/pktwallet/wallet/txrules"
)

// DefaultDustLimit is used to calculate the dust HTLC amount which will be
// send to other node during funding process.
func DefaultDustLimit() btcutil.Amount {
	return txrules.GetDustThreshold(input.P2WSHSize, txrules.DefaultRelayFeePerKb)
}

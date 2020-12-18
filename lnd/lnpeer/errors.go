package lnpeer

import "github.com/pkt-cash/pktd/btcutil/er"

// ErrPeerExiting signals that the peer received a disconnect request.
var ErrPeerExiting = er.GenericErrorType.CodeWithDetail("ErrPeerExiting",
	"peer exiting")

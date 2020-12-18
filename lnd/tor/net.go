package tor

import (
	"context"
	"net"
	"time"

	"github.com/pkt-cash/pktd/btcutil/er"
)

// TODO: this interface and its implementations should ideally be moved
// elsewhere as they are not Tor-specific.

const (
	// DefaultConnTimeout is the maximum amount of time a dial will wait for
	// a connect to complete.
	DefaultConnTimeout time.Duration = time.Second * 120
)

// DialFunc is a type defines the signature of a dialer used by our Net
// interface.
type DialFunc func(net, addr string, timeout time.Duration) (net.Conn, er.R)

// Net is an interface housing a Dial function and several DNS functions that
// allows us to abstract the implementations of these functions over different
// networks, e.g. clearnet, Tor net, etc.
type Net interface {
	// Dial connects to the address on the named network.
	Dial(network, address string, timeout time.Duration) (net.Conn, er.R)

	// LookupHost performs DNS resolution on a given host and returns its
	// addresses.
	LookupHost(host string) ([]string, er.R)

	// LookupSRV tries to resolve an SRV query of the given service,
	// protocol, and domain name.
	LookupSRV(service, proto, name string,
		timeout time.Duration) (string, []*net.SRV, er.R)

	// ResolveTCPAddr resolves TCP addresses.
	ResolveTCPAddr(network, address string) (*net.TCPAddr, er.R)
}

// ClearNet is an implementation of the Net interface that defines behavior
// for regular network connections.
type ClearNet struct{}

// Dial on the regular network uses net.Dial
func (r *ClearNet) Dial(
	network, address string, timeout time.Duration) (net.Conn, er.R) {

	c, e := net.DialTimeout(network, address, timeout)
	return c, er.E(e)
}

// LookupHost for regular network uses the net.LookupHost function
func (r *ClearNet) LookupHost(host string) ([]string, er.R) {
	c, e := net.LookupHost(host)
	return c, er.E(e)
}

// LookupSRV for regular network uses net.LookupSRV function
func (r *ClearNet) LookupSRV(service, proto, name string,
	timeout time.Duration) (string, []*net.SRV, er.R) {

	// Create a context with a timeout value.
	ctxt, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	s, c, e := net.DefaultResolver.LookupSRV(ctxt, service, proto, name)
	return s, c, er.E(e)
}

// ResolveTCPAddr for regular network uses net.ResolveTCPAddr function
func (r *ClearNet) ResolveTCPAddr(network, address string) (*net.TCPAddr, er.R) {
	c, e := net.ResolveTCPAddr(network, address)
	return c, er.E(e)
}

// ProxyNet is an implementation of the Net interface that defines behavior
// for Tor network connections.
type ProxyNet struct {
	// SOCKS is the host:port which Tor's exposed SOCKS5 proxy is listening
	// on.
	SOCKS string

	// DNS is the host:port of the DNS server for Tor to use for SRV
	// queries.
	DNS string

	// StreamIsolation is a bool that determines if we should force the
	// creation of a new circuit for this connection. If true, then this
	// means that our traffic may be harder to correlate as each connection
	// will now use a distinct circuit.
	StreamIsolation bool
}

// Dial uses the Tor Dial function in order to establish connections through
// Tor. Since Tor only supports TCP connections, only TCP networks are allowed.
func (p *ProxyNet) Dial(network, address string,
	timeout time.Duration) (net.Conn, er.R) {

	switch network {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, er.New("cannot dial non-tcp network via Tor")
	}
	return Dial(address, p.SOCKS, p.StreamIsolation, timeout)
}

// LookupHost uses the Tor LookupHost function in order to resolve hosts over
// Tor.
func (p *ProxyNet) LookupHost(host string) ([]string, er.R) {
	return LookupHost(host, p.SOCKS)
}

// LookupSRV uses the Tor LookupSRV function in order to resolve SRV DNS queries
// over Tor.
func (p *ProxyNet) LookupSRV(service, proto,
	name string, timeout time.Duration) (string, []*net.SRV, er.R) {

	return LookupSRV(
		service, proto, name, p.SOCKS, p.DNS,
		p.StreamIsolation, timeout,
	)
}

// ResolveTCPAddr uses the Tor ResolveTCPAddr function in order to resolve TCP
// addresses over Tor.
func (p *ProxyNet) ResolveTCPAddr(network, address string) (*net.TCPAddr, er.R) {
	switch network {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, er.New("cannot dial non-tcp network via Tor")
	}
	return ResolveTCPAddr(address, p.SOCKS)
}

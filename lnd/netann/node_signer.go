package netann

import (
	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/lnd/input"
	"github.com/pkt-cash/pktd/lnd/keychain"
	"github.com/pkt-cash/pktd/lnd/lnwallet"
)

// NodeSigner is an implementation of the MessageSigner interface backed by the
// identity private key of running lnd node.
type NodeSigner struct {
	keySigner keychain.SingleKeyDigestSigner
}

// NewNodeSigner creates a new instance of the NodeSigner backed by the target
// private key.
func NewNodeSigner(keySigner keychain.SingleKeyDigestSigner) *NodeSigner {
	return &NodeSigner{
		keySigner: keySigner,
	}
}

// SignMessage signs a double-sha256 digest of the passed msg under the
// resident node's private key. If the target public key is _not_ the node's
// private key, then an error will be returned.
func (n *NodeSigner) SignMessage(pubKey *btcec.PublicKey,
	msg []byte) (input.Signature, er.R) {
	// If this isn't our identity public key, then we'll exit early with an
	// error as we can't sign with this key.
	if !pubKey.IsEqual(n.keySigner.PubKey()) {
		return nil, er.Errorf("unknown public key")
	}

	// Otherwise, we'll sign the dsha256 of the target message.
	var digest [32]byte
	copy(digest[:], chainhash.DoubleHashB(msg))
	sig, err := n.keySigner.SignDigest(digest)
	if err != nil {
		return nil, er.Errorf("can't sign the message: %v", err)
	}

	return sig, nil
}

// SignCompact signs a double-sha256 digest of the msg parameter under the
// resident node's private key. The returned signature is a pubkey-recoverable
// signature.
func (n *NodeSigner) SignCompact(msg []byte) ([]byte, er.R) {
	// We'll sign the dsha256 of the target message.
	digest := chainhash.DoubleHashB(msg)

	return n.SignDigestCompact(digest)
}

// SignDigestCompact signs the provided message digest under the resident
// node's private key. The returned signature is a pubkey-recoverable signature.
func (n *NodeSigner) SignDigestCompact(hash []byte) ([]byte, er.R) {
	var digest [32]byte
	copy(digest[:], hash)

	// keychain.SignDigestCompact returns a pubkey-recoverable signature.
	sig, err := n.keySigner.SignDigestCompact(digest)
	if err != nil {
		return nil, er.Errorf("can't sign the hash: %v", err)
	}

	return sig, nil
}

// A compile time check to ensure that NodeSigner implements the MessageSigner
// interface.
var _ lnwallet.MessageSigner = (*NodeSigner)(nil)

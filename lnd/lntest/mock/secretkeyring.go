package mock

import (
	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/lnd/keychain"
)

// SecretKeyRing is a mock implementation of the SecretKeyRing interface.
type SecretKeyRing struct {
	RootKey *btcec.PrivateKey
}

// DeriveNextKey currently returns dummy values.
func (s *SecretKeyRing) DeriveNextKey(keyFam keychain.KeyFamily) (
	keychain.KeyDescriptor, er.R) {
	return keychain.KeyDescriptor{
		PubKey: s.RootKey.PubKey(),
	}, nil
}

// DeriveKey currently returns dummy values.
func (s *SecretKeyRing) DeriveKey(keyLoc keychain.KeyLocator) (keychain.KeyDescriptor,
	er.R) {
	return keychain.KeyDescriptor{
		PubKey: s.RootKey.PubKey(),
	}, nil
}

// DerivePrivKey currently returns dummy values.
func (s *SecretKeyRing) DerivePrivKey(keyDesc keychain.KeyDescriptor) (*btcec.PrivateKey,
	er.R) {
	return s.RootKey, nil
}

// ECDH currently returns dummy values.
func (s *SecretKeyRing) ECDH(_ keychain.KeyDescriptor, pubKey *btcec.PublicKey) ([32]byte,
	er.R) {
	return [32]byte{}, nil
}

// SignDigest signs the passed digest and ignores the KeyDescriptor.
func (s *SecretKeyRing) SignDigest(_ keychain.KeyDescriptor,
	digest [32]byte) (*btcec.Signature, er.R) {
	return s.RootKey.Sign(digest[:])
}

// SignDigestCompact signs the passed digest.
func (s *SecretKeyRing) SignDigestCompact(_ keychain.KeyDescriptor,
	digest [32]byte) ([]byte, er.R) {
	return btcec.SignCompact(btcec.S256(), s.RootKey, digest[:], true)
}

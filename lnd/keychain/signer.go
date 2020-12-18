package keychain

import (
	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/btcutil/er"
)

func NewPubKeyDigestSigner(keyDesc KeyDescriptor,
	signer DigestSignerRing) *PubKeyDigestSigner {
	return &PubKeyDigestSigner{
		keyDesc:      keyDesc,
		digestSigner: signer,
	}
}

type PubKeyDigestSigner struct {
	keyDesc      KeyDescriptor
	digestSigner DigestSignerRing
}

func (p *PubKeyDigestSigner) PubKey() *btcec.PublicKey {
	return p.keyDesc.PubKey
}

func (p *PubKeyDigestSigner) SignDigest(digest [32]byte) (*btcec.Signature,
	er.R) {
	return p.digestSigner.SignDigest(p.keyDesc, digest)
}

func (p *PubKeyDigestSigner) SignDigestCompact(digest [32]byte) ([]byte,
	er.R) {
	return p.digestSigner.SignDigestCompact(p.keyDesc, digest)
}

type PrivKeyDigestSigner struct {
	PrivKey *btcec.PrivateKey
}

func (p *PrivKeyDigestSigner) PubKey() *btcec.PublicKey {
	return p.PrivKey.PubKey()
}

func (p *PrivKeyDigestSigner) SignDigest(digest [32]byte) (*btcec.Signature,
	er.R) {
	return p.PrivKey.Sign(digest[:])
}

func (p *PrivKeyDigestSigner) SignDigestCompact(digest [32]byte) ([]byte,
	er.R) {
	return btcec.SignCompact(btcec.S256(), p.PrivKey, digest[:], true)
}

var (
	_ SingleKeyDigestSigner = (*PubKeyDigestSigner)(nil)
	_ SingleKeyDigestSigner = (*PrivKeyDigestSigner)(nil)
)

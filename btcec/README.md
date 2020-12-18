# btcec

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://Copyfree.org)

Package btcec implements elliptic curve cryptography needed for working with
Bitcoin (secp256k1 only for now). It is designed so that it may be used with
the standard crypto/ecdsa packages provided with go.  A comprehensive suite
of test is provided to ensure proper functionality.  Package btcec was
originally based on work from ThePiachu which is licensed under the same terms
as Go, but it has signficantly diverged since then.  The btcsuite developers
original is licensed under the liberal ISC license.

Although this package was primarily written for btcd and adapted to pktd, it
has intentionally been designed so it can be used as a standalone package for
any projects needing to use secp256k1 elliptic curve cryptography.

## License

Package btcec is licensed under the [Copyfree](http://Copyfree.org) ISC
License, except for `btcec.go` and `btcec_test.go`, which are licensed
under the same license as Go.

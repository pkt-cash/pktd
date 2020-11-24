hdkeychain
==========

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://Copyfree.org)

Package hdkeychain provides an API for BIP0032 hierarchical deterministic
extended keys

## Feature Overview

- Full BIP0032 implementation
- Single type for private and public extended keys
- Convenient cryptograpically secure seed generation
- Simple creation of master nodes
- Support for multi-layer derivation
- Easy serialization and deserialization for both private and public extended keys
- Support for custom networks by registering them with chaincfg
- Obtaining the underlying EC pubkeys, EC privkeys, and associated bitcoin addresses ties in seamlessly with existing btcec and btcutil types which provide powerful tools for working with them to do things like sign transations and generate payment scripts
- Uses the btcec package which is highly optimized for secp256k1
- Code examples including:
  - Generating a cryptographically secure random seed and deriving a master node from it
  - Default HD wallet layout as described by BIP0032
  - Audits use case as described by BIP0032
- Comprehensive test coverage including the BIP0032 test vectors
- Benchmarks

## License

Package hdkeychain is licensed under the [Copyfree](http://Copyfree.org) ISC
License.


walletdb
========

Package walletdb provides a namespaced database interface for pktwallet.

A wallet essentially consists of a multitude of stored data such as private
and public keys, key derivation bits, pay-to-script-hash scripts, and various
metadata.  One of the issues with many wallets is they are tightly integrated.
Designing a wallet with loosely coupled components that provide specific
functionality is ideal, however it presents a challenge in regards to data
storage since each component needs to store its own data without knowing the
internals of other components or breaking atomicity.

This package solves this issue by providing a namespaced database interface that
is intended to be used by the main wallet daemon.  This allows the potential for
any backend database type with a suitable driver.  Each component, which will
typically be a package, can then implement various functionality such as address
management, voting pools, and colored coin metadata in their own namespace
without having to worry about conflicts with other packages even though they are
sharing the same database that is managed by the wallet.

This interfaces provided by this package were heavily inspired by the original
boltdb project (https://github.com/boltdb/bolt) by Ben B. Johnson.

## Feature Overview

- Key/value store
- Namespace support
  - Allows multiple packages to have their own area in the database without
    worrying about conflicts
- Read-only and read-write transactions with both manual and managed modes
- Nested buckets
- Supports registration of backend databases
- Comprehensive test coverage

## License

Package walletdb is licensed under the [copyfree](http://copyfree.org) ISC
License.

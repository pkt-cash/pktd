database
========

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

Package database provides a block and metadata storage database.

Please note that this package is intended to enable pktd to support different
database backends and is not something that a client can directly access as only
one entity can have the database open at a time (for most database backends),
and that entity will be pktd.

When a client wants programmatic access to the data provided by pktd, they'll
likely want to use the [rpcclient](https://github.com/pkt-cash/pktd/tree/master/rpcclient)
package which makes use of the [JSON-RPC API](https://github.com/pkt-cash/pktd/tree/master/docs/json_rpc_api.md).

However, this package could be extremely useful for any applications requiring
Bitcoin block storage capabilities.

The default backend, ffldb, has a strong focus on speed, efficiency, and
robustness.  It makes use of GoLevelDB for the metadata, flat files for
block storage, and checksums in key areas to ensure data integrity.

## Feature Overview

- Key/value metadata store
- Bitcoin block storage
- Efficient retrieval of block headers and regions (transactions, scripts, etc)
- Read-only and read-write transactions with both manual and managed modes
- Nested buckets
- Iteration support including cursors with seek capability
- Supports registration of backend databases
- Comprehensive test coverage

## License

Package database is licensed under the [copyfree](http://copyfree.org) ISC
License.

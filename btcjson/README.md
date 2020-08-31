btcjson
=======

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

Package btcjson implements concrete types for marshalling to and from the
bitcoin JSON-RPC API.  A comprehensive suite of tests is provided to ensure
proper functionality.

Although this package was primarily written for the btcsuite and adapted to
pktd, it has intentionally been designed so it can be used as a standalone
package for any projects needing to marshal to and from bitcoin JSON-RPC
requests and responses.

Although it's possible to use this package directly to implement an RPC
client, it is not recommended since it is only intended as an infrastructure
package.  

## License

Package btcjson is licensed under the [copyfree](http://copyfree.org) ISC
License.

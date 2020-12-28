# rpctest

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://Copyfree.org)

Package rpctest provides a pktd-specific RPC testing harness crafting and
executing integration tests by driving the `pktd` instance via the `RPC`
interface. Each instance of an active harness comes equipped with a simple
in-memory HD wallet capable of properly syncing to the generated chain,
creating new addresses, and crafting fully signed transactions paying to an
arbitrary set of outputs.

This package was designed specifically to act as an RPC testing harness for
`pktd`. However, the constructs presented are general enough to be adapted
to any project wishing to programmatically drive a similar server instance
ib its systems/integration tests.

## License

Package rpctest is licensed under the [Copyfree](http://Copyfree.org) ISC
License.

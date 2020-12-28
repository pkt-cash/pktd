# Lightning Network Daemon

[![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/lightningnetwork/lnd/blob/master/LICENSE)

<img src="logo.png">

The Lightning Network Daemon (`lnd`) - is a complete implementation of a
[Lightning Network](https://lightning.network) node. `lnd` has several pluggable back-end
chain services including [`pktd`](https://github.com/pkt-cash/pktd) (a
full-node), [`bitcoind`](https://github.com/bitcoin/bitcoin), and
[`neutrino`](https://github.com/pkt-cash/pktd/neutrino) (a new experimental light client).
The project's codebase uses the [btcsuite](https://github.com/btcsuite/) set of libraries,
and also exports a large set of isolated re-usable Lightning Network related libraries
within it. In the current state `lnd` is capable of:

- Creating channels.
- Closing channels.
- Completely managing all channel states (including the exceptional ones!).
- Maintaining a fully authenticated+validated channel graph.
- Performing path finding within the network, passively forwarding incoming payments.
- Sending outgoing [onion-encrypted payments](https://github.com/lightningnetwork/lightning-onion)
  through the network.
- Updating advertised fee schedules.
- Automatic channel management ([`autopilot`](https://github.com/lightningnetwork/lnd/tree/master/autopilot)).

## Lightning Network Specification Compliance

`lnd` _fully_ conforms to the [Lightning Network specification
(BOLTs)](https://github.com/lightningnetwork/lightning-rfc). BOLT stands for:
Basis of Lightning Technology. The specifications are currently being drafted
by several groups of implementers based around the world including the
developers of `lnd`. The set of specification documents as well as our
implementation of the specification are still a work-in-progress. With that
said, the current status of `lnd`'s BOLT compliance is:

- [x] BOLT 1: Base Protocol
- [x] BOLT 2: Peer Protocol for Channel Management
- [x] BOLT 3: Bitcoin Transaction and Script Formats
- [x] BOLT 4: Onion Routing Protocol
- [x] BOLT 5: Recommendations for On-chain Transaction Handling
- [x] BOLT 7: P2P Node and Channel Discovery
- [x] BOLT 8: Encrypted and Authenticated Transport
- [x] BOLT 9: Assigned Feature Flags
- [x] BOLT 10: DNS Bootstrap and Assisted Node Location
- [x] BOLT 11: Invoice Protocol for Lightning Payments

## Developer Resources

The daemon has been designed to be as developer friendly as possible in order
to facilitate application development on top of `lnd`. Two primary RPC
interfaces are exported: an HTTP REST API, and a [gRPC](https://grpc.io/)
service. The exported API's are not yet stable, so be warned: they may change
drastically in the near future.

An automatically generated set of documentation for the RPC APIs can be found
at [api.lightning.community](https://api.lightning.community). A set of developer
resources including talks, articles, and example applications can be found at:
[dev.lightning.community](https://dev.lightning.community).

## Safety

When operating a mainnet `lnd` node, please refer to our [operational safety
guildelines](docs/safety.md). It is important to note that `lnd` is still
**beta** software and that ignoring these operational guidelines can lead to
loss of funds.

## Security

The **upstream** developers of `lnd` take security _very_ seriously. The
disclosure of security vulnerabilities helps them secure the health of `lnd`,
the privacy of our users, and also the health of the Lightning Network as a
whole. If you find any issues regarding security or privacy, please disclose
the information responsibly by sending an email to <mailto:security@lightning.engineering>,
preferably encrypted using the Lightning Labs designated PGP key
(`91FE464CD75101DA6B6BAB60555C6465E5BCB3AF`) which can be found
[here](https://gist.githubusercontent.com/Roasbeef/6fb5b52886183239e4aa558f83d085d3/raw/5fa96010af201628bcfa61e9309d9b13d23d220f/security@lightning.engineering).

## Further reading

- [Step-by-step send payment guide with docker](https://github.com/lightningnetwork/lnd/tree/master/docker)
- [Contribution guide](https://github.com/lightningnetwork/lnd/blob/master/docs/code_contribution_guidelines.md)

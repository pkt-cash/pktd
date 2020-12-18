pktwallet
=========

pktwallet is a daemon handling bitcoin wallet functionality for a
single user. It acts as both an RPC client to `pktd` and an RPC server
for wallet clients and legacy RPC applications.

Public and private keys are derived using the hierarchical
deterministic format described by
[BIP0032](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki).
Unencrypted private keys are not supported and are never written to
disk.  pktwallet uses the
`m/44'/<coin type>'/<account>'/<branch>/<address index>`
HD path for all derived addresses, as described by
[BIP0044](https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki).

Due to the sensitive nature of public data in a BIP0032 wallet,
pktwallet provides the option of encrypting not just private keys, but
public data as well.  This is intended to thwart privacy risks where a
wallet file is compromised without exposing all current and future
addresses (public keys) managed by the wallet. While access to this
information would not allow an attacker to spend or steal coins, it
does mean they could track all transactions involving your addresses
and therefore know your exact balance. In a future release, public data
encryption will extend to transactions as well.

pktwallet is not an SPV client and requires a connection to a local or
remote `pktd` instance for asynchronous blockchain queries, as well as
to receive notifications over websockets.

Wallet clients can use one of two RPC servers:

  1. A legacy JSON-RPC server mostly compatible with Bitcoin Core

     The JSON-RPC server exists to ease the migration of wallet applications
     from Core, but complete compatibility is not guaranteed.  Some portions of
     the API (and especially accounts) have to work differently due to other
     design decisions (mostly due to BIP0044).  However, if you find a
     compatibility issue and feel that it could be reasonably supported, please
     report an issue.  This server is enabled by default.

  2. An experimental RPC server

     The RPC server uses a new API built for pktwallet, but, because the API is
	 not stable, the server is feature gated behind a user configurations option
	 (`--experimentalrpclisten`).  If you a) don't mind applications breaking
	 due to API changes, b) have issues with the legacy API, or c) need to get
	 notifications for changes to the wallet, this is the RPC server to use.

## Issue Tracker

The [integrated GitHub issue tracker](https://github.com/pkt-cash/pktd/issues)
is used for this project.

## License

pktwallet is licensed under the liberal ISC License.

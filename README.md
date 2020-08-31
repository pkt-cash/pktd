pktd
====

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

pktd is the full node PKT Cash implementation written in Go (golang).

This project is currently under active development and is in a Beta state.

It properly downloads, validates, and serves the PKT Cash blockchain, using
rules for block acceptance based on Bitcoin Core, with the addition of
PacketCrypt Proofs. 

It relays newly mined blocks, maintains a transaction pool, and relays
individual transactions that have not yet made it into a block.  It ensures
all individual transactions admitted to the pool follow the rules required
by the PKT Cash chain, and includes strict checks which filter transactions
based on miner requirements ("standard" transactions).

Unlike other similar software, pkt does *NOT* directly include wallet
functionality - this was a very intentional design decision.  This means
you can't actually make or receive payments directly with pktd directly.

That functionality is provided by the included
[pktwallet](https://github.com/pkt-cash/pktd/pktwallet) package.

## Requirements

[Go](http://golang.org) 1.14 or newer.

## Issue Tracker

The [integrated github issue tracker](https://github.com/pkt-cash/pktd/issues)
is used for this project.

## Documentation

The documentation is a work-in-progress.

It is located in the [docs](https://github.com/pkt-cash/pktd/tree/master/docs) folder.

## License

pktd is licensed under the [copyfree](http://copyfree.org) ISC License.

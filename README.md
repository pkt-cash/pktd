pktd
====

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

`pktd` is the full node *PKT Cash* implementation, written in Go (golang).

This project is currently under active development.

It properly downloads, validates, and serves the PKT Cash blockchain, using
rules for block acceptance based on Bitcoin Core, with the addition of
PacketCrypt Proofs. 

It relays newly mined blocks, and individual transactions that have not yet
made it into a block, as well as maintaining a transaction pool. All
individual transactions admitted to the pool follow the rules required
by the PKT Cash chain, and includes strict checks which filter transactions
based on miner requirements ("standard" transactions).

Unlike other similar software, `pktd` does *NOT* directly include wallet
functionality - this was a very intentional design decision.  This means
you can't actually make or receive payments directly with `pktd` directly.

That functionality is provided by the included
[pktwallet](https://github.com/pkt-cash/pktd/pktwallet) package.

## Requirements

[Go](http://golang.org) 1.14 or later.
A recent release of Git.

## Issue Tracker

The [integrated github issue tracker](https://github.com/pkt-cash/pktd/issues)
is used for this project.

## Building

Using `git`, clone the project from the repository:

`git clone https://github.com/pkt-cash/pktd`

Use the `./do` shell script to build `pktd`, `pktwallet`, and `pktctl`.
Several different build modes and options are available;
`./do -h` will give usage details.

Build modes are tested on Linux/x86_64, and are supported on a best-effort
basis for other platforms. If you wish to verify your build output is fully
using the mode you specific, you can use the GNU `file` command, Debian's
`hardening-check` utility, [checksec](https://github.com/slimm609/checksec.sh),
or similar tools provided by your operating system. Also note, not every
operating system fully supports or provides for every Go runtime feature.

NOTE: It is highly recommended to use a Go distribution from the
[official Golang homepage](https://golang.org/dl). Linux distributions often
provide different defaults and/or patches against the official distribution,
to meet specific requirements (for example, Red Hat backports security fixes,
and provides a different default linker configuration vs. upstream Golang.)

Useful assistance can only be provided for binaries compiled from unmodified
sources, and using the upstream Golang toolchain. Sorry, we simply cannot test
and support every different distribution-provided toolchain out there. 

An official Golang Linux installer always available for download [here](https://storage.googleapis.com/golang/getgo/installer_linux).

## Documentation

The documentation is a work-in-progress.

It is located in the [docs](https://github.com/pkt-cash/pktd/tree/master/docs) folder.

## License

pktd is licensed under the [copyfree](http://copyfree.org) ISC License.

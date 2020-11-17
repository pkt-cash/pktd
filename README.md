# PKT Cash

## A blockchain utilizing PacketCrypt, a bandwidth-hard proof-of-work algorithm.

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://Copyfree.org)
[![MadeWithGo](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](http://golang.org)
[![GoVersion](https://img.shields.io/github/go-mod/go-version/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd/blob/master/go.mod)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/pkt-cash/pktd)](https://pkg.go.dev/github.com/pkt-cash/pktd)
[![GoReportCard](https://goreportcard.com/badge/github.com/pkt-cash/pktd)](https://goreportcard.com/report/github.com/pkt-cash/pktd)
[![GitHubRelease](https://img.shields.io/github/release/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd/releases/)
[![GitHubTag](https://img.shields.io/github/tag/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd/tags/)
[![LocCount](https://img.shields.io/tokei/lines/github/pkt-cash/pktd.svg)](https://github.com/XAMPPRocky/tokei)
[![GitHubCodeSize](https://img.shields.io/github/languages/code-size/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd)

# pktd

`pktd` is the reference full node [*PKT Cash*](https://pkt.cash/)
implementation, written in [Go](https://go.dev/).

The `pktd` software is currently under active development and considered to
be *beta* quality software. In particular, the `develop` branch of `pktd` is
highly experimental, and should not be used in production environments, or on
the PKT Cash mainnet, without specific requirements. The `master` branch is
intended to provide stable sources appropriate for production deployments.

`pktd` is the primary mainnet node software for the PKT blockchain. It is
known to correctly download, validate, and serve the blockchain, using rules
for block acceptance based mainly on Bitcoin Core, but with the addition of
[PacketCrypt proofs](https://pkt.cash/PacketCrypt-2020-09-04.pdf), providing 
verification of the *bandwidth-hard* proof-of-work.

The full-node software uses a peer-to-peer network to relay both newly mined
blocks and individual transactions that have not yet made it into a block,
while maintaining a local transaction pool. Any individual transaction
admitted to the pool must follow the rules as defined by the network operators,
which include strict checks to filter transactions based on miner requirements
("standard" vs "non-standard" transactions).

Unlike similar software, `pktd` does *NOT* directly implement any wallet
functionality - this was an intentional design decision. While you cannot
make or receive payments with `pktd` directly, those functions are provided
by the bundled, but separate, [pktwallet](https://github.com/pkt-cash/pktd/pktwallet) package.


## Requirements

* Google [Go](http://golang.org) (Golang) version 1.14 or higher.
* A somewhat recent release of Git (used to clone the repository, and by Go to download dependencies).


## Bug reporting

* The [integrated GitHub issue tracker](https://github.com/pkt-cash/pktd/issues) is used for this project.
  * When submitting a [new issue](https://github.com/pkt-cash/pktd/issues/new),
    please provide the output of `go version` and `go env`. Linux users should
	also provide the output of `cat /etc/*elease`, and `lsb_release -a`. This
	will help to identify distribution or environment-specific quirks.


## Building pktd

Using `git`, clone the project from the repository:

`git clone https://github.com/pkt-cash/pktd`

Then, use the `./do` script to build `pktd`, `pktwallet`, and `pktctl`.


## Linux-specific notes

It is *highly* recommended to use only the official Go toolchain distributed
by Google. It can be obtained from the [Google Go homepage](https://golang.org/dl).

Go toolchains provided by Linux distributions often set different defaults,
make their own modifications, and apply non-standard patches to software they
distribute, sometimes deviating significantly from official upstream releases.
This is usually done to meet distribution-specific policies and requirements.

For example, [Red Hat](https://bugzilla.redhat.com/buglist.cgi?bug_status=NEW&bug_status=ASSIGNED&bug_status=ON_QA&component=golang)
will backport specific changes, and currently ships with a different linker
configuration than the upstream distribution. [Debian](https://tracker.debian.org/pkg/golang-defaults)
is also known to make changes to upstream sources, often extensively, and 
sometimes [incorrectly](https://www.zdnet.com/article/debian-and-ubuntu-openssl-generates-useless-crypto-keys/).

Because of this, support can only be provided for Linux when binaries are
compiled from the official source code, using an unmodified upstream toolchain.

If you have doubts about your installed Go software, Google provides an
official installer for Linux, available [here](https://storage.googleapis.com/golang/getgo/installer_linux).

## Documentation

Full documentation for `pktd`, `pktwallet`, and `pktctl` is currently a
work-in-progress.

First, you should review the local [README](https://github.com/pkt-cash/pktd/blob/master/README.md)
file and browse the [docs](https://github.com/pkt-cash/pktd/tree/master/docs) directory.

GoDoc documentation is also provided. It is currently mostly developer focused.
You can view it by running `godoc -http=:6543` in the directory containing the
`pktd` source code, and then loading [http://localhost:6543/pkg/github.com/pkt-cash/pktd/](http://localhost:6060/pkg/github.com/pkt-cash/pktd/) in your browser.

The GoDoc documentation is also available via [go.dev](https://pkg.go.dev/github.com/pkt-cash/pktd),
and is accessible even without a local copy of the source distribution.

## License

`pktd` is licensed under the [Copyfree](http://Copyfree.org) **ISC License**.


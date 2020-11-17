# PKT Cash

### A blockchain utilizing *PacketCrypt*, a bandwidth-hard proof-of-work algorithm.

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://Copyfree.org) [![MadeWithGo](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](http://golang.org) [![GoVersion](https://img.shields.io/github/go-mod/go-version/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd/blob/master/go.mod) [![PkgGoDev](https://pkg.go.dev/badge/github.com/pkt-cash/pktd)](https://pkg.go.dev/github.com/pkt-cash/pktd) [![GitHubRelease](https://img.shields.io/github/release/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd/releases/) [![GitHubTag](https://img.shields.io/github/tag/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd/tags/) [![LocCount](https://img.shields.io/tokei/lines/github/pkt-cash/pktd.svg)](https://github.com/XAMPPRocky/tokei) [![GitHubCodeSize](https://img.shields.io/github/languages/code-size/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd)

## pktd

`pktd` is the reference full node [*PKT Cash*](https://pkt.cash/)
implementation, written in [Go](https://go.dev/).

The `pktd` software is currently under active development and releases should
be considered to be *beta* quality software. The unreleased `develop` branch
of `pktd` is highly experimental, not rigorously tested, and should **not**
be used in production environments. The `master` branch, or the most recent
tagged release, is intended to provide stable and tested sources appropriate
for production deployments.

`pktd` is the primary mainnet node software for the PKT blockchain. It is
known to correctly download, validate, and serve the chain, using rules for
block acceptance based (mainly) on Bitcoin Core, with the addition of
[PacketCrypt proofs](https://pkt.cash/PacketCrypt-2020-09-04.pdf) to provide 
verification of the *bandwidth-hard* proof-of-work algorithm.

The full-node software uses a domain-specific peer-to-peer network for relaying
newly mined blocks and individual transactions not yet included in a block. It
also maintains a local transaction pool, where any individual transaction
must follow rules defined by the network operators to be accepted into the
pool. These rules allow for strict rule checks, filtering transactions based
on miner requirements (*"standard" vs. "non-standard" transactions*).

Unlike similar software, `pktd` does *NOT* directly provide wallet capability.
This was an intentional design decision to enhance security. While you cannot
make or receive payments with `pktd` directly, that functionality is provided
for by bundled, but separate [pktwallet](https://github.com/pkt-cash/pktd/pktwallet) software package.


## Requirements

* [Go](http://golang.org) (*Golang*), release 1.14 or later.
* A somewhat recent release of Git (*used to clone the repository, and by Go to download dependencies*).


## Bug reporting

* The [integrated GitHub issue tracker](https://github.com/pkt-cash/pktd/issues) is used for this project.
  * When submitting a [new issue](https://github.com/pkt-cash/pktd/issues/new),
    please provide the output of `go version` and `go env`. 
  * Linux users should also provide the output of `cat /etc/*elease`, and 
    `lsb_release -a`. This will help us to identify any distribution or 
	environment-specific issues.


## Building pktd

Using `git`, clone the project from the repository:
* `git clone https://github.com/pkt-cash/pktd`

Then, use the `do` script to build `pktd`, `pktwallet`, and `pktctl`:
* `cd pktd && ./do`


## Linux distributions

It is *highly* recommended to use the official Go toolchain distributed
by Google, which can be obtained from the [Go homepage](https://golang.org/dl).

Software packaged by Linux distributions is often built with different 
defaults settings and non-standard patches, sometimes deviating significantly
from corresponding official upstream releases. This usually done to meet
distribution-specific policies and requirements. This is a hazard for software
software such as `pktd`, which operates based on distributed consensus, and
requires *bug-for-bug* compatability in consensus-critical code. Nodes which
incorporate changes, even well-intentioned *"bug fixes"* might not be able to
properly interoperate on the main network - in sufficient numbers, such nodes
may become unintentionally forked, incompatible with the main network and only
able to operate with similarly modified nodes.

While not *currently* the source of any *known* issue, [Red Hat](https://bugzilla.redhat.com/buglist.cgi?bug_status=NEW&bug_status=ASSIGNED&bug_status=ON_QA&component=golang)
will backport cherry-picked updates from unreleased versions (or produce their
own patches) to fix specific issues in their released software. They are also 
currently shipping their Golang distributions with a different linker
configuration than Google upstream release. [Debian](https://tracker.debian.org/pkg/golang-defaults)
is well known for making changes to upstream sources, often extensively, and
sometimes [incorrectly](https://www.zdnet.com/article/debian-and-ubuntu-openssl-generates-useless-crypto-keys/).

For this reason, support can only be provided for Linux when the software is
compiled from unmodified source code and when using the official toolchain. If
you have any doubts about your installed Go software, Google provides a [Go installer for Linux](https://storage.googleapis.com/golang/getgo/installer_linux).


## Documentation

Currently, full documentation for `pktd`, `pktwallet`, and `pktctl` is a
"work-in-progress".

GoDoc documentation is provided, currently mostly developer-focused. You can
view it by running `godoc -http=:6543` in the directory containing your local
`pktd` source code, then loading [http://localhost:6543/pkg/github.com/pkt-cash/pktd/](http://localhost:6060/pkg/github.com/pkt-cash/pktd/) in your browser.
The same GoDoc documentation is also available at [pkg.go.dev/github.com/pkt-cash/pktd](https://pkg.go.dev/github.com/pkt-cash/pktd),
which is accessible online and does not require a local copy of the sources or
the `godoc` tool installed.

There is also documentation in the [docs](https://github.com/pkt-cash/pktd/tree/master/docs) directory of the source tree which you should review.


## Community

* [PKT.chat](https://pkt.chat) is a [Matterfoss](https://github.com/cjdelisle/Matterfoss) server providing real-time interaction for the PKT community.
* Other options are available and listed on the [PKT.cash](https://pkt.cash/community/) web site.


## License

`pktd` is licensed under the [Copyfree](http://Copyfree.org) **ISC License**.




# PKT Cash

### A blockchain utilizing [PacketCrypt]((https://pkt.cash/PacketCrypt-2020-09-04.pdf)), a new *bandwidth-hard* proof-of-work algorithm.

 [![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://Copyfree.org) 
 [![GoVersion](https://img.shields.io/github/go-mod/go-version/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd/blob/master/go.mod) 
 [![PkgGoDev](https://pkg.go.dev/badge/github.com/pkt-cash/pktd)](https://pkg.go.dev/github.com/pkt-cash/pktd) 
 [![GitHubRelease](https://img.shields.io/github/release/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd/releases/) 
 [![GitHubTag](https://img.shields.io/github/tag/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd/tags/) 
 [![LocCount](https://img.shields.io/tokei/lines/github/pkt-cash/pktd.svg)](https://github.com/XAMPPRocky/tokei) 
 [![GitHubCodeSize](https://img.shields.io/github/languages/code-size/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd) 
 [![CoverageStatus](https://coveralls.io/repos/pkt-cash/pktd/badge.svg?branch=develop)](https://coveralls.io/pkt-cash/pktd?branch=develop) 
 [![GoReport](https://goreportcard.com/badge/github.com/pkt-cash/pktd)](https://goreportcard.com/report/github.com/pkt-cash/pktd) 


## pktd

`pktd` is the reference full node [*PKT Cash*](https://pkt.cash/)
implementation, written in [Go](https://go.dev/).

`pktd` is the primary mainnet node software for the PKT blockchain. It is
known to correctly download, validate, and serve the blockchain, using rules for
block acceptance based primarily on Bitcoin Core, with the addition of
[PacketCrypt proofs](https://pkt.cash/PacketCrypt-2020-09-04.pdf) to provide 
verification for our unique *bandwidth-hard* proof-of-work algorithm.

The PKT blockchain uniquely provides for a [Network Steward](https://pkt.cash/network-steward/),
chosen by consensus, who receives a portion of the PKT mined from each block.
The role of the Network Steward is to support the PKT blockchain and ecosystem,
fighting for enhanced privacy and greater individual autonomy for all. Anyone
holding PKT Cash is eligible to cast a vote to choose the Network Steward. The
Network Steward's treasury account does not accumulate indefinitely - it *"burns"*
any balance older than 129,600 blocks (~3 months), which encourages the Steward to
use these funds to provide support and funding for [worthy projects](https://github.com/pkt-cash/ns-projects).

The `pktd` software is currently under active development, and releases should
be considered *beta* quality. The unreleased code in the `develop` branch is
highly experimental and of *alpha* quality, not yet rigorously tested, and
should **not** be used in production environments.

It is recommended that users use the most recent tagged release (or a
checkout of the `master` branch) for any production deployments.

The `pktd` full-node software utilizies a domain-specific peer-to-peer network
to relay newly mined blocks, as well as individual transactions not yet included
in a block. It also maintains a local transaction pool, consisting of transactions
that have been accepted based on rules defined by network consensus. Transactions
undergo strict checks and filtering, based on miner-defined requirements, allowing
them to be separated into two classes, *"standard"* and *"non-standard"*, and
processed accordingly.

Unlike most cryptocurrency software, `pktd` does *NOT* directly provide built-in
wallet capabilities. This intentional design decision was made to enhance overall
security and encourage software modularity. This means you cannot make or receive
payments using `pktd` directly, however, this functionality is provided for by
the bundled, but separate, [pktwallet](https://github.com/pkt-cash/pktd/pktwallet)
software package. Other options are currently under development.


## Requirements

* [Go](http://golang.org) (*Golang*), release 1.14 or later.
* A somewhat recent release of Git (*used to clone the repository, and by Go to download dependencies*).


## Bug reporting

* The [integrated GitHub issue tracker](https://github.com/pkt-cash/pktd/issues) is used for this project.
  * When submitting a [new issue](https://github.com/pkt-cash/pktd/issues/new), please provide the output of `go version` and `go env`. 
  * Linux users should also provide the output of `cat /etc/*elease`, and `lsb_release -a`. 
    * This will help us to identify any distribution or environment-specific issues.


## Building pktd

Using `git`, clone the project from the repository:
* `git clone https://github.com/pkt-cash/pktd`

Then, use the `do` script to build `pktd`, `pktwallet`, and `pktctl`:
* `cd pktd && ./do`


## Linux distributions

It is *highly* recommended to use the official Go toolchain distributed
by Google, available to download from the [Go homepage](https://golang.org/dl).

Software built and packaged by Linux distributions is often compiled with 
different defaults and non-standard patches, often deviating significantly
from the corresponding official upstream release. This usually done to meet
distribution-specific policies and requirements, and presents a unique hazard
for software software such as `pktd`, operating on distributed consensus. This
class of software is unique, as it must maintain *bug-for-bug* compatability in
consensus-critical codepaths. Nodes which incorporate changes, even well-intentioned
*"fixes"*, might not be able to properly interoperate on the main PKT network,
and, in sufficient numbers, such nodes could unintentionally '*fork*', 
creating an isolated network only able to work with similarly modified nodes.

While not *currently* the source of any *known* issues, [Red Hat](https://bugzilla.redhat.com/buglist.cgi?bug_status=NEW&bug_status=ASSIGNED&bug_status=ON_QA&component=golang)
will backport cherry-picked updates from unreleased versions (and produce their
own patches) to fix specific issues in their released software. They are also
known to currently ship a Go distribution with a much different linker
configuration than the upstream release. [Debian](https://tracker.debian.org/pkg/golang-defaults)
is well known for making changes to upstream sources, often extensively, sometimes [incorrectly](https://www.zdnet.com/article/debian-and-ubuntu-openssl-generates-useless-crypto-keys/).

For these reasons, support can only be provided for Linux when the software is
compiled from unmodified source code using the official toolchain. If you have
any doubts about your installed Go software, Google provides an official
[Go installer for Linux](https://storage.googleapis.com/golang/getgo/installer_linux).


## Documentation

Currently, documentation for `pktd`, `pktwallet`, and `pktctl` is a "work-in-progress".

**GoDoc** documentation is provided, but is mostly developer-focused at this time.
You can view it by running `godoc -http=:6543` in the directory containing your
`pktd` source code tree, then loading
[http://localhost:6543/pkg/github.com/pkt-cash/pktd/](http://localhost:6060/pkg/github.com/pkt-cash/pktd/)
in your web browser. The same **GoDoc** documentation is also available at
[pkg.go.dev/github.com/pkt-cash/pktd](https://pkg.go.dev/github.com/pkt-cash/pktd),
and does not require a local copy of the source code, or the `godoc` tool.

There is also documentation in the [docs](https://github.com/pkt-cash/pktd/tree/master/docs)
directory of the source tree which you should review.


## Community

* [PKT.chat](https://pkt.chat) is a [Matterfoss](https://github.com/cjdelisle/Matterfoss) server providing real-time interaction for the PKT community.
* Other options are available and listed on the [PKT.cash](https://pkt.cash/community/) web site.


## License

`pktd` is licensed under the [Copyfree](http://Copyfree.org) **ISC License**.


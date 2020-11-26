# PKT Cash

### A blockchain utilizing [PacketCrypt]((https://pkt.cash/PacketCrypt-2020-09-04.pdf)), a new *bandwidth-hard* proof-of-work algorithm.

 [![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://Copyfree.org)
 [![PkgGoDev](https://pkg.go.dev/badge/github.com/pkt-cash/pktd)](https://pkg.go.dev/github.com/pkt-cash/pktd)
 [![GoReport](https://goreportcard.com/badge/github.com/pkt-cash/pktd)](https://goreportcard.com/report/github.com/pkt-cash/pktd)
 [![GitHubRelease](https://img.shields.io/github/release/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd/releases/)
 [![GitHubTag](https://img.shields.io/github/tag/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd/tags/)
 [![LocCount](https://img.shields.io/tokei/lines/github/pkt-cash/pktd.svg)](https://github.com/XAMPPRocky/tokei)
 [![GitHubCodeSize](https://img.shields.io/github/languages/code-size/pkt-cash/pktd.svg)](https://github.com/pkt-cash/pktd)
 [![CoverageStatus](https://coveralls.io/repos/pkt-cash/pktd/badge.svg?branch=develop)](https://coveralls.io/pkt-cash/pktd?branch=develop)
 [![TickgitTODOs](https://img.shields.io/endpoint?url=https://api.tickgit.com/badge?repo=github.com/pkt-cash/pktd)](https://www.tickgit.com/browse?repo=github.com/pkt-cash/pktd)


## pktd

`pktd` is the reference full node [*PKT Cash*](https://pkt.cash/)
implementation, written in [Go](https://go.dev/).

`pktd` is the primary mainnet node software for the PKT blockchain. It is
known to correctly download, validate, and serve the blockchain, using rules for
block acceptance based primarily on Bitcoin Core, with the addition of
[PacketCrypt proofs](https://pkt.cash/PacketCrypt-2020-09-04.pdf) to provide 
verification of the unique *bandwidth-hard* proof-of-work algorithm.

The PKT blockchain provides for a [Network Steward](https://pkt.cash/network-steward/),
chosen by consensus, who receives a portion of the PKT mined from each block.
The role of the Network Steward is to support the PKT blockchain and ecosystem,
fighting for enhanced privacy and greater individual autonomy for all. Anyone
holding PKT Cash is eligible to cast a vote to choose the Network Steward. The
Network Steward's treasury account does not accumulate indefinitely - it *"burns"*
any balance older than 129,600 blocks (~3 months), which encourages the Steward to
use their PKT holdings to provide support and funding for [worthy projects](https://github.com/pkt-cash/ns-projects).

The `pktd` software is under active development - releases should be considered
*beta* quality software. The unreleased code in the `develop` branch is highly
experimental and of *alpha* quality - it has not yet been rigorously tested,
and should **not** be used in production environments.

It is recommended that users run the most recent tagged release (or a
checkout of the `master` branch) for production deployments.

The `pktd` full-node software utilizies peer-to-peer networking to relay newly
mined blocks, as well as individual transactions not yet included in a block.
It also maintains a local transaction pool, consisting of transactions that
have been accepted, based on rules defined by network consensus. Transactions
undergo strict checking and filtering based on miner-defined requirements,
separating them into two classes: *"standard"* and *"non-standard"*, allowing
them to be processed accordingly.

Unlike most cryptocurrency software, `pktd` does *NOT* provide any "built-in"
wallet capability. This intentional design decision was made to enhance overall
security and software modularity. While this means you cannot make or receive
payments using `pktd` directly; this functionality is provided by the bundled,
but separate, [pktwallet](https://github.com/pkt-cash/pktd/pktwallet) package.
Additional alternative wallet packages are currently under active development.


## Requirements

* [Go](http://golang.org) (*Golang*), release **1.14.1** or later, running on a supported **64-bit** platform.
* A somewhat recent release of Git (*used to clone the repository, and by Go to download dependencies*).


## Building pktd

* Using `git`, clone the project from the repository:
   * `git clone https://github.com/pkt-cash/pktd`
* Then, use the `do` script to build `pktd`, `pktwallet`, and `pktctl`:
   * `cd pktd && ./do`

*Optionally*, run extended tests and generate code coverage reports:
* Using `go get`, install the `gocov` and `gocov-html` tools into your `GOPATH`:
   * `GO111MODULES=off go get -t -u github.com/axw/gocov/gocov`
   * `GO111MODULES=off go get -t -u github.com/matm/gocov-html`  
* Then, run the coverage generation script: (*zsh, bash, ksh, or mksh required*.)
   * `PATH=$(go env GOPATH)/bin:${PATH} ./cov_report.sh` 


## Bug Reporting

The **GitHub** [**Issue Tracker**](https://github.com/pkt-cash/pktd/issues) is used for this project.

  * All users submitting a [new bug report](https://github.com/pkt-cash/pktd/issues/new/choose) should attach the output of:
     * `go version`, `go env`
     * `git describe --tags --always --abbrev=40`, `git status -s -b` 
  * macOS X users should attach the output of:
     * `uname -a`, `sw_vers`
     * `system_profiler -detailLevel mini`, `serverinfo --plist`. 
  * Linux users should attach the output of:
     * `uname -a`, `cat /etc/*elease`, `lsb_release -a`
  * Windows users should attach the *PowerShell* output of:
     * `Get-CimInstance Win32_OperatingSystem | FL * | ?{$_ -notmatch 'SerialNumber'}`
  * POSIX environment users (*including* Linux, OS X, WSL/WSL2, other UNIX systems, etc.) should attach the output of:
     * `command -p getconf -a`

This extra information is useful to identify potential operating system, distribution, or environment-specific issues. 


## Linux Distributions

It is *highly* recommended to use the official Go toolchain distributed
by Google, available for download from the [Go homepage](https://golang.org/dl).

Software built and packaged for Linux distributions is often compiled with 
different defaults and non-standard patches, often deviating significantly
from the corresponding official upstream release. This usually done to meet
distribution-specific policies and requirements, but presents a unique hazard
for software software such as `pktd`, operating on distributed consensus. This
class of software is unique, in that it must maintain *bug-for-bug* compatability
in consensus-critical codepaths. Nodes which incorporate their own changes, even
well-intentioned *"fixes"*, may not be able to properly interoperate on the main
network, and, in sufficient numbers, these nodes could, unintentionally, create an
isolated or '*forked*' blockchain, usable only by other similarly broken nodes.

While not *currently* the source of any *known* issue, [Red Hat](https://bugzilla.redhat.com/buglist.cgi?bug_status=NEW&bug_status=ASSIGNED&bug_status=ON_QA&component=golang)
will backport cherry-picked updates from unreleased versions (or produce their
own patches) to fix specific issues in their releases. They are also
known to currently ship a Go distribution with a much different linker
configuration than the upstream. [Debian](https://tracker.debian.org/pkg/golang-defaults)
is well known for making changes to upstream sources, often extensively,
sometimes [incorrectly](https://www.zdnet.com/article/debian-and-ubuntu-openssl-generates-useless-crypto-keys/).

For these reasons, support can only be provided for Linux when the software is
compiled from unmodified source code, using the official toolchain. If you have
any doubts about your installed Go software, Google provides an automated
[Go installer for Linux](https://storage.googleapis.com/golang/getgo/installer_linux).


## Documentation

Currently, documentation for `pktd`, `pktwallet`, and `pktctl` is a "work-in-progress".

**GoDoc** documentation is provided, but is mostly developer-focused at this time.
It can be viewed by running `godoc -http=:6543` in the directory containing the
`pktd` source code tree, then loading
[http://localhost:6543/pkg/github.com/pkt-cash/pktd/](http://localhost:6543/pkg/github.com/pkt-cash/pktd/)
in a web browser.

The same **GoDoc** documentation is also available at
[pkg.go.dev/github.com/pkt-cash/pktd](https://pkg.go.dev/github.com/pkt-cash/pktd),
which does not require a local copy of the source code, or the `godoc` tool installed.

There is also documentation in the [docs](https://github.com/pkt-cash/pktd/tree/develop/docs)
directory of the source tree available for review.


## Community

* [PKT.chat](https://pkt.chat) is a [Matterfoss](https://github.com/cjdelisle/Matterfoss) server providing real-time interaction for the PKT community.
* Other options are available and listed on the [PKT.cash](https://pkt.cash/community/) web site.


## License

`pktd` is licensed under the [Copyfree](http://Copyfree.org) **ISC License**.

#!/usr/bin/env sh 
die() { printf '%s\n' "$*" >&2; exit 1; }
export GO111MODULE=on
PKTD_GIT_ID=$(git update-index -q --refresh 2>/dev/null; git describe --tags --dirty 2>/dev/null)
if [ -n "${PKT_FAIL_DIRTY:-}" ]; then
	export GIT_PAGER=cat; git diff && die "Build is dirty, aborting."
fi
PKTD_LDFLAGS="-X github.com/pkt-cash/pktd/pktconfig/version.appBuild=${PKTD_GIT_ID:?"Failed to generate PKTD_GIT_ID, aborting."}"
mkdir -p ./bin || die "Failed to create output directory, aborting."
printf '%s\n' "Building pktd"
go build -ldflags="${PKTD_LDFLAGS:?}" -o ./bin/pktd || die "Failed to build pktd, aborting."
printf '%s\n' "Building wallet"
go build -ldflags="${PKTD_LDFLAGS:?}" -o ./bin/pktwallet ./pktwallet || die "Failed to build wallet, aborting."
printf '%s\n' "Building btcctl"
go build -ldflags="${PKTD_LDFLAGS:?}" -o ./bin/pktctl ./cmd/btcctl || die "Failed to build pktctl, aborting."
printf '%s\n' "Running tests"
go test ./... || die "Tests failed, aborting."
./bin/pktd --version || die "Unable to execute pktd, aborting."
printf '%s\n' "Everything looks good - use ./bin/pktd to launch"

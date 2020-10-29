#!/bin/sh
die() { printf '%s\n' "$*" >&2; exit 1; }
export GO111MODULE=on
PKTD_GIT_ID=$(git describe --always --tags HEAD 2>/dev/null) || die "failed to run git describe"
if [ -n "${PKTD_GIT_ID:?}" ] && ! git diff --ignore-cr-at-eol --quiet; then
    if [ -n "${PKT_FAIL_DIRTY:-}" ]; then
        printf '%s\n' "Build is dirty, failing" >&2
        git diff --ignore-cr-at-eol
        exit 1;
    fi
    PKTD_GIT_ID="${PKTD_GIT_ID:?}-dirty"
fi
PKTD_LDFLAGS="-X github.com/pkt-cash/pktd/pktconfig/version.appBuild=${PKTD_GIT_ID:?}"

mkdir -p ./bin || die "failed to create output directory"
printf '%s\n' "Building pktd"
go build -ldflags="${PKTD_LDFLAGS:?}" -o ./bin/pktd || die "failed to build pktd"
printf '%s\n' "Building wallet"
go build -ldflags="${PKTD_LDFLAGS:?}" -o ./bin/pktwallet ./pktwallet || die "failed to build wallet"
printf '%s\n' "Building btcctl"
go build -ldflags="${PKTD_LDFLAGS:?}" -o ./bin/pktctl ./cmd/btcctl || die "failed to build pktctl"
printf '%s\n' "Running tests"
go test ./... || die "tests failed"
if [ -z "${SKIP_GOLEVELDB_TESTS:-}" ]; then { { cd goleveldb; go test ./... || die "tests failed"; } && cd ..; }; fi
./bin/pktd --version || die "can't run pktd"
printf '%s\n' "Everything looks good - use ./bin/pktd to launch"

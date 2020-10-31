#!/bin/sh
die() { echo $1; exit 1; }
export GO111MODULE=on
PKTD_GIT_ID=$(git describe --tags HEAD)
if ! git diff --quiet; then
    if test "x$PKT_FAIL_DIRTY" != x; then
        echo "Build is dirty, failing"
        git diff
        exit 1;
    fi
    PKTD_GIT_ID="${PKTD_GIT_ID}-dirty"
fi
PKTD_LDFLAGS="-X github.com/pkt-cash/pktd/pktconfig/version.appBuild=${PKTD_GIT_ID}"

mkdir -p ./bin
echo "Building pktd"
go build -ldflags="${PKTD_LDFLAGS}" -o ./bin/pktd || die "failed to build pktd"
echo "Building wallet"
go build -ldflags="${PKTD_LDFLAGS}" -o ./bin/pktwallet ./pktwallet || die "failed to build wallet"
echo "Building btcctl"
go build -ldflags="${PKTD_LDFLAGS}" -o ./bin/pktctl ./cmd/btcctl || die "failed to build pktctl"
echo "Running tests"
go test ./... || die "tests failed"
if [ -z "${SKIP_GOLEVELDB_TESTS:-}" ]; then 
	{ { cd goleveldb; go test ./... || die "tests failed"; } && cd ..; }; 
fi
./bin/pktd --version || die "can't run pktd"
echo "Everything looks good - use ./bin/pktd to launch"

#!/usr/bin/env bash
LANG=C LC_ALL=C
die() { printf '%s\n' "${1}" 1>&2; exit 1; };
usage() {
if [ "${#}" -lt 1 ]; then
	printf '\n%s\n\n' "Usage: ./do [ARGS]"
	printf '%s\n' "Build Options:"
	printf '%s\n' " -v	Enable verbose compiler output"
	printf '%s\n' " -x	Enable debugging compiler output"
	printf '%s\n\n' " -f	Fully rebuild all source packages"
	printf '%s\n' "Build Modes:"
	printf '%s\n' " -d	Go defaults (default mode if build mode not specified) **"
	printf '%s\n' " -p	pure, attempt static-linked Pure Go-only build (includes symbols)"
	printf '%s\n' " -r	hard, Full RELRO/PIE/FORTIFY/ASLR/SSP/NX build (no symbols)"
	printf '\n%s\n\n' " **	Defaults often between distribution-provided Golang packages!"
	exit 1
fi
}

GCFLAGS="-trimpath"

while getopts ":xrvfpdha" option
do
  case "$option"
  in
  r) export ARGS="-buildmode=pie $ARGS"; export CGO_ENABLED=1; export CGO_CPPFLAGS="${CGO_CPPFLAGS:-} -O2 -D_FORTIFY_SOURCE=2 -fstack-protector-all -fstack-clash-protection"; export CGO_CFLAGS="${CGO_CFLAGS:-} -O2 -D_FORTIFY_SOURCE=2 -fstack-protector-all -fstack-clash-protection"; export CGO_CPPFLAGS="${CGO_CPPFLAGS:-} -O2 -D_FORTIFY_SOURCE=2 -fstack-protector-all -fstack-clash-protection"; export CGO_CXXFLAGS="${CGO_CXXFLAGS:-} -O2 -D_FORTIFY_SOURCE=2 -fstack-protector-all -fstack-clash-protection"; export CGO_FFLAGS="${CGO_FFLAGS:-} -O2 -D_FORTIFY_SOURCE=2"; export CGO_LDFLAGS="${CGO_LDFLAGS:-} -O2 -D_FORTIFY_SOURCE=2"; export CGO_LDFLAGS="${CGO_LDFLAGS:-} -s -w -Wl,-z,relro,-z,now"; export LD_FLAGS=" -s -w -extldflags=-Wl,-O1,--sort-common,--as-needed,-z,relro,-z,now,--build-id=none "; export BUILDMODE="${BUILDMODE}R"; export LINKMODE="-linkmode=external -buildid= ";;
  v) ARGS="$ARGS -v";;
  x) ARGS="$ARGS -x";;
  f) ARGS="$ARGS -a";;
  p) ARGS="$ARGS " CGO_ENABLED=0 GODEBUG=netdns=go BUILD_TAGS="-tags=netgo,osusergo" BUILDMODE="${BUILDMODE}P";;
  d) ARGS="$ARGS " BUILDMODE="${BUILDMODE}D";;
  h) usage;;
 \?) { printf '%s\n' "./do: Invalid option -$OPTARG; see \"./do -h\" for usage." 1>&2 ; exit 1; }; ;;
  :) usage;;
  *) usage;;
  esac
done

if [ -z "${ARGS}" ]; then
	usage
fi

MODESPEC=${#BUILDMODE}
if [ "${MODESPEC}" -gt 1 ]; then
	die "./do: Error: Specify only one Build Mode; see \"./do -h\" for usage."
fi

export GO111MODULE=on
export PKT_FAIL_DIRTY=1
PKTD_GIT_ID=$(git describe --tags HEAD) || { die "Unable to execute git."; };
if ! git diff --quiet; then
    if [ "x$PKT_FAIL_DIRTY" != "x" ]; then
		git --no-pager diff
		printf '\n%s\n' "ERROR: Build is dirty." "Modifications displayed above; aborting."
		exit 1;
    fi
    PKTD_GIT_ID="${PKTD_GIT_ID}-dirty"
fi
PKTD_LDFLAGS="${LINKMODE}-X github.com/pkt-cash/pktd/pktconfig/version.appBuild=${PKTD_GIT_ID}${LD_FLAGS}"

if [ ! -d "./bin" ]; then
	mkdir -p ./bin ||\
		die "mkdir failed; exiting."
fi

printf '%s\n' "Building pktd"
go build ${ARGS} ${BUILD_TAGS} -ldflags "${PKTD_LDFLAGS}" ${GCFLAGS} -o ./bin/pktd . ||\
	die "Failed to build pktd"

printf '%s\n' "Building pktwallet"
go build ${ARGS} ${BUILD_TAGS} -ldflags "${PKTD_LDFLAGS}" ${GCFLAGS} -o ./bin/pktwallet ./pktwallet ||\
	die "Failed to build pktwallet"

printf '%s\n' "Building pktctl"
go build ${ARGS} ${BUILD_TAGS} -ldflags "${PKTD_LDFLAGS}" ${GCFLAGS} -o ./bin/pktctl ./cmd/btcctl ||\
	die "Failed to build pktctl"

printf '%s\n' "Running tests"
go test ./... ||\
	die "Tests failed"

./bin/pktd --version ||\
	die "Error: Couldn't run compiled pktd"

printf '%s\n' "Build completed; output in ./bin"

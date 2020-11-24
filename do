#!/usr/bin/env sh
die() { printf '%s\n' "Error: ${*:?}" >&2; exit 1; }
build() { l="${1:-}"; printf '%s\n' "Building ${l:?${e_unset:?}}"; o=$(printf '%s\n' "${l:?${e_unset:?}}"|sed 's/^pktd$/./'); go build -o "${bindir:?${e_unset:?}}"/"${l?${e_unset:?}}" -trimpath -ldflags="${PKTD_LDFLAGS:?${e_unset:?}}" "./${o?${e_unset:?}}" || die "Failed building ${l?${e_unset:?}}"; }
export GO111MODULE="on" && export e_unset="Error: Variable is e_unset; aborting."
if [ -n "${LONG_TESTS:-}" ]; then { export long_tests=",long_tests"; }; fi
export bindir="./bin" && export PKTD_TESTFLAGS="-count=1 -cover -parallel=1 -tags osnetgo,osusergo${long_tests}"
PKTD_GIT_ID=$(git update-index -q --refresh 2>/dev/null; git describe --tags HEAD 2>/dev/null)
if ! git diff --quiet 2>/dev/null; then
    if [ -n "${PKT_FAIL_DIRTY:-}" ]; then { git diff 2>/dev/null; die "Build is dirty, aborting."; }; fi
    export PKTD_GIT_ID="${PKTD_GIT_ID:?${e_unset:?}}-dirty"
fi
export PKTD_LDFLAGS="-X github.com/pkt-cash/pktd/pktconfig/version.appBuild=${PKTD_GIT_ID:?${e_unset:?}}"
mkdir -p "${bindir:?${e_unset:?}}" || die "Failed to create output directory; aborting."
build pktd && build pktwallet && build pktctl
printf '%s\n' "Running tests"; # shellcheck disable=SC2086,SC2046
go test ${PKTD_TESTFLAGS:?${e_unset:?}} $(go list ./... | grep -v test | sort | uniq) || die "One or more tests failed."
"${bindir?${e_unset:?}}/pktd" --version || die "Unable to run compiled pktd executable."; # shellcheck disable=SC2250
printf '%s\n' "Success! $( (cd "${bindir:?${e_unset:?}}" 2>/dev/null && d=$(pwd -P 2>/dev/null) && printf '%s\n' "Compiled output is located at ${d:?${bindir:?$e_unset}}." 2>/dev/null) 2>/dev/null )"


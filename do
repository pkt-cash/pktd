#!/usr/bin/env sh
# shellcheck disable=SC2015

just() {
	"$@" || true :
}

say() {
	printf '%s\n' "${*}" ||\
		just printf '%s\n' ""
}

die() {
	just say "Error ${?}: ${*}" >&2
	exit 1;
}

try() {
	"$@" ||\
		{ die "${*}"; };
}

doBuild() {
	if [ "${#}" -lt 2 ]; then
		just say "doBuild(null): Malformed input."
		return 1
	else
		target="${1:?}"
		output="${2:?}"
		just say "Building ${output:?}"
		${doRun} go build "-o" "${output:?}" "-trimpath" "-ldflags=${PKTD_LDFLAGS:?} -buildid= -linkmode=auto -extldflags=-static" "-tags=osusergo,netgo,static_build" "${target:?}" ||\
			{ die "doBuild(${target:?}): Go build failed."
				return 1; };
	fi
}

if command stdbuf -oL true 1>/dev/null 2>&1; then
	doRun="stdbuf -oL command" &&\
		export doRun ||\
			{ die "Could not export doRun variable."; };
else
	doRun="command" &&\
		export DoRun ||\
			{ die "Could not export doRun variable."; };
fi

if say "X" |\
	${doRun} grep -q --line-buffered "X" 1>/dev/null 2>&1; then
	lineBuffered="--line-buffered" &&\
		export lineBuffered ||\
			{ die "Could not export lineBuffered variable."; };
fi

GO111MODULE="on" &&\
	export GO111MODULE ||\
		{ die "Could not export GO111MODULE variable."; }; 

CGO_ENABLED=0 &&\
	export CGO_ENABLED ||\
		{ die "Could not export CGO_ENABLED variable."; };

LC_ALL=C &&\
	export LC_ALL ||\
		{ die "Could not export LC_ALL variable."; };

BINDIR="./bin" &&\
	export BINDIR ||\
		{ die "Could not export BINDIR variable."; };

if ! ${doRun} git version 1>/dev/null 2>&1; then
	die "Could not execute \"git version\" command."
fi

if ! ${doRun} go version 1>/dev/null 2>&1; then
	die "Could not execute \"go version\" command."
fi

PKTD_GIT_ID=$(${doRun} git describe --tags HEAD --always) ||\
	die "\"git describe\" failed. Unable to set PKTD_GIT_ID."
export PKTD_GIT_ID ||\
	die "Could not export PKTD_GIT_ID variable."

if ! ${doRun} git diff --ignore-cr-at-eol --quiet 1>/dev/null 2>&1; then
	PKTD_GIT_ID="${PKTD_GIT_ID:?}-dirty" &&\
		export PKTD_GIT_ID ||\
			{ die "Could not export PKTD_GIT_ID variable."; };
	PKT_DIRTY_OUTPUT=$(${doRun} git diff --ignore-cr-at-eol -p 2>&1)
	if [ "${PKT_FAIL_DIRTY:=1}" -ne 0 ]; then
			if [ -n "${PKT_DIRTY_OUTPUT:-}" ]; then
				just say "${PKT_DIRTY_OUTPUT:?}"
				die "Build is dirty; changes shown above; aborting build."
			else
				die "Build is dirty; aborting build."
			fi
	fi
fi

PKTD_LDFLAGS="-X github.com/pkt-cash/pktd/pktconfig/version.appBuild=${PKTD_GIT_ID:?}" &&\
	export PKTD_LDFLAGS ||\
		{ die "Could not export PKTD_LDFLAGS variable."; };

mkdir -p "${BINDIR:?}" ||\
	die "Could not create output directory \"${BINDIR:?}\"; aborting build."

try doBuild "." "${BINDIR}/pktd"
try doBuild "./pktwallet" "${BINDIR}/pktwallet"
try doBuild "./cmd/btcctl" "${BINDIR}/pktctl"

test_timeout="15s"
just say "Testing (${test_timeout} timeout)"
${doRun} go test -timeout="${test_timeout}" -count=1 ./... |\
	${doRun} grep "${lineBuffered:-}" -v '\[no test files\]' |\
		${doRun} grep "${lineBuffered:-}" -F -e '' -e '' -e 'FAIL' &&\
			{ false && die "One or more tests failed."; };
${doRun} "${BINDIR}"/pktd --version ||\
	die "Failed to execute compiled pktd binary." &&\
		just say "Successful build!"

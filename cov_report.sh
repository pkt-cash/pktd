#!/usr/bin/env sh

# This script will use your default ${GOROOT}
# and ${GOPATH}, so ensure these locations exist,
# and "${GOPATH}/bin" is in your default ${PATH}.

# NOTE: This script is known to work on bash, zsh,
# and ksh (ksh93 and ksh2000+), but it is not
# POSIX compliant, due to the using the ksh-derived
# "type" utility. If you are not able to use this
# script, the tools are easily accessible and manual
# runs should be straight-forward on most platforms.

# Abort and inform in the case of csh or tcsh as sh.
# shellcheck disable=SC2046,SC2006,SC2116,SC2065
test _`echo asdf 2>/dev/null` != _asdf >/dev/null &&\
    printf '%s\n' "Error: csh as sh is unsupported." &&\
    exit 1

cleanUp() {
	printf '\n%s\n' "Running cleanup tasks." >&2 || true :
	set +u >/dev/null 2>&1 || true :
	set +e >/dev/null 2>&1 || true :
	rm -f ./gocov_report_goleveldb.json 2>&1 || true :
	rm -f ./gocov_report_goleveldb.json 2>&1 || true :
	rm -f ./gocov_report_pktd.txt >/dev/null 2>&1 || true :
	rm -f ./gocov_report_goleveldb.txt >/dev/null 2>&1 || true :
	rm -f ./gocov_report_pktd.html >/dev/null 2>&1 || true :
	rm -f ./gocov_report_goleveldb.html >/dev/null 2>&1 || true:
	printf '%s\n' "All cleanup tasks completed." >&2 || true :
	printf '%s\n' "" || true :
}

global_trap() {
    err=${?}
    trap - EXIT; trap '' EXIT INT TERM ABRT ALRM HUP
    cleanUp
}
trap 'global_trap $?' EXIT
trap 'err=$?; global_trap; exit $?' ABRT ALRM HUP TERM
trap 'err=$?; trap - EXIT; global_trap $err; exit $err' QUIT
trap 'global_trap; trap - INT; kill -INT $$; sleep 1; trap - TERM; kill -TERM $$' INT
trap '' EMT IO LOST SYS URG >/dev/null 2>&1 || true :

set -o pipefail >/dev/null 2>&1
if [ ! -f "./.pktd_root" ]; then
	printf '\n%s\n' "You must run this tool from the root"           >&2
	printf '%s\n'   "directory of the pktd source tree."             >&2
	exit 1 || :;
fi

export CGO_ENABLED=0
export TEST_FLAGS='-count=1 -cover -parallel=1'
export GOFLAGS='-tags=osnetgo,osusergo'
# shellcheck disable=SC2155
export PKT_TARGETS="$(go list ./... | sort | uniq)"
# shellcheck disable=SC2155
export LDB_TARGETS="$(cd goleveldb/leveldb && go list ./... | sort | uniq)"

type gocov 1>/dev/null 2>&1; # shellcheck disable=SC2181
if [ "${?}" -ne 0 ]; then
	printf '\n%s\n' "This script requires the gocov tool."           >&2
	printf '%s\n'   "You may obtain it with the following command:"  >&2
	printf '%s\n\n' "\"go get github.com/axw/gocov/gocov\""          >&2
	exit 1 || :;
fi

cleanUp || true && \
unset="Error: Testing flags are unset, aborting." &&\
	export unset
# shellcheck disable=SC2086
(date 2>/dev/null; gocov test ${TEST_FLAGS:?${unset:?}} ${PKT_TARGETS} > gocov_report_pktd.json && \
	gocov report < gocov_report_pktd.json > gocov_report_pktd.txt) || \
	{ printf '\n%s\n' "gocov failed complete pktd successfully." >&2
		exit 1 || :; };
# shellcheck disable=SC2086
(date 2>/dev/null; cd goleveldb/leveldb && \
	gocov test ${TEST_FLAGS:?${unset:?}} ${LDB_TARGETS} > ../../gocov_report_goleveldb.json && \
	gocov report < ../../gocov_report_goleveldb.json > ../../gocov_report_goleveldb.txt) || \
	{ printf '\n%s\n' "gocov failed to complete goleveldb successfully." >&2
		exit 1 || :; };

type gocov-html 1>/dev/null 2>&1; # shellcheck disable=SC2181
if [ "${?}" -ne 0 ]; then
	printf '%\n%s\n' "This script optionally utilizes gocov-html." >&2
    printf '%s\n'    "You may obtain it with the following command:" >&2
    printf '%s\n\n'  "\"go get https://github.com/matm/gocov-html\"" >&2
	exit 1 || :;
fi
(gocov-html < gocov_report_pktd.json > gocov_report_pktd.html) || \
	{ printf '\n%s\n' "gocov-html failed to complete pktd successfully." >&2
		exit 1 || :; };

printf '%s\n' "" >&2

(cd goleveldb/leveldb;\
	gocov-html < ../../gocov_report_goleveldb.json > ../../gocov_report_goleveldb.html) || \
	{ printf '\n%s\n' "gocov-html failed to complete goleveldb successfully." >&2
		exit 1 || :; };

printf '\n%s\n' "" >&2

mkdir -p ./cov && mv -f gocov_report_* ./cov && \
printf '\n%s\n' "Done - output is located at ./cov"

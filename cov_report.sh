#!/usr/bin/env sh
# This script uses the default ${GOROOT} and ${GOPATH},
# so ensure these locations exist and "${GOPATH}/bin" is
# in your ${PATH}. NOTE: This script is known to work on
# bash, zsh, and ksh (mksh, ksh93, and ksh2000+), but is
# not POSIX-compliant, due to the use of kshisms such as
# the "type" command. If you can't use this script, the
# tools are accessible and manual runs straight-forward.
# Set TEST_LND to include LND; TEST_LDB includes LevelDB.
# shellcheck disable=SC2046,SC2006,SC2116,SC2065
test _$(echo asdf 2> /dev/null) != _asdf > /dev/null &&
	printf '%s\n' "$(date): Error: csh as sh is unsupported." && exit 1
cleanUp() {
	printf '%s\n' "$(date): Running cleanup tasks." >&2 || true
	set +u > /dev/null 2>&1 || true && { set +e > /dev/null 2>&1 || true; }
	rm -f -- ./gocov_report_pktd.* > /dev/null 2>&1 || true
	printf '%s\n' "$(date): All cleanup tasks completed." >&2 || true
	set -u > /dev/null 2>&1 || true && { set -e > /dev/null 2>&1 || true; }
}
global_trap() {
	err=${?}
	trap - EXIT
	trap '' EXIT INT TERM ABRT ALRM HUP
	cleanUp
}
trap 'global_trap $?' EXIT
trap 'err=$?; global_trap; exit $?' ABRT ALRM HUP TERM
trap 'err=$?; trap - EXIT; global_trap $err; exit $err' QUIT
trap 'global_trap; trap - INT; kill -INT $$; sleep 1; trap - TERM; kill -TERM $$' INT
trap '' EMT IO LOST SYS URG > /dev/null 2>&1 || true
set -o pipefail > /dev/null 2>&1
set -e > /dev/null 2>&1
set -u > /dev/null 2>&1
# shellcheck disable=SC2236
if [ -n "${TEST_LND:-}" ] && [ ! -z "${TEST_LND:-}" ]; then
	LNDTEST_FLAGS='/lnd/' && export LNDTEST_FLAGS
else
	LNDTEST_FLAGS="//////////" && export LNDTEST_FLAGS
fi
# shellcheck disable=SC2236
if [ -n "${TEST_LDB:-}" ] && [ ! -z "${TEST_LDB:-}" ]; then
	LDBTEST_FLAGS='/goleveldb/' && export LDBTEST_FLAGS
else
	LDBTEST_FLAGS="//////////" && export LDBTEST_FLAGS
fi
if [ ! -f "./.pktd_root" ]; then
	printf '%s\n' "You must run this tool from the root" >&2
	printf '%s\n' "directory of the pktd source tree." >&2
	exit 1 || true
fi
export CGO_ENABLED=0
export TEST_FLAGS='-count=1 -cover -cpu=1 -parallel=1 -covermode=atomic -trimpath'
export GOFLAGS='-tags=osnetgo,osusergo'
type gocov 1> /dev/null 2>&1
# shellcheck disable=SC2181
if [ "${?}" -ne 0 ]; then
	printf '%s\n' "$(date): Error: This script requires the gocov tool." >&2
	printf '%s\n' "$(date):    You may obtain gocov with the following command:" >&2
	printf '%s\n' "$(date):     \"go get github.com/axw/gocov/gocov\"" >&2
	exit 1 || true
fi
cleanUp || true && unset="$(date): Error: Testing flags are unset, aborting." && export unset
TEST_TARGETS=$(go list ./... 2> /dev/null | grep -v test 2> /dev/null | grep -v "${LDBTEST_FLAGS:?}" 2> /dev/null | grep -v "${LNDTEST_FLAGS:?}" 2> /dev/null | sort 2> /dev/null | uniq 2> /dev/null)
if [ -t 1 ]; then
	if [ "0$(
		ccze -A < /dev/null > /dev/null 2>&1
		printf %s ${?:?}
	)" -ne 0 ]; then
		CCZE="cat" && export CCZE
	else
		CCZE="ccze -A" && export CCZE
	fi
	if ! (printf %s "XYXY" 2> /dev/null | grep -q --line-buffered 2> /dev/null); then
		LINEBUFFER="--line-buffered" && export LINEBUFFER
	else
		LINEBUFFER="" && export LINEBUFFER
	fi
else
	CCZE="cat" && export CCZE && LINEBUFFER="" && export LINEBUFFER
fi
# shellcheck disable=SC2086,SC2015
printf '%s\n' "$(date): Starting testing with coverage examination." && ( ( (gocov test ${TEST_FLAGS:?${unset:?}} ${TEST_TARGETS:?${unset:?}} 1> gocov_report_pktd.json) 2>&1 | grep -v ${LINEBUFFER} 'no test files' | ${CCZE}) 2>&1 | awk '{ system("printf \"%s\" \"$(date)\""); print ": "$0 }' 2>&1) 2>&1 && printf '%s\n' "$(date): Completed test and coverage run." && printf '%s\n' "$(date): Starting coverage analysis." && gocov report < gocov_report_pktd.json > gocov_report_pktd.txt || {
	printf '%s\n' "$(date): Error: gocov failed complete pktd successfully." >&2
	exit 1 || true
}
printf '%s\n' "$(date): Completed coverage analysis."
type gocov-html 1> /dev/null 2>&1
# shellcheck disable=SC2181
if [ "${?}" -ne 0 ]; then
	printf '%s\n' "$(date): This requires gocov-html to produce final HTML output." >&2
	printf '%s\n' "$(date):    You may obtain gocov-html with the following command:" >&2
	printf '%s\n' "$(date):     \"go get github.com/matm/gocov-html\"" >&2
	exit 1 || true
fi
gocov-html < gocov_report_pktd.json > gocov_report_pktd.html || {
	printf '%s\n' "$(date): Error: gocov-html failed to complete pktd successfully." >&2
	exit 1 || true
}
mkdir -p ./cov && mv -f gocov_report_* ./cov && printf '%s\n' "$(date): Process complete: output located at ./cov"

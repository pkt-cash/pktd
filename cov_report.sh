#!/usr/bin/env sh
cleanUp() { err="${1}"; touch "./gocov_report_pktd.$(mktemp -u XXXXXXXX)" || true;
			rm -f ./gocov_report_pktd.* >/dev/null 2>&1 || true;
			if [ "${#err}" -gt 0 ]; then exit "${err}"; else return; fi; };
global_trap() {	err="${?}"; trap - EXIT; trap '' EXIT INT TERM QUIT HUP; cleanUp "${err}"; };
trap 'global_trap $?' EXIT; trap 'err=$?; global_trap; exit $?' HUP TERM;
trap 'err=$?; trap - EXIT global_trap $err; exit $err' QUIT;
trap 'global_trap; trap - INT; kill -INT $$; sleep 1; trap - TERM; kill -TERM $$' INT; # shellcheck disable=SC2154,SC2236
if [ -z "${TEST_LND}" ]; then LNDTEST='lnd' && export LNDTEST && LGTTEST='lightning' && export LGTTEST;
	else LNDTEST="_X_X_X_X_" && export LNDTEST && LGTTEST="_X_X_X_X_" && export LGTTEST; fi; # shellcheck disable=SC2154,SC2236
if [ -z "${TEST_LDB}" ]; then LDBTEST='goleveldb' && export LDBTEST;
	else LDBTEST="_X_X_X_X_" && export LDBTEST; fi;
if [ ! -f "./.pktd_root" ]; then printf '%s\n' "Error: Must execute in the pktd source directory."; exit 1; fi;
CGO_ENABLED=0 && export CGO_ENABLED; # shellcheck disable=SC2089,SC2090
TESTF='-tags="osnetgo,osusergo,leaktest" -count=1 -cpu=1 -parallel=1 -covermode=atomic -trimpath' && export TESTF;
cleanUp || true; GOTAR=$(command go list ./... |
	grep -vE \'\(test\|"${LDBTEST}"\|"${LGTTEST}"\|"${LNDTEST}"\)\' | sort | uniq); # shellcheck disable=SC2090,SC2086
(gocov test ${TESTF} ${GOTAR} 1>gocov_report_pktd.json) 2>&1 |
	tr '\t' ' ' | tr -s ' ' | sed -e 's/github.com\/pkt-cash\/pktd\///g' -e 's/github.com\/pkt-cash\///g'
gocov report <gocov_report_pktd.json >gocov_report_pktd.txt ||
	{ printf '%s\n' "Error: gocov failed to complete."; exit 1; };
gocov-html <gocov_report_pktd.json >gocov_report_pktd.html ||
	{ printf '%s\n' "Error: gocov-html failed to complete."; exit 1; };

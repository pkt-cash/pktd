#!/usr/bin/env sh

# This script should be run from the project root
# e.g. ./contrib/freebsd/build.sh

BINARY_FOLDER=.
FREEBSD_PACKAGE_NAME=pktd

./do
echo "Binary built. Building TXZ now."

if which fpm; then
	fpm -n $FREEBSD_PACKAGE_NAME -s dir -t freebsd $BINARY_FOLDER
	echo "FreeBSD TXZ image built."
else
	echo "fpm not installed or not reachable"
	exit 1
fi

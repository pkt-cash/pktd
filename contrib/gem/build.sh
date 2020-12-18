#!/usr/bin/env sh


# This script should be run from the project root
# e.g. ./contrib/gem/build.sh


BINARY_FOLDER=.
RPM_PACKAGE_NAME=pktd

./do
echo "Binary built. Building GEM now."

if which fpm; then
	if which rpmbuild; then
		fpm -n $RPM_PACKAGE_NAME -s dir -t gem $BINARY_FOLDER
		echo "GEM file built."
	else
		echo "rpmbuild not installed or not reachable"
		exit 1
	fi
else
	echo "fpm not installed or not reachable"
	exit 1
fi

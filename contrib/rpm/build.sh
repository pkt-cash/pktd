#!/usr/bin/env sh

# This script should be run from the project root
# e.g. ./contrib/rpm/build.sh

if which fpm; then
	if which rpmbuild; then
		fpm -n pktd-linux -s dir -t rpm -v "$(./bin/pktctl --version | sed 's/.* version //' | tr -d '\n')" ./bin
		echo "RPM file built."
	else
		echo "rpmbuild not installed or not reachable"
		exit 1
	fi
else
	echo "fpm not installed or not reachable"
	exit 1
fi

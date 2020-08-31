#!/usr/bin/env sh

# swap imports between syndtr and johnsonjh

grep -q pkt-cash README.md && find . -path ./fixup-imports.sh -prune -o -type f -print0 | xargs -0 sed -i 's#github.com/pkt-cash/pktd/goleveldb#github.com/syndtr/goleveldb#g' && exit

grep -q syndtr README.md && find . -path ./fixup-imports.sh -prune -o -type f -print0 | xargs -0 sed -i 's#github.com/syndtr/goleveldb#github.com/pkt-cash/pktd/goleveldb#g' && exit

# ffldb

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://Copyfree.org)

Package ffldb implements a driver for the database package that uses leveldb for
the backing metadata and flat files for block storage.

This driver is the recommended driver for use with pktd. It makes use of
GoLevelDB for storing metadata, flat files for block storage, and uses
checksums in key areas for data integrity.

Package ffldb is licensed under the Copyfree ISC license.

## License

Package ffldb is licensed under the [Copyfree](http://Copyfree.org) ISC
License.

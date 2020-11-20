// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

/*
Package ffldb implements the block database used by pktd. It uses GoLevelDB
for storage of metadata, raw flat files for block storage, and has checksums
in key areas to ensure data integrity.

Package ffldb is licensed under the Copyfree ISC license.

*/
package ffldb

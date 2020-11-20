// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ffldb

import (
	"github.com/pkt-cash/pktd/pktlog"
)

var log = pktlog.Disabled

const (
	dbType = "ffldb"
)

// useLogger is the callback provided during driver registration that sets the
// current logger to the provided one.
func useLogger(logger pktlog.Logger) {
	log = logger
}

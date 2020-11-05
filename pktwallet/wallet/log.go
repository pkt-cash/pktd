// Copyright (c) 2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"github.com/pkt-cash/pktd/pktlog"
	"github.com/pkt-cash/pktd/pktwallet/waddrmgr"
	"github.com/pkt-cash/pktd/pktwallet/walletdb/migration"
	"github.com/pkt-cash/pktd/pktwallet/wtxmgr"
)

// log is a logger that is initialized with no output filters.  This
// means the package will not perform any logging by default until the caller
// requests it.
var log pktlog.Logger

// The default amount of logging is none.
func init() {
	DisableLog()
}

// DisableLog disables all library log output.  Logging output is disabled
// by default until either UseLogger or SetLogWriter are called.
func DisableLog() {
	UseLogger(pktlog.Disabled)
}

// UseLogger uses a specified Logger to output package logging info.
// This should be used in preference to SetLogWriter if the caller is also
// using pktlog.
func UseLogger(logger pktlog.Logger) {
	log = logger

	migration.UseLogger(logger)
	waddrmgr.UseLogger(logger)
	wtxmgr.UseLogger(logger)
}

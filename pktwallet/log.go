// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/pkt-cash/pktd/addrmgr"
	"github.com/pkt-cash/pktd/blockchain"
	"github.com/pkt-cash/pktd/blockchain/indexers"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/block"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/block/proof"
	"github.com/pkt-cash/pktd/connmgr"
	"github.com/pkt-cash/pktd/mempool"
	"github.com/pkt-cash/pktd/mining"
	"github.com/pkt-cash/pktd/mining/cpuminer"
	"github.com/pkt-cash/pktd/netsync"
	"github.com/pkt-cash/pktd/neutrino"
	"github.com/pkt-cash/pktd/peer"
	"github.com/pkt-cash/pktd/pktlog"
	"github.com/pkt-cash/pktd/pktwallet/chain"
	"github.com/pkt-cash/pktd/pktwallet/rpc/legacyrpc"
	"github.com/pkt-cash/pktd/pktwallet/wallet"
	"github.com/pkt-cash/pktd/pktwallet/wtxmgr"
	"github.com/pkt-cash/pktd/rpcclient"
	"github.com/pkt-cash/pktd/txscript"
)

// logWriter implements an io.Writer that outputs to both standard output and
// the write-end pipe of an initialized log rotator.
type logWriter struct{}

func (logWriter) Write(p []byte) (n int, err error) {
	_, err = os.Stdout.Write(p)
	if err != nil {
		panic("logWriter: os.Stdout.Write failure")
	}
	return len(p), nil
}

// Loggers per subsystem.  A single backend logger is created and all subsytem
// loggers created from it will write to the backend.  When adding new
// subsystems, add the subsystem logger variable here and to the
// subsystemLoggers map.
//
// Loggers can not be used before the log rotator has been initialized with a
// log file.  This must be performed early during application startup by calling
// initLogRotator.
var (
	// backendLog is the logging backend used to create all subsystem loggers.
	// The backend must not be used before the log rotator has been initialized,
	// or data races and/or nil pointer dereferences will occur.
	backendLog = pktlog.NewBackend(logWriter{})

	log          = backendLog.Logger("BTCW")
	walletLog    = backendLog.Logger("WLLT")
	txmgrLog     = backendLog.Logger("TMGR")
	chainLog     = backendLog.Logger("CHNS")
	grpcLog      = backendLog.Logger("GRPC")
	legacyRPCLog = backendLog.Logger("RPCS")
	btcnLog      = backendLog.Logger("BTCN")

	adxrLog = backendLog.Logger("ADXR")
	amgrLog = backendLog.Logger("AMGR")
	cmgrLog = backendLog.Logger("CMGR")
	bcdbLog = backendLog.Logger("BCDB")
	pktdLog = backendLog.Logger("BTCD")
	chanLog = backendLog.Logger("CHAN")
	discLog = backendLog.Logger("DISC")
	indxLog = backendLog.Logger("INDX")
	minrLog = backendLog.Logger("MINR")
	peerLog = backendLog.Logger("PEER")
	rpcsLog = backendLog.Logger("RPCS")
	scrpLog = backendLog.Logger("SCRP")
	srvrLog = backendLog.Logger("SRVR")
	syncLog = backendLog.Logger("SYNC")
	txmpLog = backendLog.Logger("TXMP")
	pcptLog = backendLog.Logger("PCPT")
)

// Initialize package-global logger variables.
func init() {
	wallet.UseLogger(walletLog)
	wtxmgr.UseLogger(txmgrLog)
	chain.UseLogger(chainLog)
	rpcclient.UseLogger(chainLog)
	legacyrpc.UseLogger(legacyRPCLog)
	neutrino.UseLogger(btcnLog)

	addrmgr.UseLogger(amgrLog)
	connmgr.UseLogger(cmgrLog)
	blockchain.UseLogger(chanLog)
	indexers.UseLogger(indxLog)
	mining.UseLogger(minrLog)
	cpuminer.UseLogger(minrLog)
	peer.UseLogger(peerLog)
	txscript.UseLogger(scrpLog)
	netsync.UseLogger(syncLog)
	mempool.UseLogger(txmpLog)
	block.UseLogger(pcptLog)
	proof.UseLogger(pcptLog)
}

// subsystemLoggers maps each subsystem identifier to its associated logger.
var subsystemLoggers = map[string]pktlog.Logger{
	"BTCW": log,
	"WLLT": walletLog,
	"TMGR": txmgrLog,
	"CHNS": chainLog,
	"GRPC": grpcLog,
	"RPCS": legacyRPCLog,
	"BTCN": btcnLog,

	"ADXR": adxrLog,
	"AMGR": amgrLog,
	"CMGR": cmgrLog,
	"BCDB": bcdbLog,
	"PKTD": pktdLog,
	"CHAN": chanLog,
	"DISC": discLog,
	"INDX": indxLog,
	"MINR": minrLog,
	"PEER": peerLog,
	"SCRP": scrpLog,
	"SRVR": srvrLog,
	"SYNC": syncLog,
	"TXMP": txmpLog,
	"PCPT": pcptLog,
}

// setLogLevel sets the logging level for provided subsystem.  Invalid
// subsystems are ignored.  Uninitialized subsystems are dynamically created as
// needed.
func setLogLevel(subsystemID string, logLevel string) {
	// Ignore invalid subsystems.
	logger, ok := subsystemLoggers[subsystemID]
	if !ok {
		return
	}

	// Defaults to info if the log level is invalid.
	level, _ := pktlog.LevelFromString(logLevel)
	logger.SetLevel(level)
}

// setLogLevels sets the log level for all subsystem loggers to the passed
// level.  It also dynamically creates the subsystem loggers as needed, so it
// can be used to initialize the logging system.
func setLogLevels(logLevel string) {
	// Configure all sub-systems with the new logging level.  Dynamically
	// create loggers as needed.
	for subsystemID := range subsystemLoggers {
		setLogLevel(subsystemID, logLevel)
	}
}

// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"time"

	"github.com/pkt-cash/pktd/blockchain/indexers"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/database"
	"github.com/pkt-cash/pktd/database/ffldb"
	"github.com/pkt-cash/pktd/limits"
	"github.com/pkt-cash/pktd/pktconfig/version"
	"github.com/arl/statsviz"
)

const (
	// blockDbNamePrefix is the prefix for the block database name.  The
	// database type is appended to this value to form the full block
	// database name.
	blockDbNamePrefix = "blocks"
)

var (
	cfg *config
)

// winServiceMain is only invoked on Windows.  It detects when pktd is running
// as a service and reacts accordingly.
var winServiceMain func() (bool, er.R)

// pktdMain is the real main function for pktd.  It is necessary to work around
// the fact that deferred functions do not run when os.Exit() is called.  The
// optional serverChan parameter is mainly used by the service code to be
// notified with the server once it is setup so it can gracefully stop it when
// requested from the service control manager.
func pktdMain(serverChan chan<- *server) er.R {
	// Load configuration and parse command line.  This function also
	// initializes logging and configures it accordingly.
	tcfg, _, err := loadConfig()
	if err != nil {
		return err
	}
	cfg = tcfg

	// Get a channel that will be closed when a shutdown signal has been
	// triggered either from an OS signal such as SIGINT (Ctrl+C) or from
	// another subsystem such as the RPC server.
	interrupt := interruptListener()
	defer pktdLog.Info("Shutdown complete")

	// Show version at startup.
	pktdLog.Infof("Version %s", version.Version())

	version.WarnIfPrerelease(pktdLog)

	// Enable http profiling server if requested.
	if cfg.Profile != "" {
		go func() {
			listenAddr := net.JoinHostPort("", cfg.Profile)
			pktdLog.Infof("Profile server listening on %s", listenAddr)
			profileRedirect := http.RedirectHandler("/debug/pprof",
				http.StatusSeeOther)
			http.Handle("/", profileRedirect)
			pktdLog.Errorf("%v", http.ListenAndServe(listenAddr, nil))
		}()
	}

	// Write cpu profile if requested.
	if cfg.CPUProfile != "" {
		f, errr := os.Create(cfg.CPUProfile)
		if errr != nil {
			pktdLog.Errorf("Unable to create cpu profile: %v", err)
			return er.E(errr)
		}
		if errp := pprof.StartCPUProfile(f); errp != nil {
            pktdLog.Errorf("could not start CPU profile: ", errp)
			return er.E(errp)
        }
		defer f.Close()
		defer pprof.StopCPUProfile()
	}

	// Enable StatsViz server if requested.
	if cfg.StatsViz != "" {
		statsvizAddr := net.JoinHostPort("", cfg.StatsViz)
		pktdLog.Infof("StatsViz server listening on %s", statsvizAddr)
		smux := http.NewServeMux()
		statsvizRedirect := http.RedirectHandler("/debug/statsviz", http.StatusSeeOther)
		smux.Handle("/", statsvizRedirect)
		if err := statsviz.Register(smux, statsviz.Root("/debug/statsviz")); err != nil {
			pktdLog.Errorf("%v", err)
		}
		go func() {
			pktdLog.Errorf("%v", http.ListenAndServe(statsvizAddr, smux))
		}()
	}

	// Perform upgrades to pktd as new versions require it.
	if err := doUpgrades(); err != nil {
		pktdLog.Errorf("%v", err)
		return err
	}

	// Return now if an interrupt signal was triggered.
	if interruptRequested(interrupt) {
		return nil
	}

	// Load the block database.
	db, err := loadBlockDB()
	if err != nil {
		pktdLog.Errorf("%v", err)
		return err
	}
	defer func() {
		// Ensure the database is sync'd and closed on shutdown.
		pktdLog.Infof("Gracefully shutting down the database...")
		db.Close()
	}()

	// Return now if an interrupt signal was triggered.
	if interruptRequested(interrupt) {
		return nil
	}

	// Drop indexes and exit if requested.
	//
	// NOTE: The order is important here because dropping the tx index also
	// drops the address index since it relies on it.
	if cfg.DropAddrIndex {
		if err := indexers.DropAddrIndex(db, interrupt); err != nil {
			pktdLog.Errorf("%v", err)
			return err
		}

		return nil
	}
	if cfg.DropTxIndex {
		if err := indexers.DropTxIndex(db, interrupt); err != nil {
			pktdLog.Errorf("%v", err)
			return err
		}

		return nil
	}
	if cfg.DropCfIndex {
		if err := indexers.DropCfIndex(db, interrupt); err != nil {
			pktdLog.Errorf("%v", err)
			return err
		}

		return nil
	}

	// Create server and start it.
	server, err := newServer(cfg.Listeners, cfg.AgentBlacklist,
		cfg.AgentWhitelist, db, activeNetParams.Params, interrupt)
	if err != nil {
		// TODO: this logging could do with some beautifying.
		pktdLog.Errorf("Unable to start server on %v: %v",
			cfg.Listeners, err)
		return err
	}
	defer func() {
		// Shut down in 2 minutes, or just pull the plug.
		const shutdownTimeout = 2 * time.Minute
		pktdLog.Infof("Attempting graceful shutdown (%s timeout)...", shutdownTimeout)
		server.Stop()
		shutdownDone := make(chan struct{})
		go func() {
			server.WaitForShutdown()
			shutdownDone <- struct{}{}
		}()

		select {
		case <-shutdownDone:
		case <-time.Tick(shutdownTimeout):
		pktdLog.Errorf("Graceful shutdown in %s failed - forcefully terminating in 5s...", shutdownTimeout)
		time.Sleep(5 * time.Second)
		panic("Forcefully terminating the server process...")
		}
		srvrLog.Infof("Server shutdown complete")
	}()

	server.Start()
	if serverChan != nil {
		serverChan <- server
	}

	// Wait until the interrupt signal is received from an OS signal or
	// shutdown is requested through one of the subsystems such as the RPC
	// server.
	<-interrupt
	return nil
}

// removeRegressionDB removes the existing regression test database if running
// in regression test mode and it already exists.
func removeRegressionDB(dbPath string) er.R {
	// Don't do anything if not in regression test mode.
	if !cfg.RegressionTest {
		return nil
	}

	// Remove the old regression test database if it already exists.
	fi, err := os.Stat(dbPath)
	if err == nil {
		pktdLog.Infof("Removing regression test database from '%s'", dbPath)
		if fi.IsDir() {
			errr := os.RemoveAll(dbPath)
			if errr != nil {
				return er.E(errr)
			}
		} else {
			errr := os.Remove(dbPath)
			if errr != nil {
				return er.E(errr)
			}
		}
	}

	return nil
}

// dbPath returns the path to the block database given a database type.
func blockDbPath(dbType string) string {
	// The database name is based on the database type.
	dbName := blockDbNamePrefix + "_" + dbType
	dbPath := filepath.Join(cfg.DataDir, dbName)
	return dbPath
}

// loadBlockDB loads (or creates when needed) the block database taking into
// account the selected database backend and returns a handle to it.  It also
// contains additional logic such warning the user if there are multiple
// databases which consume space on the file system and ensuring the regression
// test database is clean when in regression test mode.
func loadBlockDB() (database.DB, er.R) {
	// The database name is based on the database type.
	dbPath := blockDbPath("ffldb")

	// The regression test is special in that it needs a clean database for
	// each run, so remove it now if it already exists.
	removeRegressionDB(dbPath)

	pktdLog.Infof("Loading block database from '%s'", dbPath)
	db, err := ffldb.OpenDB(dbPath, activeNetParams.Net, false)
	if err != nil {
		// Return the error if it's not because the database doesn't exist.
		if !database.ErrDbDoesNotExist.Is(err) {
			return nil, err
		}

		// Create the db if it does not exist.
		errr := os.MkdirAll(cfg.DataDir, 0700)
		if errr != nil {
			return nil, er.E(errr)
		}
		db, err = ffldb.OpenDB(dbPath, activeNetParams.Net, true)
		if err != nil {
			return nil, err
		}
	}

	pktdLog.Info("Block database loaded")
	return db, nil
}

func main() {
	version.SetUserAgentName("pktd")
	runtime.GOMAXPROCS(runtime.NumCPU()*6)

	// Block and transaction processing can cause bursty allocations.  This
	// limits the garbage collector from excessively overallocating during
	// bursts.  This value was arrived at with the help of profiling live
	// usage.
	debug.SetGCPercent(10)

	// Up some limits.
	if err := limits.SetLimits(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set limits: %v\n", err)
		os.Exit(1)
	}

	// Call serviceMain on Windows to handle running as a service.  When
	// the return isService flag is true, exit now since we ran as a
	// service.  Otherwise, just fall through to normal operation.
	if runtime.GOOS == "windows" {
		isService, err := winServiceMain()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if isService {
			os.Exit(0)
		}
	}

	// Work around defer not working after os.Exit()
	if err := pktdMain(nil); err != nil {
		os.Exit(1)
	}
}

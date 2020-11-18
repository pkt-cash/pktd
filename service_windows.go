// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/pktconfig/version"

	"github.com/btcsuite/winsvc/eventlog"
	"github.com/btcsuite/winsvc/mgr"
	"github.com/btcsuite/winsvc/svc"
)

const (
	// svcName is the name of pktd service.
	svcName = "pktdsvc"

	// svcDisplayName is the service name that will be shown in the windows
	// services list.  Not the svcName is the "real" name which is used
	// to control the service.  This is only for display purposes.
	svcDisplayName = "PKTd Service"

	// svcDesc is the description of the service.
	svcDesc = "Synchronizes with the PKT blockchain" +
		"and provides access and services to applications."
)

// elog is used to send messages to the Windows event log.
var elog *eventlog.Log

// logServiceStartOfDay logs information about pktd when the main server has
// been started to the Windows event log.
func logServiceStartOfDay(srvr *server) {
	var message string
	message += fmt.Sprintf("Version %s\n", version.Version())
	message += fmt.Sprintf("Configuration directory: %s\n", defaultHomeDir)
	message += fmt.Sprintf("Configuration file: %s\n", cfg.ConfigFile)
	message += fmt.Sprintf("Data directory: %s\n", cfg.DataDir)

	elog.Info(1, message)
}

// pktdService houses the main service handler which handles all service
// updates and launching pktdMain.
type pktdService struct{}

// Execute is the main entry point the winsvc package calls when receiving
// information from the Windows service control manager.  It launches the
// long-running pktdMain (which is the real meat of pktd), handles service
// change requests, and notifies the service control manager of changes.
func (s *pktdService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	// Service start is pending.
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	// Start pktdMain in a separate goroutine so the service can start
	// quickly.  Shutdown (along with a potential error) is reported via
	// doneChan.  serverChan is notified with the main server instance once
	// it is started so it can be gracefully stopped.
	doneChan := make(chan er.R)
	serverChan := make(chan *server)
	go func() {
		err := pktdMain(serverChan)
		doneChan <- err
	}()

	// Service is now started.
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	var mainServer *server
loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus

			case svc.Stop, svc.Shutdown:
				// Service stop is pending.  Don't accept any
				// more commands while pending.
				changes <- svc.Status{State: svc.StopPending}

				// Signal the main function to exit.
				shutdownRequestChannel <- struct{}{}

			default:
				elog.Error(1, fmt.Sprintf("Unexpected control "+
					"request #%d.", c))
			}

		case srvr := <-serverChan:
			mainServer = srvr
			logServiceStartOfDay(mainServer)

		case err := <-doneChan:
			if err != nil {
				elog.Error(1, err.String())
			}
			break loop
		}
	}

	// Service is now stopped.
	changes <- svc.Status{State: svc.Stopped}
	return false, 0
}

// installService attempts to install the pktd service.  Typically this should
// be done by the msi installer, but it is provided here since it can be useful
// for development.
func installService() er.R {
	// Get the path of the current executable.  This is needed because
	// os.Args[0] can vary depending on how the application was launched.
	// For example, under cmd.exe it will only be the name of the app
	// without the path or extension, but under mingw it will be the full
	// path including the extension.
	exePath, errr := filepath.Abs(os.Args[0])
	if errr != nil {
		return er.E(errr)
	}
	if filepath.Ext(exePath) == "" {
		exePath += ".exe"
	}

	// Connect to the windows service manager.
	serviceManager, errr := mgr.Connect()
	if errr != nil {
		return er.E(errr)
	}
	defer serviceManager.Disconnect()

	// Ensure the service doesn't already exist.
	service, err := serviceManager.OpenService(svcName)
	if err == nil {
		service.Close()
		return er.Errorf("service %s already exists", svcName)
	}

	// Install the service.
	service, errr = serviceManager.CreateService(svcName, exePath, mgr.Config{
		DisplayName: svcDisplayName,
		Description: svcDesc,
	})
	if errr != nil {
		return er.E(errr)
	}
	defer service.Close()

	// Support events to the event log using the standard "standard" Windows
	// EventCreate.exe message file.  This allows easy logging of custom
	// messges instead of needing to create our own message catalog.
	eventlog.Remove(svcName)
	eventsSupported := uint32(eventlog.Error | eventlog.Warning | eventlog.Info)
	if errr := eventlog.InstallAsEventCreate(svcName, eventsSupported); errr != nil {
		return er.E(errr)
	}
	return nil
}

// removeService attempts to uninstall the pktd service.  Typically this should
// be done by the msi uninstaller, but it is provided here since it can be
// useful for development.  Not the eventlog entry is intentionally not removed
// since it would invalidate any existing event log messages.
func removeService() er.R {
	// Connect to the windows service manager.
	serviceManager, errr := mgr.Connect()
	if errr != nil {
		return er.E(errr)
	}
	defer serviceManager.Disconnect()

	// Ensure the service exists.
	service, err := serviceManager.OpenService(svcName)
	if err != nil {
		return er.Errorf("service %s is not installed", svcName)
	}
	defer service.Close()

	// Remove the service.
	if errr := service.Delete(); errr != nil {
		return er.E(errr)
	}
	return nil
}

// startService attempts to start the pktd service.
func startService() er.R {
	// Connect to the windows service manager.
	serviceManager, errr := mgr.Connect()
	if errr != nil {
		return er.E(errr)
	}
	defer serviceManager.Disconnect()

	service, err := serviceManager.OpenService(svcName)
	if err != nil {
		return er.Errorf("could not access service: %v", err)
	}
	defer service.Close()

	err = service.Start(os.Args)
	if err != nil {
		return er.Errorf("could not start service: %v", err)
	}

	return nil
}

// controlService allows commands which change the status of the service.  It
// also waits for up to 10 seconds for the service to change to the passed
// state.
func controlService(c svc.Cmd, to svc.State) er.R {
	// Connect to the windows service manager.
	serviceManager, errr := mgr.Connect()
	if errr != nil {
		return er.E(errr)
	}
	defer serviceManager.Disconnect()

	service, err := serviceManager.OpenService(svcName)
	if err != nil {
		return er.Errorf("could not access service: %v", err)
	}
	defer service.Close()

	status, err := service.Control(c)
	if err != nil {
		return er.Errorf("could not send control=%d: %v", c, err)
	}

	// Send the control message.
	timeout := time.Now().Add(10 * time.Second)
	for status.State != to {
		if timeout.Before(time.Now()) {
			return er.Errorf("timeout waiting for service to go "+
				"to state=%d", to)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = service.Query()
		if err != nil {
			return er.Errorf("could not retrieve service "+
				"status: %v", err)
		}
	}

	return nil
}

// performServiceCommand attempts to run one of the supported service commands
// provided on the command line via the service command flag.  An appropriate
// error is returned if an invalid command is specified.
func performServiceCommand(command string) er.R {
	var err er.R
	switch command {
	case "install":
		err = installService()

	case "remove":
		err = removeService()

	case "start":
		err = startService()

	case "stop":
		err = controlService(svc.Stop, svc.Stopped)

	default:
		err = er.Errorf("invalid service command [%s]", command)
	}

	return err
}

// serviceMain checks whether we're being invoked as a service, and if so uses
// the service control manager to start the long-running server.  A flag is
// returned to the caller so the application can determine whether to exit (when
// running as a service) or launch in normal interactive mode.
func serviceMain() (bool, er.R) {
	// Don't run as a service if we're running interactively (or that can't
	// be determined due to an error).
	isInteractive, errr := svc.IsAnInteractiveSession()
	if errr != nil {
		return false, er.E(errr)
	}
	if isInteractive {
		return false, nil
	}

	elog, errr = eventlog.Open(svcName)
	if errr != nil {
		return false, er.E(errr)
	}
	defer elog.Close()

	errr = svc.Run(svcName, &pktdService{})
	if errr != nil {
		elog.Error(1, fmt.Sprintf("Service start failed: %v", errr))
		return true, er.E(errr)
	}

	return true, nil
}

// Set windows specific functions to real functions.
func init() {
	runServiceCommand = performServiceCommand
	winServiceMain = serviceMain
}

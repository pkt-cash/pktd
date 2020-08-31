rpcclient
=========

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

rpcclient implements a Websocket-enabled JSON-RPC client package written
in [Go](http://golang.org/).  It provides a robust and easy to use client for
interfacing with an RPC server that uses the btcd/bitcoin/pktd-compatible
JSON-RPC API.

## Status

This package is currently under active development.  It is already stable and
the infrastructure is complete.  However, there are still several RPCs left to
implement and the API is not stable yet.

## Major Features

* Supports Websockets (pktd/pktwallet) and HTTP POST mode (bitcoin core)
* Provides callback and registration functions for pktd/pktwallet notifications
* Supports btcd/pktd extensions
* Translates to and from higher-level and easier to use Go types
* Offers a synchronous (blocking) and asynchronous API
* When running in Websockets mode (the default):
  * Automatic reconnect handling (can be disabled)
  * Outstanding commands are automatically reissued
  * Registered notifications are automatically reregistered
  * Back-off support on reconnect attempts

## License

Package rpcclient is licensed under the [copyfree](http://copyfree.org) ISC
License.

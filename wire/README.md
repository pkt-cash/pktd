wire
====

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://Copyfree.org)

Package wire implements the pktd wire protocol. A comprehensive suite of
tests with 100% test coverage is provided to ensure proper functionality.

This package has intentionally been designed so it can be used as a standalone
package for any projects needing to interface with pktd/bitcoin-like peers at 
the wire protocol level.

## Message Overview

The protocol consists of exchanging messages between peers. Each message is
preceded by a header which identifies information about it such as which 
network it is a part of, its type, how big it is, and a checksum to verify
validity. All encoding and decoding of message headers is handled by this
package.

To accomplish this, there is a generic interface for network messages named
`Message` which allows messages of any type to be read, written, or passed
around through channels, functions, etc. In addition, concrete implementations
of most of the currently supported network messages are provided. For these
supported messages, all of the details of marshaling and unmarshalling to and
from the wire using network encoding are handled so the caller doesn't have to
concern themselves with the specifics.

## Reading Messages Example

In order to unmarshal network messages from the wire, use the `ReadMessage`
function. It accepts any `io.Reader`, but typically this will be a `net.Conn`
to a remote node running a pktd peer.  

## Writing Messages Example

In order to marshal network messages to the wire, use the `WriteMessage`
function. It accepts any `io.Writer`, but typically this will be a `net.Conn`
to a remote node running a pktd peer. 

## License

Package wire is licensed under the [Copyfree](http://Copyfree.org) ISC
License.

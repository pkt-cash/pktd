// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"fmt"

	"github.com/pkt-cash/pktd/btcutil/er"
)

// MessageError describes an issue with a message.
// An example of some potential issues are messages from the wrong bitcoin
// network, invalid commands, mismatched checksums, and exceeding max payloads.
//
// This provides a mechanism for the caller to type assert the error to
// differentiate between general io errors such as io.EOF and issues that
// resulted from malformed messages.
var MessageError *er.ErrorCode = er.GenericErrorType.Code("wire.MessageError")

// messageError creates an error for the given function and description.
func messageError(f, desc string) er.R {
	return MessageError.New(fmt.Sprintf("%s: %s", f, desc), nil)
}

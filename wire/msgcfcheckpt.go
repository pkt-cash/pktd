// Copyright (c) 2018 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"errors"
	"fmt"
	"io"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
)

const (
	// CFCheckptInterval is the gap (in number of blocks) between each
	// filter header checkpoint.
	CFCheckptInterval = 1000

	// maxCFHeadersLen is the max number of filter headers we will attempt
	// to decode.
	maxCFHeadersLen = 100000
)

// ErrInsaneCFHeaderCount signals that we were asked to decode an
// unreasonable number of cfilter headers.
var ErrInsaneCFHeaderCount = errors.New(
	"refusing to decode unreasonable number of filter headers")

// MsgCFCheckpt implements the Message interface and represents a bitcoin
// cfcheckpt message.  It is used to deliver committed filter header information
// in response to a getcfcheckpt message (MsgGetCFCheckpt). See MsgGetCFCheckpt
// for details on requesting the headers.
type MsgCFCheckpt struct {
	FilterType    FilterType
	StopHash      chainhash.Hash
	FilterHeaders []*chainhash.Hash
}

// AddCFHeader adds a new committed filter header to the message.
func (msg *MsgCFCheckpt) AddCFHeader(header *chainhash.Hash) er.R {
	if len(msg.FilterHeaders) == cap(msg.FilterHeaders) {
		str := fmt.Sprintf("FilterHeaders has insufficient capacity for "+
			"additional header: len = %d", len(msg.FilterHeaders))
		return messageError("MsgCFCheckpt.AddCFHeader", str)
	}

	msg.FilterHeaders = append(msg.FilterHeaders, header)
	return nil
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgCFCheckpt) BtcDecode(r io.Reader, pver uint32, _ MessageEncoding) er.R {
	// Read filter type
	err := readElement(r, &msg.FilterType)
	if err != nil {
		return err
	}

	// Read stop hash
	err = readElement(r, &msg.StopHash)
	if err != nil {
		return err
	}

	// Read number of filter headers
	count, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}

	// Refuse to decode an insane number of cfheaders.
	if count > maxCFHeadersLen {
		return er.E(ErrInsaneCFHeaderCount)
	}

	// Create a contiguous slice of hashes to deserialize into in order to
	// reduce the number of allocations.
	msg.FilterHeaders = make([]*chainhash.Hash, count)
	for i := uint64(0); i < count; i++ {
		var cfh chainhash.Hash
		err := readElement(r, &cfh)
		if err != nil {
			return err
		}
		msg.FilterHeaders[i] = &cfh
	}

	return nil
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgCFCheckpt) BtcEncode(w io.Writer, pver uint32, _ MessageEncoding) er.R {
	// Write filter type
	err := writeElement(w, msg.FilterType)
	if err != nil {
		return err
	}

	// Write stop hash
	err = writeElement(w, msg.StopHash)
	if err != nil {
		return err
	}

	// Write length of FilterHeaders slice
	count := len(msg.FilterHeaders)
	err = WriteVarInt(w, pver, uint64(count))
	if err != nil {
		return err
	}

	for _, cfh := range msg.FilterHeaders {
		err := writeElement(w, cfh)
		if err != nil {
			return err
		}
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgCFCheckpt) Command() string {
	return CmdCFCheckpt
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver. This is part of the Message interface implementation.
func (msg *MsgCFCheckpt) MaxPayloadLength(pver uint32) uint32 {
	// Message size depends on the blockchain height, so return general limit
	// for all messages.
	return MaxMessagePayload
}

// NewMsgCFCheckpt returns a new bitcoin cfheaders message that conforms to
// the Message interface. See MsgCFCheckpt for details.
func NewMsgCFCheckpt(filterType FilterType, stopHash *chainhash.Hash,
	headersCount int) *MsgCFCheckpt {
	return &MsgCFCheckpt{
		FilterType:    filterType,
		StopHash:      *stopHash,
		FilterHeaders: make([]*chainhash.Hash, 0, headersCount),
	}
}

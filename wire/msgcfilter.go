// Copyright (c) 2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"fmt"
	"io"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
)

// FilterType is used to represent a filter type.
type FilterType uint8

const (
	// GCSFilterRegular is the regular filter type.
	GCSFilterRegular FilterType = iota
)

const (
	// MaxCFilterDataSize is the maximum byte size of a committed filter.
	// The maximum size is currently defined as 2MiB.
	MaxCFilterDataSize = 2 * 1024 * 1024
)

// MsgCFilter implements the Message interface and represents a bitcoin cfilter
// message. It is used to deliver a committed filter in response to a
// getcfilters (MsgGetCFilters) message.
type MsgCFilter struct {
	FilterType FilterType
	BlockHash  chainhash.Hash
	Data       []byte
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgCFilter) BtcDecode(r io.Reader, pver uint32, _ MessageEncoding) er.R {
	// Read filter type
	err := readElement(r, &msg.FilterType)
	if err != nil {
		return err
	}

	// Read the hash of the filter's block
	err = readElement(r, &msg.BlockHash)
	if err != nil {
		return err
	}

	// Read filter data
	msg.Data, err = ReadVarBytes(r, pver, MaxCFilterDataSize,
		"cfilter data")
	return err
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgCFilter) BtcEncode(w io.Writer, pver uint32, _ MessageEncoding) er.R {
	size := len(msg.Data)
	if size > MaxCFilterDataSize {
		str := fmt.Sprintf("cfilter size too large for message "+
			"[size %v, max %v]", size, MaxCFilterDataSize)
		return messageError("MsgCFilter.BtcEncode", str)
	}

	err := writeElement(w, msg.FilterType)
	if err != nil {
		return err
	}

	err = writeElement(w, msg.BlockHash)
	if err != nil {
		return err
	}

	return WriteVarBytes(w, pver, msg.Data)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgCFilter) Command() string {
	return CmdCFilter
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgCFilter) MaxPayloadLength(pver uint32) uint32 {
	return uint32(VarIntSerializeSize(MaxCFilterDataSize)) +
		MaxCFilterDataSize + chainhash.HashSize + 1
}

// NewMsgCFilter returns a new bitcoin cfilter message that conforms to the
// Message interface. See MsgCFilter for details.
func NewMsgCFilter(filterType FilterType, blockHash *chainhash.Hash,
	data []byte) *MsgCFilter {
	return &MsgCFilter{
		FilterType: filterType,
		BlockHash:  *blockHash,
		Data:       data,
	}
}

// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// code derived from https://github .com/btcsuite/btcd/blob/master/wire/message.go
// Copyright (C) 2015-2019 The Lightning Network Developers

package lnwire

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/btcutil/util"
)

// MaxMessagePayload is the maximum bytes a message can be regardless of other
// individual limits imposed by messages themselves.
const MaxMessagePayload = 65535 // 65KB

// MessageType is the unique 2 byte big-endian integer that indicates the type
// of message on the wire. All messages have a very simple header which
// consists simply of 2-byte message type. We omit a length field, and checksum
// as the Lightning Protocol is intended to be encapsulated within a
// confidential+authenticated cryptographic messaging protocol.
type MessageType uint16

// The currently defined message types within this current version of the
// Lightning protocol.
const (
	MsgInit                    MessageType = 16
	MsgError                   MessageType = 17
	MsgPing                    MessageType = 18
	MsgPong                    MessageType = 19
	MsgOpenChannel             MessageType = 32
	MsgAcceptChannel           MessageType = 33
	MsgFundingCreated          MessageType = 34
	MsgFundingSigned           MessageType = 35
	MsgFundingLocked           MessageType = 36
	MsgShutdown                MessageType = 38
	MsgClosingSigned           MessageType = 39
	MsgUpdateAddHTLC           MessageType = 128
	MsgUpdateFulfillHTLC       MessageType = 130
	MsgUpdateFailHTLC          MessageType = 131
	MsgCommitSig               MessageType = 132
	MsgRevokeAndAck            MessageType = 133
	MsgUpdateFee               MessageType = 134
	MsgUpdateFailMalformedHTLC MessageType = 135
	MsgChannelReestablish      MessageType = 136
	MsgChannelAnnouncement     MessageType = 256
	MsgNodeAnnouncement        MessageType = 257
	MsgChannelUpdate           MessageType = 258
	MsgAnnounceSignatures      MessageType = 259
	MsgQueryShortChanIDs       MessageType = 261
	MsgReplyShortChanIDsEnd    MessageType = 262
	MsgQueryChannelRange       MessageType = 263
	MsgReplyChannelRange       MessageType = 264
	MsgGossipTimestampRange    MessageType = 265
)

// String return the string representation of message type.
func (t MessageType) String() string {
	switch t {
	case MsgInit:
		return "Init"
	case MsgOpenChannel:
		return "MsgOpenChannel"
	case MsgAcceptChannel:
		return "MsgAcceptChannel"
	case MsgFundingCreated:
		return "MsgFundingCreated"
	case MsgFundingSigned:
		return "MsgFundingSigned"
	case MsgFundingLocked:
		return "FundingLocked"
	case MsgShutdown:
		return "Shutdown"
	case MsgClosingSigned:
		return "ClosingSigned"
	case MsgUpdateAddHTLC:
		return "UpdateAddHTLC"
	case MsgUpdateFailHTLC:
		return "UpdateFailHTLC"
	case MsgUpdateFulfillHTLC:
		return "UpdateFulfillHTLC"
	case MsgCommitSig:
		return "CommitSig"
	case MsgRevokeAndAck:
		return "RevokeAndAck"
	case MsgUpdateFailMalformedHTLC:
		return "UpdateFailMalformedHTLC"
	case MsgChannelReestablish:
		return "ChannelReestablish"
	case MsgError:
		return "Error"
	case MsgChannelAnnouncement:
		return "ChannelAnnouncement"
	case MsgChannelUpdate:
		return "ChannelUpdate"
	case MsgNodeAnnouncement:
		return "NodeAnnouncement"
	case MsgPing:
		return "Ping"
	case MsgAnnounceSignatures:
		return "AnnounceSignatures"
	case MsgPong:
		return "Pong"
	case MsgUpdateFee:
		return "UpdateFee"
	case MsgQueryShortChanIDs:
		return "QueryShortChanIDs"
	case MsgReplyShortChanIDsEnd:
		return "ReplyShortChanIDsEnd"
	case MsgQueryChannelRange:
		return "QueryChannelRange"
	case MsgReplyChannelRange:
		return "ReplyChannelRange"
	case MsgGossipTimestampRange:
		return "GossipTimestampRange"
	default:
		return "<unknown>"
	}
}

// UnknownMessage is an implementation of the error interface that allows the
// creation of an error in response to an unknown message.
var UnknownMessage = Err.CodeWithDetail("UnknownMessage",
	"unable to parse message of unknown type")

// Serializable is an interface which defines a lightning wire serializable
// object.
type Serializable interface {
	// Decode reads the bytes stream and converts it to the object.
	Decode(io.Reader, uint32) er.R

	// Encode converts object to the bytes stream and write it into the
	// writer.
	Encode(io.Writer, uint32) er.R
}

// Message is an interface that defines a lightning wire protocol message. The
// interface is general in order to allow implementing types full control over
// the representation of its data.
type Message interface {
	Serializable
	MsgType() MessageType
	MaxPayloadLength(uint32) uint32
}

// makeEmptyMessage creates a new empty message of the proper concrete type
// based on the passed message type.
func makeEmptyMessage(msgType MessageType) (Message, er.R) {
	var msg Message

	switch msgType {
	case MsgInit:
		msg = &Init{}
	case MsgOpenChannel:
		msg = &OpenChannel{}
	case MsgAcceptChannel:
		msg = &AcceptChannel{}
	case MsgFundingCreated:
		msg = &FundingCreated{}
	case MsgFundingSigned:
		msg = &FundingSigned{}
	case MsgFundingLocked:
		msg = &FundingLocked{}
	case MsgShutdown:
		msg = &Shutdown{}
	case MsgClosingSigned:
		msg = &ClosingSigned{}
	case MsgUpdateAddHTLC:
		msg = &UpdateAddHTLC{}
	case MsgUpdateFailHTLC:
		msg = &UpdateFailHTLC{}
	case MsgUpdateFulfillHTLC:
		msg = &UpdateFulfillHTLC{}
	case MsgCommitSig:
		msg = &CommitSig{}
	case MsgRevokeAndAck:
		msg = &RevokeAndAck{}
	case MsgUpdateFee:
		msg = &UpdateFee{}
	case MsgUpdateFailMalformedHTLC:
		msg = &UpdateFailMalformedHTLC{}
	case MsgChannelReestablish:
		msg = &ChannelReestablish{}
	case MsgError:
		msg = &Error{}
	case MsgChannelAnnouncement:
		msg = &ChannelAnnouncement{}
	case MsgChannelUpdate:
		msg = &ChannelUpdate{}
	case MsgNodeAnnouncement:
		msg = &NodeAnnouncement{}
	case MsgPing:
		msg = &Ping{}
	case MsgAnnounceSignatures:
		msg = &AnnounceSignatures{}
	case MsgPong:
		msg = &Pong{}
	case MsgQueryShortChanIDs:
		msg = &QueryShortChanIDs{}
	case MsgReplyShortChanIDsEnd:
		msg = &ReplyShortChanIDsEnd{}
	case MsgQueryChannelRange:
		msg = &QueryChannelRange{}
	case MsgReplyChannelRange:
		msg = &ReplyChannelRange{}
	case MsgGossipTimestampRange:
		msg = &GossipTimestampRange{}
	default:
		return nil, UnknownMessage.New(fmt.Sprintf("%v", msgType), nil)
	}

	return msg, nil
}

// WriteMessage writes a lightning Message to w including the necessary header
// information and returns the number of bytes written.
func WriteMessage(w io.Writer, msg Message, pver uint32) (int, er.R) {
	totalBytes := 0

	// Encode the message payload itself into a temporary buffer.
	// TODO(roasbeef): create buffer pool
	var bw bytes.Buffer
	if err := msg.Encode(&bw, pver); err != nil {
		return totalBytes, err
	}
	payload := bw.Bytes()
	lenp := len(payload)

	// Enforce maximum overall message payload.
	if lenp > MaxMessagePayload {
		return totalBytes, er.Errorf("message payload is too large - "+
			"encoded %d bytes, but maximum message payload is %d bytes",
			lenp, MaxMessagePayload)
	}

	// Enforce maximum message payload on the message type.
	mpl := msg.MaxPayloadLength(pver)
	if uint32(lenp) > mpl {
		return totalBytes, er.Errorf("message payload is too large - "+
			"encoded %d bytes, but maximum message payload of "+
			"type %v is %d bytes", lenp, msg.MsgType(), mpl)
	}

	// With the initial sanity checks complete, we'll now write out the
	// message type itself.
	var mType [2]byte
	binary.BigEndian.PutUint16(mType[:], uint16(msg.MsgType()))
	n, err := util.Write(w, mType[:])
	totalBytes += n
	if err != nil {
		return totalBytes, err
	}

	// With the message type written, we'll now write out the raw payload
	// itself.
	n, err = util.Write(w, payload)
	totalBytes += n

	return totalBytes, err
}

// ReadMessage reads, validates, and parses the next Lightning message from r
// for the provided protocol version.
func ReadMessage(r io.Reader, pver uint32) (Message, er.R) {
	// First, we'll read out the first two bytes of the message so we can
	// create the proper empty message.
	var mType [2]byte
	if _, err := util.ReadFull(r, mType[:]); err != nil {
		return nil, err
	}

	msgType := MessageType(binary.BigEndian.Uint16(mType[:]))

	// Now that we know the target message type, we can create the proper
	// empty message type and decode the message into it.
	msg, err := makeEmptyMessage(msgType)
	if err != nil {
		return nil, err
	}
	if err := msg.Decode(r, pver); err != nil {
		return nil, err
	}

	return msg, nil
}

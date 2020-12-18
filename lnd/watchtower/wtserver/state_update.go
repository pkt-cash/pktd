package wtserver

import (
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/lnd/watchtower/wtdb"
	"github.com/pkt-cash/pktd/lnd/watchtower/wtwire"
	"github.com/pkt-cash/pktd/pktlog/log"
)

// handleStateUpdates processes a stream of StateUpdate requests from the
// client. The provided update should be the first such update read, subsequent
// updates will be consumed if the peer does not signal IsComplete on a
// particular update.
func (s *Server) handleStateUpdates(peer Peer, id *wtdb.SessionID,
	update *wtwire.StateUpdate) er.R {
	// Set the current update to the first update read off the wire.
	// Additional updates will be read if this value is set to nil after
	// processing the first.
	curUpdate := update
	for {
		// If this is not the first update, read the next state update
		// from the peer.
		if curUpdate == nil {
			nextMsg, err := s.readMessage(peer)
			if err != nil {
				return err
			}

			var ok bool
			curUpdate, ok = nextMsg.(*wtwire.StateUpdate)
			if !ok {
				return er.Errorf("client sent %T after "+
					"StateUpdate", nextMsg)
			}
		}

		// Try to accept the state update from the client.
		err := s.handleStateUpdate(peer, id, curUpdate)
		if err != nil {
			return err
		}

		// If the client signals that this is last StateUpdate
		// message, we can disconnect the client.
		if curUpdate.IsComplete == 1 {
			return nil
		}

		// Reset the current update to read subsequent updates in the
		// stream.
		curUpdate = nil

		select {
		case <-s.quit:
			return ErrServerExiting.Default()
		default:
		}
	}
}

// handleStateUpdate processes a StateUpdate message request from a client. An
// attempt will be made to insert the update into the db, where it is validated
// against the client's session. The possible errors are then mapped back to
// StateUpdateCodes specified by the watchtower wire protocol, and sent back
// using a StateUpdateReply message.
func (s *Server) handleStateUpdate(peer Peer, id *wtdb.SessionID,
	update *wtwire.StateUpdate) er.R {
	var (
		lastApplied uint16
		failCode    wtwire.ErrorCode
		err         er.R
	)

	sessionUpdate := wtdb.SessionStateUpdate{
		ID:            *id,
		Hint:          update.Hint,
		SeqNum:        update.SeqNum,
		LastApplied:   update.LastApplied,
		EncryptedBlob: update.EncryptedBlob,
	}

	lastApplied, err = s.cfg.DB.InsertStateUpdate(&sessionUpdate)
	switch {
	case err == nil:
		log.Debugf("State update %d accepted for %s",
			update.SeqNum, id)

		failCode = wtwire.CodeOK

	// Return a permanent failure if a client tries to send an update for
	// which we have no session.
	case wtdb.ErrSessionNotFound.Is(err):
		failCode = wtwire.CodePermanentFailure

	case wtdb.ErrSeqNumAlreadyApplied.Is(err):
		failCode = wtwire.CodePermanentFailure

		// TODO(conner): remove session state for protocol
		// violation. Could also double as clean up method for
		// session-related state.

	case wtdb.ErrLastAppliedReversion.Is(err):
		failCode = wtwire.StateUpdateCodeClientBehind

	case wtdb.ErrSessionConsumed.Is(err):
		failCode = wtwire.StateUpdateCodeMaxUpdatesExceeded

	case wtdb.ErrUpdateOutOfOrder.Is(err):
		failCode = wtwire.StateUpdateCodeSeqNumOutOfOrder

	default:
		failCode = wtwire.CodeTemporaryFailure
	}

	if s.cfg.NoAckUpdates {
		return er.E(&connFailure{
			ID:   *id,
			Code: failCode,
		})
	}

	return s.replyStateUpdate(
		peer, id, failCode, lastApplied,
	)
}

// replyStateUpdate sends a response to a StateUpdate from a client. If the
// status code in the reply is OK, the error from the write will be bubbled up.
// Otherwise, this method returns a connection error to ensure we don't continue
// communication with the client.
func (s *Server) replyStateUpdate(peer Peer, id *wtdb.SessionID,
	code wtwire.StateUpdateCode, lastApplied uint16) er.R {
	msg := &wtwire.StateUpdateReply{
		Code:        code,
		LastApplied: lastApplied,
	}

	err := s.sendMessage(peer, msg)
	if err != nil {
		log.Errorf("unable to send StateUpdateReply to %s", id)
	}

	// Return the write error if the request succeeded.
	if code == wtwire.CodeOK {
		return err
	}

	// Otherwise the request failed, return a connection failure to
	// disconnect the client.
	return er.E(&connFailure{
		ID:   *id,
		Code: code,
	})
}

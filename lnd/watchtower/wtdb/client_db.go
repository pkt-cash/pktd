package wtdb

import (
	"bytes"
	"math"
	"net"

	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/lnd/channeldb/kvdb"
	"github.com/pkt-cash/pktd/lnd/lnwire"
)

const (
	// clientDBName is the filename of client database.
	clientDBName = "wtclient.db"
)

var (
	// cSessionKeyIndexBkt is a top-level bucket storing:
	//   tower-id -> reserved-session-key-index (uint32).
	cSessionKeyIndexBkt = []byte("client-session-key-index-bucket")

	// cChanSummaryBkt is a top-level bucket storing:
	//   channel-id -> encoded ClientChanSummary.
	cChanSummaryBkt = []byte("client-channel-summary-bucket")

	// cSessionBkt is a top-level bucket storing:
	//   session-id => cSessionBody -> encoded ClientSessionBody
	//              => cSessionCommits => seqnum -> encoded CommittedUpdate
	//              => cSessionAcks => seqnum -> encoded BackupID
	cSessionBkt = []byte("client-session-bucket")

	// cSessionBody is a sub-bucket of cSessionBkt storing only the body of
	// the ClientSession.
	cSessionBody = []byte("client-session-body")

	// cSessionBody is a sub-bucket of cSessionBkt storing:
	//    seqnum -> encoded CommittedUpdate.
	cSessionCommits = []byte("client-session-commits")

	// cSessionAcks is a sub-bucket of cSessionBkt storing:
	//    seqnum -> encoded BackupID.
	cSessionAcks = []byte("client-session-acks")

	// cTowerBkt is a top-level bucket storing:
	//    tower-id -> encoded Tower.
	cTowerBkt = []byte("client-tower-bucket")

	// cTowerIndexBkt is a top-level bucket storing:
	//    tower-pubkey -> tower-id.
	cTowerIndexBkt = []byte("client-tower-index-bucket")

	// ErrTowerNotFound signals that the target tower was not found in the
	// database.
	ErrTowerNotFound = Err.CodeWithDetail("ErrTowerNotFound", "tower not found")

	// ErrTowerUnackedUpdates is an error returned when we attempt to mark a
	// tower's sessions as inactive, but one of its sessions has unacked
	// updates.
	ErrTowerUnackedUpdates = Err.CodeWithDetail("ErrTowerUnackedUpdates", "tower has unacked updates")

	// ErrCorruptClientSession signals that the client session's on-disk
	// structure deviates from what is expected.
	ErrCorruptClientSession = Err.CodeWithDetail("ErrCorruptClientSession", "client session corrupted")

	// ErrClientSessionAlreadyExists signals an attempt to reinsert a client
	// session that has already been created.
	ErrClientSessionAlreadyExists = Err.CodeWithDetail("ErrClientSessionAlreadyExists",
		"client session already exists",
	)

	// ErrChannelAlreadyRegistered signals a duplicate attempt to register a
	// channel with the client database.
	ErrChannelAlreadyRegistered = Err.CodeWithDetail("ErrChannelAlreadyRegistered", "channel already registered")

	// ErrChannelNotRegistered signals a channel has not yet been registered
	// in the client database.
	ErrChannelNotRegistered = Err.CodeWithDetail("ErrChannelNotRegistered", "channel not registered")

	// ErrClientSessionNotFound signals that the requested client session
	// was not found in the database.
	ErrClientSessionNotFound = Err.CodeWithDetail("ErrClientSessionNotFound", "client session not found")

	// ErrUpdateAlreadyCommitted signals that the chosen sequence number has
	// already been committed to an update with a different breach hint.
	ErrUpdateAlreadyCommitted = Err.CodeWithDetail("ErrUpdateAlreadyCommitted", "update already committed")

	// ErrCommitUnorderedUpdate signals the client tried to commit a
	// sequence number other than the next unallocated sequence number.
	ErrCommitUnorderedUpdate = Err.CodeWithDetail("ErrCommitUnorderedUpdate", "update seqnum not monotonic")

	// ErrCommittedUpdateNotFound signals that the tower tried to ACK a
	// sequence number that has not yet been allocated by the client.
	ErrCommittedUpdateNotFound = Err.CodeWithDetail("ErrCommittedUpdateNotFound", "committed update not found")

	// ErrUnallocatedLastApplied signals that the tower tried to provide a
	// LastApplied value greater than any allocated sequence number.
	ErrUnallocatedLastApplied = Err.CodeWithDetail("ErrUnallocatedLastApplied", "tower echoed last appiled "+
		"greater than allocated seqnum")

	// ErrNoReservedKeyIndex signals that a client session could not be
	// created because no session key index was reserved.
	ErrNoReservedKeyIndex = Err.CodeWithDetail("ErrNoReservedKeyIndex", "key index not reserved")

	// ErrIncorrectKeyIndex signals that the client session could not be
	// created because session key index differs from the reserved key
	// index.
	ErrIncorrectKeyIndex = Err.CodeWithDetail("ErrIncorrectKeyIndex", "incorrect key index")

	// ErrLastTowerAddr is an error returned when the last address of a
	// watchtower is attempted to be removed.
	ErrLastTowerAddr = Err.CodeWithDetail("ErrLastTowerAddr", "cannot remove last tower address")
)

// ClientDB is single database providing a persistent storage engine for the
// wtclient.
type ClientDB struct {
	db     kvdb.Backend
	dbPath string
}

// OpenClientDB opens the client database given the path to the database's
// directory. If no such database exists, this method will initialize a fresh
// one using the latest version number and bucket structure. If a database
// exists but has a lower version number than the current version, any necessary
// migrations will be applied before returning. Any attempt to open a database
// with a version number higher that the latest version will fail to prevent
// accidental reversion.
func OpenClientDB(dbPath string) (*ClientDB, er.R) {
	bdb, firstInit, err := createDBIfNotExist(dbPath, clientDBName)
	if err != nil {
		return nil, err
	}

	clientDB := &ClientDB{
		db:     bdb,
		dbPath: dbPath,
	}

	err = initOrSyncVersions(clientDB, firstInit, clientDBVersions)
	if err != nil {
		bdb.Close()
		return nil, err
	}

	// Now that the database version fully consistent with our latest known
	// version, ensure that all top-level buckets known to this version are
	// initialized. This allows us to assume their presence throughout all
	// operations. If an known top-level bucket is expected to exist but is
	// missing, this will trigger a ErrUninitializedDB error.
	err = kvdb.Update(clientDB.db, initClientDBBuckets, func() {})
	if err != nil {
		bdb.Close()
		return nil, err
	}

	return clientDB, nil
}

// initClientDBBuckets creates all top-level buckets required to handle database
// operations required by the latest version.
func initClientDBBuckets(tx kvdb.RwTx) er.R {
	buckets := [][]byte{
		cSessionKeyIndexBkt,
		cChanSummaryBkt,
		cSessionBkt,
		cTowerBkt,
		cTowerIndexBkt,
	}

	for _, bucket := range buckets {
		_, err := tx.CreateTopLevelBucket(bucket)
		if err != nil {
			return err
		}
	}

	return nil
}

// bdb returns the backing bbolt.DB instance.
//
// NOTE: Part of the versionedDB interface.
func (c *ClientDB) bdb() kvdb.Backend {
	return c.db
}

// Version returns the database's current version number.
//
// NOTE: Part of the versionedDB interface.
func (c *ClientDB) Version() (uint32, er.R) {
	var version uint32
	err := kvdb.View(c.db, func(tx kvdb.RTx) er.R {
		var err er.R
		version, err = getDBVersion(tx)
		return err
	}, func() {
		version = 0
	})
	if err != nil {
		return 0, err
	}

	return version, nil
}

// Close closes the underlying database.
func (c *ClientDB) Close() er.R {
	return c.db.Close()
}

// CreateTower initialize an address record used to communicate with a
// watchtower. Each Tower is assigned a unique ID, that is used to amortize
// storage costs of the public key when used by multiple sessions. If the tower
// already exists, the address is appended to the list of all addresses used to
// that tower previously and its corresponding sessions are marked as active.
func (c *ClientDB) CreateTower(lnAddr *lnwire.NetAddress) (*Tower, er.R) {
	var towerPubKey [33]byte
	copy(towerPubKey[:], lnAddr.IdentityKey.SerializeCompressed())

	var tower *Tower
	err := kvdb.Update(c.db, func(tx kvdb.RwTx) er.R {
		towerIndex := tx.ReadWriteBucket(cTowerIndexBkt)
		if towerIndex == nil {
			return ErrUninitializedDB.Default()
		}

		towers := tx.ReadWriteBucket(cTowerBkt)
		if towers == nil {
			return ErrUninitializedDB.Default()
		}

		// Check if the tower index already knows of this pubkey.
		towerIDBytes := towerIndex.Get(towerPubKey[:])
		if len(towerIDBytes) == 8 {
			// The tower already exists, deserialize the existing
			// record.
			var err er.R
			tower, err = getTower(towers, towerIDBytes)
			if err != nil {
				return err
			}

			// Add the new address to the existing tower. If the
			// address is a duplicate, this will result in no
			// change.
			tower.AddAddress(lnAddr.Address)

			// If there are any client sessions that correspond to
			// this tower, we'll mark them as active to ensure we
			// load them upon restarts.
			//
			// TODO(wilmer): with an index of tower -> sessions we
			// can avoid the linear lookup.
			sessions := tx.ReadWriteBucket(cSessionBkt)
			if sessions == nil {
				return ErrUninitializedDB.Default()
			}
			towerID := TowerIDFromBytes(towerIDBytes)
			towerSessions, err := listClientSessions(
				sessions, &towerID,
			)
			if err != nil {
				return err
			}
			for _, session := range towerSessions {
				err := markSessionStatus(
					sessions, session, CSessionActive,
				)
				if err != nil {
					return err
				}
			}
		} else {
			// No such tower exists, create a new tower id for our
			// new tower. The error is unhandled since NextSequence
			// never fails in an Update.
			towerID, _ := towerIndex.NextSequence()

			tower = &Tower{
				ID:          TowerID(towerID),
				IdentityKey: lnAddr.IdentityKey,
				Addresses:   []net.Addr{lnAddr.Address},
			}

			towerIDBytes = tower.ID.Bytes()

			// Since this tower is new, record the mapping from
			// tower pubkey to tower id in the tower index.
			err := towerIndex.Put(towerPubKey[:], towerIDBytes)
			if err != nil {
				return err
			}
		}

		// Store the new or updated tower under its tower id.
		return putTower(towers, tower)
	}, func() {
		tower = nil
	})
	if err != nil {
		return nil, err
	}

	return tower, nil
}

// RemoveTower modifies a tower's record within the database. If an address is
// provided, then _only_ the address record should be removed from the tower's
// persisted state. Otherwise, we'll attempt to mark the tower as inactive by
// marking all of its sessions inactive. If any of its sessions has unacked
// updates, then ErrTowerUnackedUpdates is returned. If the tower doesn't have
// any sessions at all, it'll be completely removed from the database.
//
// NOTE: An error is not returned if the tower doesn't exist.
func (c *ClientDB) RemoveTower(pubKey *btcec.PublicKey, addr net.Addr) er.R {
	return kvdb.Update(c.db, func(tx kvdb.RwTx) er.R {
		towers := tx.ReadWriteBucket(cTowerBkt)
		if towers == nil {
			return ErrUninitializedDB.Default()
		}
		towerIndex := tx.ReadWriteBucket(cTowerIndexBkt)
		if towerIndex == nil {
			return ErrUninitializedDB.Default()
		}

		// Don't return an error if the watchtower doesn't exist to act
		// as a NOP.
		pubKeyBytes := pubKey.SerializeCompressed()
		towerIDBytes := towerIndex.Get(pubKeyBytes)
		if towerIDBytes == nil {
			return nil
		}

		// If an address is provided, then we should _only_ remove the
		// address record from the database.
		if addr != nil {
			tower, err := getTower(towers, towerIDBytes)
			if err != nil {
				return err
			}

			// Towers should always have at least one address saved.
			tower.RemoveAddress(addr)
			if len(tower.Addresses) == 0 {
				return ErrLastTowerAddr.Default()
			}

			return putTower(towers, tower)
		}

		// Otherwise, we should attempt to mark the tower's sessions as
		// inactive.
		//
		// TODO(wilmer): with an index of tower -> sessions we can avoid
		// the linear lookup.
		sessions := tx.ReadWriteBucket(cSessionBkt)
		if sessions == nil {
			return ErrUninitializedDB.Default()
		}
		towerID := TowerIDFromBytes(towerIDBytes)
		towerSessions, err := listClientSessions(sessions, &towerID)
		if err != nil {
			return err
		}

		// If it doesn't have any, we can completely remove it from the
		// database.
		if len(towerSessions) == 0 {
			if err := towerIndex.Delete(pubKeyBytes); err != nil {
				return err
			}
			return towers.Delete(towerIDBytes)
		}

		// We'll mark its sessions as inactive as long as they don't
		// have any pending updates to ensure we don't load them upon
		// restarts.
		for _, session := range towerSessions {
			if len(session.CommittedUpdates) > 0 {
				return ErrTowerUnackedUpdates.Default()
			}
			err := markSessionStatus(
				sessions, session, CSessionInactive,
			)
			if err != nil {
				return err
			}
		}

		return nil
	}, func() {})
}

// LoadTowerByID retrieves a tower by its tower ID.
func (c *ClientDB) LoadTowerByID(towerID TowerID) (*Tower, er.R) {
	var tower *Tower
	err := kvdb.View(c.db, func(tx kvdb.RTx) er.R {
		towers := tx.ReadBucket(cTowerBkt)
		if towers == nil {
			return ErrUninitializedDB.Default()
		}

		var err er.R
		tower, err = getTower(towers, towerID.Bytes())
		return err
	}, func() {
		tower = nil
	})
	if err != nil {
		return nil, err
	}

	return tower, nil
}

// LoadTower retrieves a tower by its public key.
func (c *ClientDB) LoadTower(pubKey *btcec.PublicKey) (*Tower, er.R) {
	var tower *Tower
	err := kvdb.View(c.db, func(tx kvdb.RTx) er.R {
		towers := tx.ReadBucket(cTowerBkt)
		if towers == nil {
			return ErrUninitializedDB.Default()
		}
		towerIndex := tx.ReadBucket(cTowerIndexBkt)
		if towerIndex == nil {
			return ErrUninitializedDB.Default()
		}

		towerIDBytes := towerIndex.Get(pubKey.SerializeCompressed())
		if towerIDBytes == nil {
			return ErrTowerNotFound.Default()
		}

		var err er.R
		tower, err = getTower(towers, towerIDBytes)
		return err
	}, func() {
		tower = nil
	})
	if err != nil {
		return nil, err
	}

	return tower, nil
}

// ListTowers retrieves the list of towers available within the database.
func (c *ClientDB) ListTowers() ([]*Tower, er.R) {
	var towers []*Tower
	err := kvdb.View(c.db, func(tx kvdb.RTx) er.R {
		towerBucket := tx.ReadBucket(cTowerBkt)
		if towerBucket == nil {
			return ErrUninitializedDB.Default()
		}

		return towerBucket.ForEach(func(towerIDBytes, _ []byte) er.R {
			tower, err := getTower(towerBucket, towerIDBytes)
			if err != nil {
				return err
			}
			towers = append(towers, tower)
			return nil
		})
	}, func() {
		towers = nil
	})
	if err != nil {
		return nil, err
	}

	return towers, nil
}

// NextSessionKeyIndex reserves a new session key derivation index for a
// particular tower id. The index is reserved for that tower until
// CreateClientSession is invoked for that tower and index, at which point a new
// index for that tower can be reserved. Multiple calls to this method before
// CreateClientSession is invoked should return the same index.
func (c *ClientDB) NextSessionKeyIndex(towerID TowerID) (uint32, er.R) {
	var index uint32
	err := kvdb.Update(c.db, func(tx kvdb.RwTx) er.R {
		keyIndex := tx.ReadWriteBucket(cSessionKeyIndexBkt)
		if keyIndex == nil {
			return ErrUninitializedDB.Default()
		}

		// Check the session key index to see if a key has already been
		// reserved for this tower. If so, we'll deserialize and return
		// the index directly.
		towerIDBytes := towerID.Bytes()
		indexBytes := keyIndex.Get(towerIDBytes)
		if len(indexBytes) == 4 {
			index = byteOrder.Uint32(indexBytes)
			return nil
		}

		// Otherwise, generate a new session key index since the node
		// doesn't already have reserved index. The error is ignored
		// since NextSequence can't fail inside Update.
		index64, _ := keyIndex.NextSequence()

		// As a sanity check, assert that the index is still in the
		// valid range of unhardened pubkeys. In the future, we should
		// move to only using hardened keys, and this will prevent any
		// overlap from occurring until then. This also prevents us from
		// overflowing uint32s.
		if index64 > math.MaxInt32 {
			return er.Errorf("exhausted session key indexes")
		}

		index = uint32(index64)

		var indexBuf [4]byte
		byteOrder.PutUint32(indexBuf[:], index)

		// Record the reserved session key index under this tower's id.
		return keyIndex.Put(towerIDBytes, indexBuf[:])
	}, func() {
		index = 0
	})
	if err != nil {
		return 0, err
	}

	return index, nil
}

// CreateClientSession records a newly negotiated client session in the set of
// active sessions. The session can be identified by its SessionID.
func (c *ClientDB) CreateClientSession(session *ClientSession) er.R {
	return kvdb.Update(c.db, func(tx kvdb.RwTx) er.R {
		keyIndexes := tx.ReadWriteBucket(cSessionKeyIndexBkt)
		if keyIndexes == nil {
			return ErrUninitializedDB.Default()
		}

		sessions := tx.ReadWriteBucket(cSessionBkt)
		if sessions == nil {
			return ErrUninitializedDB.Default()
		}

		// Check that  client session with this session id doesn't
		// already exist.
		existingSessionBytes := sessions.NestedReadWriteBucket(session.ID[:])
		if existingSessionBytes != nil {
			return ErrClientSessionAlreadyExists.Default()
		}

		// Check that this tower has a reserved key index.
		towerIDBytes := session.TowerID.Bytes()
		keyIndexBytes := keyIndexes.Get(towerIDBytes)
		if len(keyIndexBytes) != 4 {
			return ErrNoReservedKeyIndex.Default()
		}

		// Assert that the key index of the inserted session matches the
		// reserved session key index.
		index := byteOrder.Uint32(keyIndexBytes)
		if index != session.KeyIndex {
			return ErrIncorrectKeyIndex.Default()
		}

		// Remove the key index reservation.
		err := keyIndexes.Delete(towerIDBytes)
		if err != nil {
			return err
		}

		// Finally, write the client session's body in the sessions
		// bucket.
		return putClientSessionBody(sessions, session)
	}, func() {})
}

// ListClientSessions returns the set of all client sessions known to the db. An
// optional tower ID can be used to filter out any client sessions in the
// response that do not correspond to this tower.
func (c *ClientDB) ListClientSessions(id *TowerID) (map[SessionID]*ClientSession, er.R) {
	var clientSessions map[SessionID]*ClientSession
	err := kvdb.View(c.db, func(tx kvdb.RTx) er.R {
		sessions := tx.ReadBucket(cSessionBkt)
		if sessions == nil {
			return ErrUninitializedDB.Default()
		}
		var err er.R
		clientSessions, err = listClientSessions(sessions, id)
		return err
	}, func() {
		clientSessions = nil
	})
	if err != nil {
		return nil, err
	}

	return clientSessions, nil
}

// listClientSessions returns the set of all client sessions known to the db. An
// optional tower ID can be used to filter out any client sessions in the
// response that do not correspond to this tower.
func listClientSessions(sessions kvdb.RBucket,
	id *TowerID) (map[SessionID]*ClientSession, er.R) {
	clientSessions := make(map[SessionID]*ClientSession)
	err := sessions.ForEach(func(k, _ []byte) er.R {
		// We'll load the full client session since the client will need
		// the CommittedUpdates and AckedUpdates on startup to resume
		// committed updates and compute the highest known commit height
		// for each channel.
		session, err := getClientSession(sessions, k)
		if err != nil {
			return err
		}

		// Filter out any sessions that don't correspond to the given
		// tower if one was set.
		if id != nil && session.TowerID != *id {
			return nil
		}

		clientSessions[session.ID] = session

		return nil
	})
	if err != nil {
		return nil, err
	}

	return clientSessions, nil
}

// FetchChanSummaries loads a mapping from all registered channels to their
// channel summaries.
func (c *ClientDB) FetchChanSummaries() (ChannelSummaries, er.R) {
	var summaries map[lnwire.ChannelID]ClientChanSummary
	err := kvdb.View(c.db, func(tx kvdb.RTx) er.R {
		chanSummaries := tx.ReadBucket(cChanSummaryBkt)
		if chanSummaries == nil {
			return ErrUninitializedDB.Default()
		}

		return chanSummaries.ForEach(func(k, v []byte) er.R {
			var chanID lnwire.ChannelID
			copy(chanID[:], k)

			var summary ClientChanSummary
			err := summary.Decode(bytes.NewReader(v))
			if err != nil {
				return err
			}

			summaries[chanID] = summary

			return nil
		})
	}, func() {
		summaries = make(map[lnwire.ChannelID]ClientChanSummary)
	})
	if err != nil {
		return nil, err
	}

	return summaries, nil
}

// RegisterChannel registers a channel for use within the client database. For
// now, all that is stored in the channel summary is the sweep pkscript that
// we'd like any tower sweeps to pay into. In the future, this will be extended
// to contain more info to allow the client efficiently request historical
// states to be backed up under the client's active policy.
func (c *ClientDB) RegisterChannel(chanID lnwire.ChannelID,
	sweepPkScript []byte) er.R {
	return kvdb.Update(c.db, func(tx kvdb.RwTx) er.R {
		chanSummaries := tx.ReadWriteBucket(cChanSummaryBkt)
		if chanSummaries == nil {
			return ErrUninitializedDB.Default()
		}

		_, err := getChanSummary(chanSummaries, chanID)
		switch {

		// Summary already exists.
		case err == nil:
			return ErrChannelAlreadyRegistered.Default()

		// Channel is not registered, proceed with registration.
		case ErrChannelNotRegistered.Is(err):

		// Unexpected error.
		default:
			return err
		}

		summary := ClientChanSummary{
			SweepPkScript: sweepPkScript,
		}

		return putChanSummary(chanSummaries, chanID, &summary)
	}, func() {})
}

// MarkBackupIneligible records that the state identified by the (channel id,
// commit height) tuple was ineligible for being backed up under the current
// policy. This state can be retried later under a different policy.
func (c *ClientDB) MarkBackupIneligible(chanID lnwire.ChannelID,
	commitHeight uint64) er.R {
	return nil
}

// CommitUpdate persists the CommittedUpdate provided in the slot for (session,
// seqNum). This allows the client to retransmit this update on startup.
func (c *ClientDB) CommitUpdate(id *SessionID,
	update *CommittedUpdate) (uint16, er.R) {
	var lastApplied uint16
	err := kvdb.Update(c.db, func(tx kvdb.RwTx) er.R {
		sessions := tx.ReadWriteBucket(cSessionBkt)
		if sessions == nil {
			return ErrUninitializedDB.Default()
		}

		// We'll only load the ClientSession body for performance, since
		// we primarily need to inspect its SeqNum and TowerLastApplied
		// fields. The CommittedUpdates will be modified on disk
		// directly.
		session, err := getClientSessionBody(sessions, id[:])
		if err != nil {
			return err
		}

		// Can't fail if the above didn't fail.
		sessionBkt := sessions.NestedReadWriteBucket(id[:])

		// Ensure the session commits sub-bucket is initialized.
		sessionCommits, err := sessionBkt.CreateBucketIfNotExists(
			cSessionCommits,
		)
		if err != nil {
			return err
		}

		var seqNumBuf [2]byte
		byteOrder.PutUint16(seqNumBuf[:], update.SeqNum)

		// Check to see if a committed update already exists for this
		// sequence number.
		committedUpdateBytes := sessionCommits.Get(seqNumBuf[:])
		if committedUpdateBytes != nil {
			var dbUpdate CommittedUpdate
			err := dbUpdate.Decode(
				bytes.NewReader(committedUpdateBytes),
			)
			if err != nil {
				return err
			}

			// If an existing committed update has a different hint,
			// we'll reject this newer update.
			if dbUpdate.Hint != update.Hint {
				return ErrUpdateAlreadyCommitted.Default()
			}

			// Otherwise, capture the last applied value and
			// succeed.
			lastApplied = session.TowerLastApplied
			return nil
		}

		// There's no committed update for this sequence number, ensure
		// that we are committing the next unallocated one.
		if update.SeqNum != session.SeqNum+1 {
			return ErrCommitUnorderedUpdate.Default()
		}

		// Increment the session's sequence number and store the updated
		// client session.
		//
		// TODO(conner): split out seqnum and last applied own bucket to
		// eliminate serialization of full struct during CommitUpdate?
		// Can also read/write directly to byes [:2] without migration.
		session.SeqNum++
		err = putClientSessionBody(sessions, session)
		if err != nil {
			return err
		}

		// Encode and store the committed update in the sessionCommits
		// sub-bucket under the requested sequence number.
		var b bytes.Buffer
		err = update.Encode(&b)
		if err != nil {
			return err
		}

		err = sessionCommits.Put(seqNumBuf[:], b.Bytes())
		if err != nil {
			return err
		}

		// Finally, capture the session's last applied value so it can
		// be sent in the next state update to the tower.
		lastApplied = session.TowerLastApplied

		return nil
	}, func() {
		lastApplied = 0
	})
	if err != nil {
		return 0, err
	}

	return lastApplied, nil
}

// AckUpdate persists an acknowledgment for a given (session, seqnum) pair. This
// removes the update from the set of committed updates, and validates the
// lastApplied value returned from the tower.
func (c *ClientDB) AckUpdate(id *SessionID, seqNum uint16,
	lastApplied uint16) er.R {
	return kvdb.Update(c.db, func(tx kvdb.RwTx) er.R {
		sessions := tx.ReadWriteBucket(cSessionBkt)
		if sessions == nil {
			return ErrUninitializedDB.Default()
		}

		// We'll only load the ClientSession body for performance, since
		// we primarily need to inspect its SeqNum and TowerLastApplied
		// fields. The CommittedUpdates and AckedUpdates will be
		// modified on disk directly.
		session, err := getClientSessionBody(sessions, id[:])
		if err != nil {
			return err
		}

		// If the tower has acked a sequence number beyond our highest
		// sequence number, fail.
		if lastApplied > session.SeqNum {
			return ErrUnallocatedLastApplied.Default()
		}

		// If the tower acked with a lower sequence number than it gave
		// us prior, fail.
		if lastApplied < session.TowerLastApplied {
			return ErrLastAppliedReversion.Default()
		}

		// TODO(conner): split out seqnum and last applied own bucket to
		// eliminate serialization of full struct during AckUpdate?  Can
		// also read/write directly to byes [2:4] without migration.
		session.TowerLastApplied = lastApplied

		// Write the client session with the updated last applied value.
		err = putClientSessionBody(sessions, session)
		if err != nil {
			return err
		}

		// Can't fail because of getClientSession succeeded.
		sessionBkt := sessions.NestedReadWriteBucket(id[:])

		// If the commits sub-bucket doesn't exist, there can't possibly
		// be a corresponding committed update to remove.
		sessionCommits := sessionBkt.NestedReadWriteBucket(cSessionCommits)
		if sessionCommits == nil {
			return ErrCommittedUpdateNotFound.Default()
		}

		var seqNumBuf [2]byte
		byteOrder.PutUint16(seqNumBuf[:], seqNum)

		// Assert that a committed update exists for this sequence
		// number.
		committedUpdateBytes := sessionCommits.Get(seqNumBuf[:])
		if committedUpdateBytes == nil {
			return ErrCommittedUpdateNotFound.Default()
		}

		var committedUpdate CommittedUpdate
		err = committedUpdate.Decode(
			bytes.NewReader(committedUpdateBytes),
		)
		if err != nil {
			return err
		}

		// Remove the corresponding committed update.
		err = sessionCommits.Delete(seqNumBuf[:])
		if err != nil {
			return err
		}

		// Ensure that the session acks sub-bucket is initialized so we
		// can insert an entry.
		sessionAcks, err := sessionBkt.CreateBucketIfNotExists(
			cSessionAcks,
		)
		if err != nil {
			return err
		}

		// The session acks only need to track the backup id of the
		// update, so we can discard the blob and hint.
		var b bytes.Buffer
		err = committedUpdate.BackupID.Encode(&b)
		if err != nil {
			return err
		}

		// Finally, insert the ack into the sessionAcks sub-bucket.
		return sessionAcks.Put(seqNumBuf[:], b.Bytes())
	}, func() {})
}

// getClientSessionBody loads the body of a ClientSession from the sessions
// bucket corresponding to the serialized session id. This does not deserialize
// the CommittedUpdates or AckUpdates associated with the session. If the caller
// requires this info, use getClientSession.
func getClientSessionBody(sessions kvdb.RBucket,
	idBytes []byte) (*ClientSession, er.R) {
	sessionBkt := sessions.NestedReadBucket(idBytes)
	if sessionBkt == nil {
		return nil, ErrClientSessionNotFound.Default()
	}

	// Should never have a sessionBkt without also having its body.
	sessionBody := sessionBkt.Get(cSessionBody)
	if sessionBody == nil {
		return nil, ErrCorruptClientSession.Default()
	}

	var session ClientSession
	copy(session.ID[:], idBytes)

	err := session.Decode(bytes.NewReader(sessionBody))
	if err != nil {
		return nil, err
	}

	return &session, nil
}

// getClientSession loads the full ClientSession associated with the serialized
// session id. This method populates the CommittedUpdates and AckUpdates in
// addition to the ClientSession's body.
func getClientSession(sessions kvdb.RBucket,
	idBytes []byte) (*ClientSession, er.R) {
	session, err := getClientSessionBody(sessions, idBytes)
	if err != nil {
		return nil, err
	}

	// Fetch the committed updates for this session.
	commitedUpdates, err := getClientSessionCommits(sessions, idBytes)
	if err != nil {
		return nil, err
	}

	// Fetch the acked updates for this session.
	ackedUpdates, err := getClientSessionAcks(sessions, idBytes)
	if err != nil {
		return nil, err
	}

	session.CommittedUpdates = commitedUpdates
	session.AckedUpdates = ackedUpdates

	return session, nil
}

// getClientSessionCommits retrieves all committed updates for the session
// identified by the serialized session id.
func getClientSessionCommits(sessions kvdb.RBucket,
	idBytes []byte) ([]CommittedUpdate, er.R) {
	// Can't fail because client session body has already been read.
	sessionBkt := sessions.NestedReadBucket(idBytes)

	// Initialize commitedUpdates so that we can return an initialized map
	// if no committed updates exist.
	committedUpdates := make([]CommittedUpdate, 0)

	sessionCommits := sessionBkt.NestedReadBucket(cSessionCommits)
	if sessionCommits == nil {
		return committedUpdates, nil
	}

	err := sessionCommits.ForEach(func(k, v []byte) er.R {
		var committedUpdate CommittedUpdate
		err := committedUpdate.Decode(bytes.NewReader(v))
		if err != nil {
			return err
		}
		committedUpdate.SeqNum = byteOrder.Uint16(k)

		committedUpdates = append(committedUpdates, committedUpdate)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return committedUpdates, nil
}

// getClientSessionAcks retrieves all acked updates for the session identified
// by the serialized session id.
func getClientSessionAcks(sessions kvdb.RBucket,
	idBytes []byte) (map[uint16]BackupID, er.R) {
	// Can't fail because client session body has already been read.
	sessionBkt := sessions.NestedReadBucket(idBytes)

	// Initialize ackedUpdates so that we can return an initialized map if
	// no acked updates exist.
	ackedUpdates := make(map[uint16]BackupID)

	sessionAcks := sessionBkt.NestedReadBucket(cSessionAcks)
	if sessionAcks == nil {
		return ackedUpdates, nil
	}

	err := sessionAcks.ForEach(func(k, v []byte) er.R {
		seqNum := byteOrder.Uint16(k)

		var backupID BackupID
		err := backupID.Decode(bytes.NewReader(v))
		if err != nil {
			return err
		}

		ackedUpdates[seqNum] = backupID

		return nil
	})
	if err != nil {
		return nil, err
	}

	return ackedUpdates, nil
}

// putClientSessionBody stores the body of the ClientSession (everything but the
// CommittedUpdates and AckedUpdates).
func putClientSessionBody(sessions kvdb.RwBucket,
	session *ClientSession) er.R {
	sessionBkt, err := sessions.CreateBucketIfNotExists(session.ID[:])
	if err != nil {
		return err
	}

	var b bytes.Buffer
	errr := session.Encode(&b)
	if errr != nil {
		return errr
	}

	return sessionBkt.Put(cSessionBody, b.Bytes())
}

// markSessionStatus updates the persisted state of the session to the new
// status.
func markSessionStatus(sessions kvdb.RwBucket, session *ClientSession,
	status CSessionStatus) er.R {
	session.Status = status
	return putClientSessionBody(sessions, session)
}

// getChanSummary loads a ClientChanSummary for the passed chanID.
func getChanSummary(chanSummaries kvdb.RBucket,
	chanID lnwire.ChannelID) (*ClientChanSummary, er.R) {
	chanSummaryBytes := chanSummaries.Get(chanID[:])
	if chanSummaryBytes == nil {
		return nil, ErrChannelNotRegistered.Default()
	}

	var summary ClientChanSummary
	err := summary.Decode(bytes.NewReader(chanSummaryBytes))
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

// putChanSummary stores a ClientChanSummary for the passed chanID.
func putChanSummary(chanSummaries kvdb.RwBucket, chanID lnwire.ChannelID,
	summary *ClientChanSummary) er.R {
	var b bytes.Buffer
	err := summary.Encode(&b)
	if err != nil {
		return err
	}

	return chanSummaries.Put(chanID[:], b.Bytes())
}

// getTower loads a Tower identified by its serialized tower id.
func getTower(towers kvdb.RBucket, id []byte) (*Tower, er.R) {
	towerBytes := towers.Get(id)
	if towerBytes == nil {
		return nil, ErrTowerNotFound.Default()
	}

	var tower Tower
	err := tower.Decode(bytes.NewReader(towerBytes))
	if err != nil {
		return nil, err
	}

	tower.ID = TowerIDFromBytes(id)

	return &tower, nil
}

// putTower stores a Tower identified by its serialized tower id.
func putTower(towers kvdb.RwBucket, tower *Tower) er.R {
	var b bytes.Buffer
	err := tower.Encode(&b)
	if err != nil {
		return err
	}

	return towers.Put(tower.ID.Bytes(), b.Bytes())
}

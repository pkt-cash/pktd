package contractcourt

import (
	"bytes"
	"io"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/btcutil/util"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/lnd/channeldb"
	"github.com/pkt-cash/pktd/lnd/channeldb/kvdb"
	"github.com/pkt-cash/pktd/lnd/input"
	"github.com/pkt-cash/pktd/lnd/lnwallet"
	"github.com/pkt-cash/pktd/wire"
)

// ContractResolutions is a wrapper struct around the two forms of resolutions
// we may need to carry out once a contract is closing: resolving the
// commitment output, and resolving any incoming+outgoing HTLC's still present
// in the commitment.
type ContractResolutions struct {
	// CommitHash is the txid of the commitment transaction.
	CommitHash chainhash.Hash

	// CommitResolution contains all data required to fully resolve a
	// commitment output.
	CommitResolution *lnwallet.CommitOutputResolution

	// HtlcResolutions contains all data required to fully resolve any
	// incoming+outgoing HTLC's present within the commitment transaction.
	HtlcResolutions lnwallet.HtlcResolutions

	// AnchorResolution contains the data required to sweep the anchor
	// output. If the channel type doesn't include anchors, the value of
	// this field will be nil.
	AnchorResolution *lnwallet.AnchorResolution
}

// IsEmpty returns true if the set of resolutions is "empty". A resolution is
// empty if: our commitment output has been trimmed, and we don't have any
// incoming or outgoing HTLC's active.
func (c *ContractResolutions) IsEmpty() bool {
	return c.CommitResolution == nil &&
		len(c.HtlcResolutions.IncomingHTLCs) == 0 &&
		len(c.HtlcResolutions.OutgoingHTLCs) == 0 &&
		c.AnchorResolution == nil
}

// ArbitratorLog is the primary source of persistent storage for the
// ChannelArbitrator. The log stores the current state of the
// ChannelArbitrator's internal state machine, any items that are required to
// properly make a state transition, and any unresolved contracts.
type ArbitratorLog interface {
	// TODO(roasbeef): document on interface the errors expected to be
	// returned

	// CurrentState returns the current state of the ChannelArbitrator. It
	// takes an optional database transaction, which will be used if it is
	// non-nil, otherwise the lookup will be done in its own transaction.
	CurrentState(tx kvdb.RTx) (ArbitratorState, er.R)

	// CommitState persists, the current state of the chain attendant.
	CommitState(ArbitratorState) er.R

	// InsertUnresolvedContracts inserts a set of unresolved contracts into
	// the log. The log will then persistently store each contract until
	// they've been swapped out, or resolved. It takes a set of report which
	// should be written to disk if as well if it is non-nil.
	InsertUnresolvedContracts(reports []*channeldb.ResolverReport,
		resolvers ...ContractResolver) er.R

	// FetchUnresolvedContracts returns all unresolved contracts that have
	// been previously written to the log.
	FetchUnresolvedContracts() ([]ContractResolver, er.R)

	// SwapContract performs an atomic swap of the old contract for the new
	// contract. This method is used when after a contract has been fully
	// resolved, it produces another contract that needs to be resolved.
	SwapContract(old, new ContractResolver) er.R

	// ResolveContract marks a contract as fully resolved. Once a contract
	// has been fully resolved, it is deleted from persistent storage.
	ResolveContract(ContractResolver) er.R

	// LogContractResolutions stores a complete contract resolution for the
	// contract under watch. This method will be called once the
	// ChannelArbitrator either force closes a channel, or detects that the
	// remote party has broadcast their commitment on chain.
	LogContractResolutions(*ContractResolutions) er.R

	// FetchContractResolutions fetches the set of previously stored
	// contract resolutions from persistent storage.
	FetchContractResolutions() (*ContractResolutions, er.R)

	// InsertConfirmedCommitSet stores the known set of active HTLCs at the
	// time channel closure. We'll use this to reconstruct our set of chain
	// actions anew based on the confirmed and pending commitment state.
	InsertConfirmedCommitSet(c *CommitSet) er.R

	// FetchConfirmedCommitSet fetches the known confirmed active HTLC set
	// from the database. It takes an optional database transaction, which
	// will be used if it is non-nil, otherwise the lookup will be done in
	// its own transaction.
	FetchConfirmedCommitSet(tx kvdb.RTx) (*CommitSet, er.R)

	// FetchChainActions attempts to fetch the set of previously stored
	// chain actions. We'll use this upon restart to properly advance our
	// state machine forward.
	//
	// NOTE: This method only exists in order to be able to serve nodes had
	// channels in the process of closing before the CommitSet struct was
	// introduced.
	FetchChainActions() (ChainActionMap, er.R)

	// WipeHistory is to be called ONLY once *all* contracts have been
	// fully resolved, and the channel closure if finalized. This method
	// will delete all on-disk state within the persistent log.
	WipeHistory() er.R
}

// ArbitratorState is an enum that details the current state of the
// ChannelArbitrator's state machine.
type ArbitratorState uint8

const (
	// StateDefault is the default state. In this state, no major actions
	// need to be executed.
	StateDefault ArbitratorState = 0

	// StateBroadcastCommit is a state that indicates that the attendant
	// has decided to broadcast the commitment transaction, but hasn't done
	// so yet.
	StateBroadcastCommit ArbitratorState = 1

	// StateCommitmentBroadcasted is a state that indicates that the
	// attendant has broadcasted the commitment transaction, and is now
	// waiting for it to confirm.
	StateCommitmentBroadcasted ArbitratorState = 6

	// StateContractClosed is a state that indicates the contract has
	// already been "closed", meaning the commitment is confirmed on chain.
	// At this point, we can now examine our active contracts, in order to
	// create the proper resolver for each one.
	StateContractClosed ArbitratorState = 2

	// StateWaitingFullResolution is a state that indicates that the
	// commitment transaction has been confirmed, and the attendant is now
	// waiting for all unresolved contracts to be fully resolved.
	StateWaitingFullResolution ArbitratorState = 3

	// StateFullyResolved is the final state of the attendant. In this
	// state, all related contracts have been resolved, and the attendant
	// can now be garbage collected.
	StateFullyResolved ArbitratorState = 4

	// StateError is the only error state of the resolver. If we enter this
	// state, then we cannot proceed with manual intervention as a state
	// transition failed.
	StateError ArbitratorState = 5
)

// String returns a human readable string describing the ArbitratorState.
func (a ArbitratorState) String() string {
	switch a {
	case StateDefault:
		return "StateDefault"

	case StateBroadcastCommit:
		return "StateBroadcastCommit"

	case StateCommitmentBroadcasted:
		return "StateCommitmentBroadcasted"

	case StateContractClosed:
		return "StateContractClosed"

	case StateWaitingFullResolution:
		return "StateWaitingFullResolution"

	case StateFullyResolved:
		return "StateFullyResolved"

	case StateError:
		return "StateError"

	default:
		return "unknown state"
	}
}

// resolverType is an enum that enumerates the various types of resolvers. When
// writing resolvers to disk, we prepend this to the raw bytes stored. This
// allows us to properly decode the resolver into the proper type.
type resolverType uint8

const (
	// resolverTimeout is the type of a resolver that's tasked with
	// resolving an outgoing HTLC that is very close to timing out.
	resolverTimeout resolverType = 0

	// resolverSuccess is the type of a resolver that's tasked with
	// resolving an incoming HTLC that we already know the preimage of.
	resolverSuccess resolverType = 1

	// resolverOutgoingContest is the type of a resolver that's tasked with
	// resolving an outgoing HTLC that hasn't yet timed out.
	resolverOutgoingContest resolverType = 2

	// resolverIncomingContest is the type of a resolver that's tasked with
	// resolving an incoming HTLC that we don't yet know the preimage to.
	resolverIncomingContest resolverType = 3

	// resolverUnilateralSweep is the type of resolver that's tasked with
	// sweeping out direct commitment output form the remote party's
	// commitment transaction.
	resolverUnilateralSweep resolverType = 4
)

// resolverIDLen is the size of the resolver ID key. This is 36 bytes as we get
// 32 bytes from the hash of the prev tx, and 4 bytes for the output index.
const resolverIDLen = 36

// resolverID is a key that uniquely identifies a resolver within a particular
// chain. For this value we use the full outpoint of the resolver.
type resolverID [resolverIDLen]byte

// newResolverID returns a resolverID given the outpoint of a contract.
func newResolverID(op wire.OutPoint) resolverID {
	var r resolverID

	copy(r[:], op.Hash[:])

	endian.PutUint32(r[32:], op.Index)

	return r
}

// logScope is a key that we use to scope the storage of a ChannelArbitrator
// within the global log. We use this key to create a unique bucket within the
// database and ensure that we don't have any key collisions. The log's scope
// is define as: chainHash || chanPoint, where chanPoint is the chan point of
// the original channel.
type logScope [32 + 36]byte

// newLogScope creates a new logScope key from the passed chainhash and
// chanPoint.
func newLogScope(chain chainhash.Hash, op wire.OutPoint) (*logScope, er.R) {
	var l logScope
	b := bytes.NewBuffer(l[0:0])

	if _, err := b.Write(chain[:]); err != nil {
		return nil, er.E(err)
	}
	if _, err := b.Write(op.Hash[:]); err != nil {
		return nil, er.E(err)
	}

	if err := util.WriteBin(b, endian, op.Index); err != nil {
		return nil, err
	}

	return &l, nil
}

var (
	// stateKey is the key that we use to store the current state of the
	// arbitrator.
	stateKey = []byte("state")

	// contractsBucketKey is the bucket within the logScope that will store
	// all the active unresolved contracts.
	contractsBucketKey = []byte("contractkey")

	// resolutionsKey is the key under the logScope that we'll use to store
	// the full set of resolutions for a channel.
	resolutionsKey = []byte("resolutions")

	// anchorResolutionKey is the key under the logScope that we'll use to
	// store the anchor resolution, if any.
	anchorResolutionKey = []byte("anchor-resolution")

	// actionsBucketKey is the key under the logScope that we'll use to
	// store all chain actions once they're determined.
	actionsBucketKey = []byte("chain-actions")

	// commitSetKey is the primary key under the logScope that we'll use to
	// store the confirmed active HTLC sets once we learn that a channel
	// has closed out on chain.
	commitSetKey = []byte("commit-set")
)

var (
	// errScopeBucketNoExist is returned when we can't find the proper
	// bucket for an arbitrator's scope.
	errScopeBucketNoExist = Err.CodeWithDetail("errScopeBucketNoExist", "scope bucket not found")

	// errNoContracts is returned when no contracts are found within the
	// log.
	errNoContracts = Err.CodeWithDetail("errNoContracts", "no stored contracts")

	// errNoResolutions is returned when the log doesn't contain any active
	// chain resolutions.
	errNoResolutions = Err.CodeWithDetail("errNoResolutions", "no contract resolutions exist")

	// errNoActions is retuned when the log doesn't contain any stored
	// chain actions.
	errNoActions = Err.CodeWithDetail("errNoActions", "no chain actions exist")

	// errNoCommitSet is return when the log doesn't contained a CommitSet.
	// This can happen if the channel hasn't closed yet, or a client is
	// running an older version that didn't yet write this state.
	errNoCommitSet = Err.CodeWithDetail("errNoCommitSet", "no commit set exists")
)

// boltArbitratorLog is an implementation of the ArbitratorLog interface backed
// by a bolt DB instance.
type boltArbitratorLog struct {
	db kvdb.Backend

	cfg ChannelArbitratorConfig

	scopeKey logScope
}

// newBoltArbitratorLog returns a new instance of the boltArbitratorLog given
// an arbitrator config, and the items needed to create its log scope.
func newBoltArbitratorLog(db kvdb.Backend, cfg ChannelArbitratorConfig,
	chainHash chainhash.Hash, chanPoint wire.OutPoint) (*boltArbitratorLog, er.R) {
	scope, err := newLogScope(chainHash, chanPoint)
	if err != nil {
		return nil, err
	}

	return &boltArbitratorLog{
		db:       db,
		cfg:      cfg,
		scopeKey: *scope,
	}, nil
}

// A compile time check to ensure boltArbitratorLog meets the ArbitratorLog
// interface.
var _ ArbitratorLog = (*boltArbitratorLog)(nil)

func fetchContractReadBucket(tx kvdb.RTx, scopeKey []byte) (kvdb.RBucket, er.R) {
	scopeBucket := tx.ReadBucket(scopeKey)
	if scopeBucket == nil {
		return nil, errScopeBucketNoExist.Default()
	}

	contractBucket := scopeBucket.NestedReadBucket(contractsBucketKey)
	if contractBucket == nil {
		return nil, errNoContracts.Default()
	}

	return contractBucket, nil
}

func fetchContractWriteBucket(tx kvdb.RwTx, scopeKey []byte) (kvdb.RwBucket, er.R) {
	scopeBucket, err := tx.CreateTopLevelBucket(scopeKey)
	if err != nil {
		return nil, err
	}

	contractBucket, err := scopeBucket.CreateBucketIfNotExists(
		contractsBucketKey,
	)
	if err != nil {
		return nil, err
	}

	return contractBucket, nil
}

// writeResolver is a helper method that writes a contract resolver and stores
// it it within the passed contractBucket using its unique resolutionsKey key.
func (b *boltArbitratorLog) writeResolver(contractBucket kvdb.RwBucket,
	res ContractResolver) er.R {
	// Only persist resolvers that are stateful. Stateless resolvers don't
	// expose a resolver key.
	resKey := res.ResolverKey()
	if resKey == nil {
		return nil
	}

	// First, we'll write to the buffer the type of this resolver. Using
	// this byte, we can later properly deserialize the resolver properly.
	var (
		buf   bytes.Buffer
		rType resolverType
	)
	switch res.(type) {
	case *htlcTimeoutResolver:
		rType = resolverTimeout
	case *htlcSuccessResolver:
		rType = resolverSuccess
	case *htlcOutgoingContestResolver:
		rType = resolverOutgoingContest
	case *htlcIncomingContestResolver:
		rType = resolverIncomingContest
	case *commitSweepResolver:
		rType = resolverUnilateralSweep
	}
	if _, err := buf.Write([]byte{byte(rType)}); err != nil {
		return er.E(err)
	}

	// With the type of the resolver written, we can then write out the raw
	// bytes of the resolver itself.
	if err := res.Encode(&buf); err != nil {
		return err
	}

	return contractBucket.Put(resKey, buf.Bytes())
}

// CurrentState returns the current state of the ChannelArbitrator. It takes an
// optional database transaction, which will be used if it is non-nil, otherwise
// the lookup will be done in its own transaction.
//
// NOTE: Part of the ContractResolver interface.
func (b *boltArbitratorLog) CurrentState(tx kvdb.RTx) (ArbitratorState, er.R) {
	var (
		s   ArbitratorState
		err er.R
	)

	if tx != nil {
		s, err = b.currentState(tx)
	} else {
		err = kvdb.View(b.db, func(tx kvdb.RTx) er.R {
			s, err = b.currentState(tx)
			return err
		}, func() {
			s = 0
		})
	}

	if err != nil && !errScopeBucketNoExist.Is(err) {
		return s, err
	}

	return s, nil
}

func (b *boltArbitratorLog) currentState(tx kvdb.RTx) (ArbitratorState, er.R) {
	scopeBucket := tx.ReadBucket(b.scopeKey[:])
	if scopeBucket == nil {
		return 0, errScopeBucketNoExist.Default()
	}

	stateBytes := scopeBucket.Get(stateKey)
	if stateBytes == nil {
		return 0, nil
	}

	return ArbitratorState(stateBytes[0]), nil
}

// CommitState persists, the current state of the chain attendant.
//
// NOTE: Part of the ContractResolver interface.
func (b *boltArbitratorLog) CommitState(s ArbitratorState) er.R {
	return kvdb.Batch(b.db, func(tx kvdb.RwTx) er.R {
		scopeBucket, err := tx.CreateTopLevelBucket(b.scopeKey[:])
		if err != nil {
			return err
		}

		return scopeBucket.Put(stateKey[:], []byte{uint8(s)})
	})
}

// FetchUnresolvedContracts returns all unresolved contracts that have been
// previously written to the log.
//
// NOTE: Part of the ContractResolver interface.
func (b *boltArbitratorLog) FetchUnresolvedContracts() ([]ContractResolver, er.R) {
	resolverCfg := ResolverConfig{
		ChannelArbitratorConfig: b.cfg,
		Checkpoint:              b.checkpointContract,
	}
	var contracts []ContractResolver
	err := kvdb.View(b.db, func(tx kvdb.RTx) er.R {
		contractBucket, err := fetchContractReadBucket(tx, b.scopeKey[:])
		if err != nil {
			return err
		}

		return contractBucket.ForEach(func(resKey, resBytes []byte) er.R {
			if len(resKey) != resolverIDLen {
				return nil
			}

			var res ContractResolver

			// We'll snip off the first byte of the raw resolver
			// bytes in order to extract what type of resolver
			// we're about to encode.
			resType := resolverType(resBytes[0])

			// Then we'll create a reader using the remaining
			// bytes.
			resReader := bytes.NewReader(resBytes[1:])

			switch resType {
			case resolverTimeout:
				res, err = newTimeoutResolverFromReader(
					resReader, resolverCfg,
				)

			case resolverSuccess:
				res, err = newSuccessResolverFromReader(
					resReader, resolverCfg,
				)

			case resolverOutgoingContest:
				res, err = newOutgoingContestResolverFromReader(
					resReader, resolverCfg,
				)

			case resolverIncomingContest:
				res, err = newIncomingContestResolverFromReader(
					resReader, resolverCfg,
				)

			case resolverUnilateralSweep:
				res, err = newCommitSweepResolverFromReader(
					resReader, resolverCfg,
				)

			default:
				return er.Errorf("unknown resolver type: %v", resType)
			}

			if err != nil {
				return err
			}

			contracts = append(contracts, res)
			return nil
		})
	}, func() {
		contracts = nil
	})
	if err != nil && !errScopeBucketNoExist.Is(err) && !errNoContracts.Is(err) {
		return nil, err
	}

	return contracts, nil
}

// InsertUnresolvedContracts inserts a set of unresolved contracts into the
// log. The log will then persistently store each contract until they've been
// swapped out, or resolved.
//
// NOTE: Part of the ContractResolver interface.
func (b *boltArbitratorLog) InsertUnresolvedContracts(reports []*channeldb.ResolverReport,
	resolvers ...ContractResolver) er.R {
	return kvdb.Batch(b.db, func(tx kvdb.RwTx) er.R {
		contractBucket, err := fetchContractWriteBucket(tx, b.scopeKey[:])
		if err != nil {
			return err
		}

		for _, resolver := range resolvers {
			err = b.writeResolver(contractBucket, resolver)
			if err != nil {
				return err
			}
		}

		// Persist any reports that are present.
		for _, report := range reports {
			err := b.cfg.PutResolverReport(tx, report)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// SwapContract performs an atomic swap of the old contract for the new
// contract. This method is used when after a contract has been fully resolved,
// it produces another contract that needs to be resolved.
//
// NOTE: Part of the ContractResolver interface.
func (b *boltArbitratorLog) SwapContract(oldContract, newContract ContractResolver) er.R {
	return kvdb.Batch(b.db, func(tx kvdb.RwTx) er.R {
		contractBucket, err := fetchContractWriteBucket(tx, b.scopeKey[:])
		if err != nil {
			return err
		}

		oldContractkey := oldContract.ResolverKey()
		if err := contractBucket.Delete(oldContractkey); err != nil {
			return err
		}

		return b.writeResolver(contractBucket, newContract)
	})
}

// ResolveContract marks a contract as fully resolved. Once a contract has been
// fully resolved, it is deleted from persistent storage.
//
// NOTE: Part of the ContractResolver interface.
func (b *boltArbitratorLog) ResolveContract(res ContractResolver) er.R {
	return kvdb.Batch(b.db, func(tx kvdb.RwTx) er.R {
		contractBucket, err := fetchContractWriteBucket(tx, b.scopeKey[:])
		if err != nil {
			return err
		}

		resKey := res.ResolverKey()
		return contractBucket.Delete(resKey)
	})
}

// LogContractResolutions stores a set of chain actions which are derived from
// our set of active contracts, and the on-chain state. We'll write this et of
// cations when: we decide to go on-chain to resolve a contract, or we detect
// that the remote party has gone on-chain.
//
// NOTE: Part of the ContractResolver interface.
func (b *boltArbitratorLog) LogContractResolutions(c *ContractResolutions) er.R {
	return kvdb.Batch(b.db, func(tx kvdb.RwTx) er.R {
		scopeBucket, err := tx.CreateTopLevelBucket(b.scopeKey[:])
		if err != nil {
			return err
		}

		var b bytes.Buffer

		if _, err := b.Write(c.CommitHash[:]); err != nil {
			return er.E(err)
		}

		// First, we'll write out the commit output's resolution.
		if c.CommitResolution == nil {
			if err := util.WriteBin(&b, endian, false); err != nil {
				return err
			}
		} else {
			if err := util.WriteBin(&b, endian, true); err != nil {
				return err
			}
			errr := encodeCommitResolution(&b, c.CommitResolution)
			if errr != nil {
				return errr
			}
		}

		// With the output for the commitment transaction written, we
		// can now write out the resolutions for the incoming and
		// outgoing HTLC's.
		numIncoming := uint32(len(c.HtlcResolutions.IncomingHTLCs))
		if err := util.WriteBin(&b, endian, numIncoming); err != nil {
			return err
		}
		for _, htlc := range c.HtlcResolutions.IncomingHTLCs {
			err := encodeIncomingResolution(&b, &htlc)
			if err != nil {
				return err
			}
		}
		numOutgoing := uint32(len(c.HtlcResolutions.OutgoingHTLCs))
		if err := util.WriteBin(&b, endian, numOutgoing); err != nil {
			return err
		}
		for _, htlc := range c.HtlcResolutions.OutgoingHTLCs {
			err := encodeOutgoingResolution(&b, &htlc)
			if err != nil {
				return err
			}
		}

		err = scopeBucket.Put(resolutionsKey, b.Bytes())
		if err != nil {
			return err
		}

		// Write out the anchor resolution if present.
		if c.AnchorResolution != nil {
			var b bytes.Buffer
			err := encodeAnchorResolution(&b, c.AnchorResolution)
			if err != nil {
				return err
			}

			err = scopeBucket.Put(anchorResolutionKey, b.Bytes())
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// FetchContractResolutions fetches the set of previously stored contract
// resolutions from persistent storage.
//
// NOTE: Part of the ContractResolver interface.
func (b *boltArbitratorLog) FetchContractResolutions() (*ContractResolutions, er.R) {
	var c *ContractResolutions
	err := kvdb.View(b.db, func(tx kvdb.RTx) er.R {
		scopeBucket := tx.ReadBucket(b.scopeKey[:])
		if scopeBucket == nil {
			return errScopeBucketNoExist.Default()
		}

		resolutionBytes := scopeBucket.Get(resolutionsKey)
		if resolutionBytes == nil {
			return errNoResolutions.Default()
		}

		resReader := bytes.NewReader(resolutionBytes)

		_, err := util.ReadFull(resReader, c.CommitHash[:])
		if err != nil {
			return err
		}

		// First, we'll attempt to read out the commit resolution (if
		// it exists).
		var haveCommitRes bool
		err = util.ReadBin(resReader, endian, &haveCommitRes)
		if err != nil {
			return err
		}
		if haveCommitRes {
			c.CommitResolution = &lnwallet.CommitOutputResolution{}
			err = decodeCommitResolution(
				resReader, c.CommitResolution,
			)
			if err != nil {
				return err
			}
		}

		var (
			numIncoming uint32
			numOutgoing uint32
		)

		// Next, we'll read out the incoming and outgoing HTLC
		// resolutions.
		err = util.ReadBin(resReader, endian, &numIncoming)
		if err != nil {
			return err
		}
		c.HtlcResolutions.IncomingHTLCs = make([]lnwallet.IncomingHtlcResolution, numIncoming)
		for i := uint32(0); i < numIncoming; i++ {
			err := decodeIncomingResolution(
				resReader, &c.HtlcResolutions.IncomingHTLCs[i],
			)
			if err != nil {
				return err
			}
		}

		err = util.ReadBin(resReader, endian, &numOutgoing)
		if err != nil {
			return err
		}
		c.HtlcResolutions.OutgoingHTLCs = make([]lnwallet.OutgoingHtlcResolution, numOutgoing)
		for i := uint32(0); i < numOutgoing; i++ {
			err := decodeOutgoingResolution(
				resReader, &c.HtlcResolutions.OutgoingHTLCs[i],
			)
			if err != nil {
				return err
			}
		}

		anchorResBytes := scopeBucket.Get(anchorResolutionKey)
		if anchorResBytes != nil {
			c.AnchorResolution = &lnwallet.AnchorResolution{}
			resReader := bytes.NewReader(anchorResBytes)
			err := decodeAnchorResolution(
				resReader, c.AnchorResolution,
			)
			if err != nil {
				return err
			}
		}

		return nil
	}, func() {
		c = &ContractResolutions{}
	})
	if err != nil {
		return nil, err
	}

	return c, err
}

// FetchChainActions attempts to fetch the set of previously stored chain
// actions. We'll use this upon restart to properly advance our state machine
// forward.
//
// NOTE: Part of the ContractResolver interface.
func (b *boltArbitratorLog) FetchChainActions() (ChainActionMap, er.R) {
	var actionsMap ChainActionMap

	err := kvdb.View(b.db, func(tx kvdb.RTx) er.R {
		scopeBucket := tx.ReadBucket(b.scopeKey[:])
		if scopeBucket == nil {
			return errScopeBucketNoExist.Default()
		}

		actionsBucket := scopeBucket.NestedReadBucket(actionsBucketKey)
		if actionsBucket == nil {
			return errNoActions.Default()
		}

		return actionsBucket.ForEach(func(action, htlcBytes []byte) er.R {
			if htlcBytes == nil {
				return nil
			}

			chainAction := ChainAction(action[0])

			htlcReader := bytes.NewReader(htlcBytes)
			htlcs, err := channeldb.DeserializeHtlcs(htlcReader)
			if err != nil {
				return err
			}

			actionsMap[chainAction] = htlcs

			return nil
		})
	}, func() {
		actionsMap = make(ChainActionMap)
	})
	if err != nil {
		return nil, err
	}

	return actionsMap, nil
}

// InsertConfirmedCommitSet stores the known set of active HTLCs at the time
// channel closure. We'll use this to reconstruct our set of chain actions anew
// based on the confirmed and pending commitment state.
//
// NOTE: Part of the ContractResolver interface.
func (b *boltArbitratorLog) InsertConfirmedCommitSet(c *CommitSet) er.R {
	return kvdb.Batch(b.db, func(tx kvdb.RwTx) er.R {
		scopeBucket, err := tx.CreateTopLevelBucket(b.scopeKey[:])
		if err != nil {
			return err
		}

		var b bytes.Buffer
		if err := encodeCommitSet(&b, c); err != nil {
			return err
		}

		return scopeBucket.Put(commitSetKey, b.Bytes())
	})
}

// FetchConfirmedCommitSet fetches the known confirmed active HTLC set from the
// database. It takes an optional database transaction, which will be used if it
// is non-nil, otherwise the lookup will be done in its own transaction.
//
// NOTE: Part of the ContractResolver interface.
func (b *boltArbitratorLog) FetchConfirmedCommitSet(tx kvdb.RTx) (*CommitSet, er.R) {
	if tx != nil {
		return b.fetchConfirmedCommitSet(tx)
	}

	var c *CommitSet
	err := kvdb.View(b.db, func(tx kvdb.RTx) er.R {
		var err er.R
		c, err = b.fetchConfirmedCommitSet(tx)
		return err
	}, func() {
		c = nil
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (b *boltArbitratorLog) fetchConfirmedCommitSet(tx kvdb.RTx) (*CommitSet, er.R) {
	scopeBucket := tx.ReadBucket(b.scopeKey[:])
	if scopeBucket == nil {
		return nil, errScopeBucketNoExist.Default()
	}

	commitSetBytes := scopeBucket.Get(commitSetKey)
	if commitSetBytes == nil {
		return nil, errNoCommitSet.Default()
	}

	return decodeCommitSet(bytes.NewReader(commitSetBytes))
}

// WipeHistory is to be called ONLY once *all* contracts have been fully
// resolved, and the channel closure if finalized. This method will delete all
// on-disk state within the persistent log.
//
// NOTE: Part of the ContractResolver interface.
func (b *boltArbitratorLog) WipeHistory() er.R {
	return kvdb.Update(b.db, func(tx kvdb.RwTx) er.R {
		scopeBucket, err := tx.CreateTopLevelBucket(b.scopeKey[:])
		if err != nil {
			return err
		}

		// Once we have the main top-level bucket, we'll delete the key
		// that stores the state of the arbitrator.
		if err := scopeBucket.Delete(stateKey[:]); err != nil {
			return err
		}

		// Next, we'll delete any lingering contract state within the
		// contracts bucket by removing the bucket itself.
		err = scopeBucket.DeleteNestedBucket(contractsBucketKey)
		if err != nil && !kvdb.ErrBucketNotFound.Is(err) {
			return err
		}

		// Next, we'll delete storage of any lingering contract
		// resolutions.
		if err := scopeBucket.Delete(resolutionsKey); err != nil {
			return err
		}

		// We'll delete any chain actions that are still stored by
		// removing the enclosing bucket.
		err = scopeBucket.DeleteNestedBucket(actionsBucketKey)
		if err != nil && !kvdb.ErrBucketNotFound.Is(err) {
			return err
		}

		// Finally, we'll delete the enclosing bucket itself.
		return tx.DeleteTopLevelBucket(b.scopeKey[:])
	}, func() {})
}

// checkpointContract is a private method that will be fed into
// ContractResolver instances to checkpoint their state once they reach
// milestones during contract resolution. If the report provided is non-nil,
// it should also be recorded.
func (b *boltArbitratorLog) checkpointContract(c ContractResolver,
	reports ...*channeldb.ResolverReport) er.R {
	return kvdb.Update(b.db, func(tx kvdb.RwTx) er.R {
		contractBucket, err := fetchContractWriteBucket(tx, b.scopeKey[:])
		if err != nil {
			return err
		}

		if err := b.writeResolver(contractBucket, c); err != nil {
			return err
		}

		for _, report := range reports {
			if err := b.cfg.PutResolverReport(tx, report); err != nil {
				return err
			}
		}

		return nil
	}, func() {})
}

func encodeIncomingResolution(w io.Writer, i *lnwallet.IncomingHtlcResolution) er.R {
	if _, err := util.Write(w, i.Preimage[:]); err != nil {
		return err
	}

	if i.SignedSuccessTx == nil {
		if err := util.WriteBin(w, endian, false); err != nil {
			return err
		}
	} else {
		if err := util.WriteBin(w, endian, true); err != nil {
			return err
		}

		if err := i.SignedSuccessTx.Serialize(w); err != nil {
			return err
		}
	}

	if err := util.WriteBin(w, endian, i.CsvDelay); err != nil {
		return err
	}
	if _, err := util.Write(w, i.ClaimOutpoint.Hash[:]); err != nil {
		return err
	}
	if err := util.WriteBin(w, endian, i.ClaimOutpoint.Index); err != nil {
		return err
	}
	err := input.WriteSignDescriptor(w, &i.SweepSignDesc)
	if err != nil {
		return err
	}

	return nil
}

func decodeIncomingResolution(r io.Reader, h *lnwallet.IncomingHtlcResolution) er.R {
	if _, err := util.ReadFull(r, h.Preimage[:]); err != nil {
		return err
	}

	var txPresent bool
	if err := util.ReadBin(r, endian, &txPresent); err != nil {
		return err
	}
	if txPresent {
		h.SignedSuccessTx = &wire.MsgTx{}
		if err := h.SignedSuccessTx.Deserialize(r); err != nil {
			return err
		}
	}

	err := util.ReadBin(r, endian, &h.CsvDelay)
	if err != nil {
		return err
	}
	_, err = util.ReadFull(r, h.ClaimOutpoint.Hash[:])
	if err != nil {
		return err
	}
	err = util.ReadBin(r, endian, &h.ClaimOutpoint.Index)
	if err != nil {
		return err
	}

	return input.ReadSignDescriptor(r, &h.SweepSignDesc)
}

func encodeOutgoingResolution(w io.Writer, o *lnwallet.OutgoingHtlcResolution) er.R {
	if err := util.WriteBin(w, endian, o.Expiry); err != nil {
		return err
	}

	if o.SignedTimeoutTx == nil {
		if err := util.WriteBin(w, endian, false); err != nil {
			return err
		}
	} else {
		if err := util.WriteBin(w, endian, true); err != nil {
			return err
		}

		if err := o.SignedTimeoutTx.Serialize(w); err != nil {
			return err
		}
	}

	if err := util.WriteBin(w, endian, o.CsvDelay); err != nil {
		return err
	}
	if _, err := util.Write(w, o.ClaimOutpoint.Hash[:]); err != nil {
		return err
	}
	if err := util.WriteBin(w, endian, o.ClaimOutpoint.Index); err != nil {
		return err
	}

	return input.WriteSignDescriptor(w, &o.SweepSignDesc)
}

func decodeOutgoingResolution(r io.Reader, o *lnwallet.OutgoingHtlcResolution) er.R {
	err := util.ReadBin(r, endian, &o.Expiry)
	if err != nil {
		return err
	}

	var txPresent bool
	if err := util.ReadBin(r, endian, &txPresent); err != nil {
		return err
	}
	if txPresent {
		o.SignedTimeoutTx = &wire.MsgTx{}
		if err := o.SignedTimeoutTx.Deserialize(r); err != nil {
			return err
		}
	}

	err = util.ReadBin(r, endian, &o.CsvDelay)
	if err != nil {
		return err
	}
	_, err = util.ReadFull(r, o.ClaimOutpoint.Hash[:])
	if err != nil {
		return err
	}
	err = util.ReadBin(r, endian, &o.ClaimOutpoint.Index)
	if err != nil {
		return err
	}

	return input.ReadSignDescriptor(r, &o.SweepSignDesc)
}

func encodeCommitResolution(w io.Writer,
	c *lnwallet.CommitOutputResolution) er.R {
	if _, err := util.Write(w, c.SelfOutPoint.Hash[:]); err != nil {
		return err
	}
	err := util.WriteBin(w, endian, c.SelfOutPoint.Index)
	if err != nil {
		return err
	}

	err = input.WriteSignDescriptor(w, &c.SelfOutputSignDesc)
	if err != nil {
		return err
	}

	return util.WriteBin(w, endian, c.MaturityDelay)
}

func decodeCommitResolution(r io.Reader,
	c *lnwallet.CommitOutputResolution) er.R {
	_, err := util.ReadFull(r, c.SelfOutPoint.Hash[:])
	if err != nil {
		return err
	}
	err = util.ReadBin(r, endian, &c.SelfOutPoint.Index)
	if err != nil {
		return err
	}

	err = input.ReadSignDescriptor(r, &c.SelfOutputSignDesc)
	if err != nil {
		return err
	}

	return util.ReadBin(r, endian, &c.MaturityDelay)
}

func encodeAnchorResolution(w io.Writer,
	a *lnwallet.AnchorResolution) er.R {
	if _, err := util.Write(w, a.CommitAnchor.Hash[:]); err != nil {
		return err
	}
	err := util.WriteBin(w, endian, a.CommitAnchor.Index)
	if err != nil {
		return err
	}

	return input.WriteSignDescriptor(w, &a.AnchorSignDescriptor)
}

func decodeAnchorResolution(r io.Reader,
	a *lnwallet.AnchorResolution) er.R {
	_, err := util.ReadFull(r, a.CommitAnchor.Hash[:])
	if err != nil {
		return err
	}
	err = util.ReadBin(r, endian, &a.CommitAnchor.Index)
	if err != nil {
		return err
	}

	return input.ReadSignDescriptor(r, &a.AnchorSignDescriptor)
}

func encodeHtlcSetKey(w io.Writer, h *HtlcSetKey) er.R {
	err := util.WriteBin(w, endian, h.IsRemote)
	if err != nil {
		return err
	}
	return util.WriteBin(w, endian, h.IsPending)
}

func encodeCommitSet(w io.Writer, c *CommitSet) er.R {
	if err := encodeHtlcSetKey(w, c.ConfCommitKey); err != nil {
		return err
	}

	numSets := uint8(len(c.HtlcSets))
	if err := util.WriteBin(w, endian, numSets); err != nil {
		return err
	}

	for htlcSetKey, htlcs := range c.HtlcSets {
		htlcSetKey := htlcSetKey
		if err := encodeHtlcSetKey(w, &htlcSetKey); err != nil {
			return err
		}

		if err := channeldb.SerializeHtlcs(w, htlcs...); err != nil {
			return err
		}
	}

	return nil
}

func decodeHtlcSetKey(r io.Reader, h *HtlcSetKey) er.R {
	err := util.ReadBin(r, endian, &h.IsRemote)
	if err != nil {
		return err
	}

	return util.ReadBin(r, endian, &h.IsPending)
}

func decodeCommitSet(r io.Reader) (*CommitSet, er.R) {
	c := &CommitSet{
		ConfCommitKey: &HtlcSetKey{},
		HtlcSets:      make(map[HtlcSetKey][]channeldb.HTLC),
	}

	if err := decodeHtlcSetKey(r, c.ConfCommitKey); err != nil {
		return nil, err
	}

	var numSets uint8
	if err := util.ReadBin(r, endian, &numSets); err != nil {
		return nil, err
	}

	for i := uint8(0); i < numSets; i++ {
		var htlcSetKey HtlcSetKey
		if err := decodeHtlcSetKey(r, &htlcSetKey); err != nil {
			return nil, err
		}

		htlcs, err := channeldb.DeserializeHtlcs(r)
		if err != nil {
			return nil, err
		}

		c.HtlcSets[htlcSetKey] = htlcs
	}

	return c, nil
}

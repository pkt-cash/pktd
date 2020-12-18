package chainntnfs

import (
	"bytes"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/lnd/channeldb"
	"github.com/pkt-cash/pktd/lnd/channeldb/kvdb"
	"github.com/pkt-cash/pktd/pktlog/log"
)

var (
	// spendHintBucket is the name of the bucket which houses the height
	// hint for outpoints. Each height hint represents the earliest height
	// at which its corresponding outpoint could have been spent within.
	spendHintBucket = []byte("spend-hints")

	// confirmHintBucket is the name of the bucket which houses the height
	// hints for transactions. Each height hint represents the earliest
	// height at which its corresponding transaction could have been
	// confirmed within.
	confirmHintBucket = []byte("confirm-hints")

	Err = er.NewErrorType("lnd.chainntnfs")

	// ErrCorruptedHeightHintCache indicates that the on-disk bucketing
	// structure has altered since the height hint cache instance was
	// initialized.
	ErrCorruptedHeightHintCache = Err.CodeWithDetail("ErrCorruptedHeightHintCache",
		"height hint cache has been corrupted")

	// ErrSpendHintNotFound is an error returned when a spend hint for an
	// outpoint was not found.
	ErrSpendHintNotFound = Err.CodeWithDetail("ErrSpendHintNotFound",
		"spend hint not found")

	// ErrConfirmHintNotFound is an error returned when a confirm hint for a
	// transaction was not found.
	ErrConfirmHintNotFound = Err.CodeWithDetail("ErrConfirmHintNotFound",
		"confirm hint not found")
)

// CacheConfig contains the HeightHintCache configuration
type CacheConfig struct {
	// QueryDisable prevents reliance on the Height Hint Cache.  This is
	// necessary to recover from an edge case when the height recorded in
	// the cache is higher than the actual height of a spend, causing a
	// channel to become "stuck" in a pending close state.
	QueryDisable bool
}

// SpendHintCache is an interface whose duty is to cache spend hints for
// outpoints. A spend hint is defined as the earliest height in the chain at
// which an outpoint could have been spent within.
type SpendHintCache interface {
	// CommitSpendHint commits a spend hint for the outpoints to the cache.
	CommitSpendHint(height uint32, spendRequests ...SpendRequest) er.R

	// QuerySpendHint returns the latest spend hint for an outpoint.
	// ErrSpendHintNotFound is returned if a spend hint does not exist
	// within the cache for the outpoint.
	QuerySpendHint(spendRequest SpendRequest) (uint32, er.R)

	// PurgeSpendHint removes the spend hint for the outpoints from the
	// cache.
	PurgeSpendHint(spendRequests ...SpendRequest) er.R
}

// ConfirmHintCache is an interface whose duty is to cache confirm hints for
// transactions. A confirm hint is defined as the earliest height in the chain
// at which a transaction could have been included in a block.
type ConfirmHintCache interface {
	// CommitConfirmHint commits a confirm hint for the transactions to the
	// cache.
	CommitConfirmHint(height uint32, confRequests ...ConfRequest) er.R

	// QueryConfirmHint returns the latest confirm hint for a transaction
	// hash. ErrConfirmHintNotFound is returned if a confirm hint does not
	// exist within the cache for the transaction hash.
	QueryConfirmHint(confRequest ConfRequest) (uint32, er.R)

	// PurgeConfirmHint removes the confirm hint for the transactions from
	// the cache.
	PurgeConfirmHint(confRequests ...ConfRequest) er.R
}

// HeightHintCache is an implementation of the SpendHintCache and
// ConfirmHintCache interfaces backed by a channeldb DB instance where the hints
// will be stored.
type HeightHintCache struct {
	cfg CacheConfig
	db  *channeldb.DB
}

// Compile-time checks to ensure HeightHintCache satisfies the SpendHintCache
// and ConfirmHintCache interfaces.
var (
	_ SpendHintCache   = (*HeightHintCache)(nil)
	_ ConfirmHintCache = (*HeightHintCache)(nil)
)

// NewHeightHintCache returns a new height hint cache backed by a database.
func NewHeightHintCache(cfg CacheConfig, db *channeldb.DB) (*HeightHintCache, er.R) {
	cache := &HeightHintCache{cfg, db}
	if err := cache.initBuckets(); err != nil {
		return nil, err
	}

	return cache, nil
}

// initBuckets ensures that the primary buckets used by the circuit are
// initialized so that we can assume their existence after startup.
func (c *HeightHintCache) initBuckets() er.R {
	return kvdb.Batch(c.db.Backend, func(tx kvdb.RwTx) er.R {
		_, err := tx.CreateTopLevelBucket(spendHintBucket)
		if err != nil {
			return err
		}

		_, err = tx.CreateTopLevelBucket(confirmHintBucket)
		return err
	})
}

// CommitSpendHint commits a spend hint for the outpoints to the cache.
func (c *HeightHintCache) CommitSpendHint(height uint32,
	spendRequests ...SpendRequest) er.R {
	if len(spendRequests) == 0 {
		return nil
	}

	log.Tracef("Updating spend hint to height %d for %v", height,
		spendRequests)

	return kvdb.Batch(c.db.Backend, func(tx kvdb.RwTx) er.R {
		spendHints := tx.ReadWriteBucket(spendHintBucket)
		if spendHints == nil {
			return ErrCorruptedHeightHintCache.Default()
		}

		var hint bytes.Buffer
		if err := channeldb.WriteElement(&hint, height); err != nil {
			return err
		}

		for _, spendRequest := range spendRequests {
			spendHintKey, err := spendRequest.SpendHintKey()
			if err != nil {
				return err
			}
			err = spendHints.Put(spendHintKey, hint.Bytes())
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// QuerySpendHint returns the latest spend hint for an outpoint.
// ErrSpendHintNotFound is returned if a spend hint does not exist within the
// cache for the outpoint.
func (c *HeightHintCache) QuerySpendHint(spendRequest SpendRequest) (uint32, er.R) {
	var hint uint32
	if c.cfg.QueryDisable {
		log.Debugf("Ignoring spend height hint for %v (height hint cache "+
			"query disabled)", spendRequest)
		return 0, nil
	}
	err := kvdb.View(c.db, func(tx kvdb.RTx) er.R {
		spendHints := tx.ReadBucket(spendHintBucket)
		if spendHints == nil {
			return ErrCorruptedHeightHintCache.Default()
		}

		spendHintKey, err := spendRequest.SpendHintKey()
		if err != nil {
			return err
		}
		spendHint := spendHints.Get(spendHintKey)
		if spendHint == nil {
			return ErrSpendHintNotFound.Default()
		}

		return channeldb.ReadElement(bytes.NewReader(spendHint), &hint)
	}, func() {
		hint = 0
	})
	if err != nil {
		return 0, err
	}

	return hint, nil
}

// PurgeSpendHint removes the spend hint for the outpoints from the cache.
func (c *HeightHintCache) PurgeSpendHint(spendRequests ...SpendRequest) er.R {
	if len(spendRequests) == 0 {
		return nil
	}

	log.Tracef("Removing spend hints for %v", spendRequests)

	return kvdb.Batch(c.db.Backend, func(tx kvdb.RwTx) er.R {
		spendHints := tx.ReadWriteBucket(spendHintBucket)
		if spendHints == nil {
			return ErrCorruptedHeightHintCache.Default()
		}

		for _, spendRequest := range spendRequests {
			spendHintKey, err := spendRequest.SpendHintKey()
			if err != nil {
				return err
			}
			if err := spendHints.Delete(spendHintKey); err != nil {
				return err
			}
		}

		return nil
	})
}

// CommitConfirmHint commits a confirm hint for the transactions to the cache.
func (c *HeightHintCache) CommitConfirmHint(height uint32,
	confRequests ...ConfRequest) er.R {
	if len(confRequests) == 0 {
		return nil
	}

	log.Tracef("Updating confirm hints to height %d for %v", height,
		confRequests)

	return kvdb.Batch(c.db.Backend, func(tx kvdb.RwTx) er.R {
		confirmHints := tx.ReadWriteBucket(confirmHintBucket)
		if confirmHints == nil {
			return ErrCorruptedHeightHintCache.Default()
		}

		var hint bytes.Buffer
		if err := channeldb.WriteElement(&hint, height); err != nil {
			return err
		}

		for _, confRequest := range confRequests {
			confHintKey, err := confRequest.ConfHintKey()
			if err != nil {
				return err
			}
			err = confirmHints.Put(confHintKey, hint.Bytes())
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// QueryConfirmHint returns the latest confirm hint for a transaction hash.
// ErrConfirmHintNotFound is returned if a confirm hint does not exist within
// the cache for the transaction hash.
func (c *HeightHintCache) QueryConfirmHint(confRequest ConfRequest) (uint32, er.R) {
	var hint uint32
	if c.cfg.QueryDisable {
		log.Debugf("Ignoring confirmation height hint for %v (height hint "+
			"cache query disabled)", confRequest)
		return 0, nil
	}
	err := kvdb.View(c.db, func(tx kvdb.RTx) er.R {
		confirmHints := tx.ReadBucket(confirmHintBucket)
		if confirmHints == nil {
			return ErrCorruptedHeightHintCache.Default()
		}

		confHintKey, err := confRequest.ConfHintKey()
		if err != nil {
			return err
		}
		confirmHint := confirmHints.Get(confHintKey)
		if confirmHint == nil {
			return ErrConfirmHintNotFound.Default()
		}

		return channeldb.ReadElement(bytes.NewReader(confirmHint), &hint)
	}, func() {
		hint = 0
	})
	if err != nil {
		return 0, err
	}

	return hint, nil
}

// PurgeConfirmHint removes the confirm hint for the transactions from the
// cache.
func (c *HeightHintCache) PurgeConfirmHint(confRequests ...ConfRequest) er.R {
	if len(confRequests) == 0 {
		return nil
	}

	log.Tracef("Removing confirm hints for %v", confRequests)

	return kvdb.Batch(c.db.Backend, func(tx kvdb.RwTx) er.R {
		confirmHints := tx.ReadWriteBucket(confirmHintBucket)
		if confirmHints == nil {
			return ErrCorruptedHeightHintCache.Default()
		}

		for _, confRequest := range confRequests {
			confHintKey, err := confRequest.ConfHintKey()
			if err != nil {
				return err
			}
			if err := confirmHints.Delete(confHintKey); err != nil {
				return err
			}
		}

		return nil
	})
}

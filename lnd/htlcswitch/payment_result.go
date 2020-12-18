package htlcswitch

import (
	"bytes"
	"encoding/binary"
	"io"
	"sync"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/lnd/channeldb"
	"github.com/pkt-cash/pktd/lnd/channeldb/kvdb"
	"github.com/pkt-cash/pktd/lnd/lnwire"
	"github.com/pkt-cash/pktd/lnd/multimutex"
	"github.com/pkt-cash/pktd/pktlog/log"
)

var (

	// networkResultStoreBucketKey is used for the root level bucket that
	// stores the network result for each payment ID.
	networkResultStoreBucketKey = []byte("network-result-store-bucket")

	// ErrPaymentIDNotFound is an error returned if the given paymentID is
	// not found.
	ErrPaymentIDNotFound = Err.CodeWithDetail("ErrPaymentIDNotFound", "paymentID not found")

	// ErrPaymentIDAlreadyExists is returned if we try to write a pending
	// payment whose paymentID already exists.
	ErrPaymentIDAlreadyExists = Err.CodeWithDetail("ErrPaymentIDAlreadyExists", "paymentID already exists")
)

// PaymentResult wraps a decoded result received from the network after a
// payment attempt was made. This is what is eventually handed to the router
// for processing.
type PaymentResult struct {
	// Preimage is set by the switch in case a sent HTLC was settled.
	Preimage [32]byte

	// Error is non-nil in case a HTLC send failed, and the HTLC is now
	// irrevocably canceled. If the payment failed during forwarding, this
	// error will be a *ForwardingError.
	Error er.R
}

// networkResult is the raw result received from the network after a payment
// attempt has been made. Since the switch doesn't always have the necessary
// data to decode the raw message, we store it together with some meta data,
// and decode it when the router query for the final result.
type networkResult struct {
	// msg is the received result. This should be of type UpdateFulfillHTLC
	// or UpdateFailHTLC.
	msg lnwire.Message

	// unencrypted indicates whether the failure encoded in the message is
	// unencrypted, and hence doesn't need to be decrypted.
	unencrypted bool

	// isResolution indicates whether this is a resolution message, in
	// which the failure reason might not be included.
	isResolution bool
}

// serializeNetworkResult serializes the networkResult.
func serializeNetworkResult(w io.Writer, n *networkResult) er.R {
	if _, err := lnwire.WriteMessage(w, n.msg, 0); err != nil {
		return err
	}

	return channeldb.WriteElements(w, n.unencrypted, n.isResolution)
}

// deserializeNetworkResult deserializes the networkResult.
func deserializeNetworkResult(r io.Reader) (*networkResult, er.R) {
	var err er.R

	n := &networkResult{}

	n.msg, err = lnwire.ReadMessage(r, 0)
	if err != nil {
		return nil, err
	}

	if err := channeldb.ReadElements(r,
		&n.unencrypted, &n.isResolution,
	); err != nil {
		return nil, err
	}

	return n, nil
}

// networkResultStore is a persistent store that stores any results of HTLCs in
// flight on the network. Since payment results are inherently asynchronous, it
// is used as a common access point for senders of HTLCs, to know when a result
// is back. The Switch will checkpoint any received result to the store, and
// the store will keep results and notify the callers about them.
type networkResultStore struct {
	db *channeldb.DB

	// results is a map from paymentIDs to channels where subscribers to
	// payment results will be notified.
	results    map[uint64][]chan *networkResult
	resultsMtx sync.Mutex

	// paymentIDMtx is a multimutex used to make sure the database and
	// result subscribers map is consistent for each payment ID in case of
	// concurrent callers.
	paymentIDMtx *multimutex.Mutex
}

func newNetworkResultStore(db *channeldb.DB) *networkResultStore {
	return &networkResultStore{
		db:           db,
		results:      make(map[uint64][]chan *networkResult),
		paymentIDMtx: multimutex.NewMutex(),
	}
}

// storeResult stores the networkResult for the given paymentID, and
// notifies any subscribers.
func (store *networkResultStore) storeResult(paymentID uint64,
	result *networkResult) er.R {
	// We get a mutex for this payment ID. This is needed to ensure
	// consistency between the database state and the subscribers in case
	// of concurrent calls.
	store.paymentIDMtx.Lock(paymentID)
	defer store.paymentIDMtx.Unlock(paymentID)

	// Serialize the payment result.
	var b bytes.Buffer
	if err := serializeNetworkResult(&b, result); err != nil {
		return err
	}

	var paymentIDBytes [8]byte
	binary.BigEndian.PutUint64(paymentIDBytes[:], paymentID)

	err := kvdb.Batch(store.db.Backend, func(tx kvdb.RwTx) er.R {
		networkResults, err := tx.CreateTopLevelBucket(
			networkResultStoreBucketKey,
		)
		if err != nil {
			return err
		}

		return networkResults.Put(paymentIDBytes[:], b.Bytes())
	})
	if err != nil {
		return err
	}

	// Now that the result is stored in the database, we can notify any
	// active subscribers.
	store.resultsMtx.Lock()
	for _, res := range store.results[paymentID] {
		res <- result
	}
	delete(store.results, paymentID)
	store.resultsMtx.Unlock()

	return nil
}

// subscribeResult is used to get the payment result for the given
// payment ID. It returns a channel on which the result will be delivered when
// ready.
func (store *networkResultStore) subscribeResult(paymentID uint64) (
	<-chan *networkResult, er.R) {
	// We get a mutex for this payment ID. This is needed to ensure
	// consistency between the database state and the subscribers in case
	// of concurrent calls.
	store.paymentIDMtx.Lock(paymentID)
	defer store.paymentIDMtx.Unlock(paymentID)

	var (
		result     *networkResult
		resultChan = make(chan *networkResult, 1)
	)

	err := kvdb.View(store.db, func(tx kvdb.RTx) er.R {
		var err er.R
		result, err = fetchResult(tx, paymentID)
		switch {

		// Result not yet available, we will notify once a result is
		// available.
		case ErrPaymentIDNotFound.Is(err):
			return nil

		case err != nil:
			return err

		// The result was found, and will be returned immediately.
		default:
			return nil
		}
	}, func() {
		result = nil
	})
	if err != nil {
		return nil, err
	}

	// If the result was found, we can send it on the result channel
	// imemdiately.
	if result != nil {
		resultChan <- result
		return resultChan, nil
	}

	// Otherwise we store the result channel for when the result is
	// available.
	store.resultsMtx.Lock()
	store.results[paymentID] = append(
		store.results[paymentID], resultChan,
	)
	store.resultsMtx.Unlock()

	return resultChan, nil
}

// getResult attempts to immediately fetch the result for the given pid from
// the store. If no result is available, ErrPaymentIDNotFound is returned.
func (store *networkResultStore) getResult(pid uint64) (
	*networkResult, er.R) {
	var result *networkResult
	err := kvdb.View(store.db, func(tx kvdb.RTx) er.R {
		var err er.R
		result, err = fetchResult(tx, pid)
		return err
	}, func() {
		result = nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func fetchResult(tx kvdb.RTx, pid uint64) (*networkResult, er.R) {
	var paymentIDBytes [8]byte
	binary.BigEndian.PutUint64(paymentIDBytes[:], pid)

	networkResults := tx.ReadBucket(networkResultStoreBucketKey)
	if networkResults == nil {
		return nil, ErrPaymentIDNotFound.Default()
	}

	// Check whether a result is already available.
	resultBytes := networkResults.Get(paymentIDBytes[:])
	if resultBytes == nil {
		return nil, ErrPaymentIDNotFound.Default()
	}

	// Decode the result we found.
	r := bytes.NewReader(resultBytes)

	return deserializeNetworkResult(r)
}

// cleanStore removes all entries from the store, except the payment IDs given.
// NOTE: Since every result not listed in the keep map will be deleted, care
// should be taken to ensure no new payment attempts are being made
// concurrently while this process is ongoing, as its result might end up being
// deleted.
func (store *networkResultStore) cleanStore(keep map[uint64]struct{}) er.R {
	return kvdb.Update(store.db.Backend, func(tx kvdb.RwTx) er.R {
		networkResults, err := tx.CreateTopLevelBucket(
			networkResultStoreBucketKey,
		)
		if err != nil {
			return err
		}

		// Iterate through the bucket, deleting all items not in the
		// keep map.
		var toClean [][]byte
		if err := networkResults.ForEach(func(k, _ []byte) er.R {
			pid := binary.BigEndian.Uint64(k)
			if _, ok := keep[pid]; ok {
				return nil
			}

			toClean = append(toClean, k)
			return nil
		}); err != nil {
			return err
		}

		for _, k := range toClean {
			err := networkResults.Delete(k)
			if err != nil {
				return err
			}
		}

		if len(toClean) > 0 {
			log.Infof("Removed %d stale entries from network "+
				"result store", len(toClean))
		}

		return nil
	}, func() {})
}

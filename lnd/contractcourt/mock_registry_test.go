package contractcourt

import (
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/lnd/channeldb"
	"github.com/pkt-cash/pktd/lnd/invoices"
	"github.com/pkt-cash/pktd/lnd/lntypes"
	"github.com/pkt-cash/pktd/lnd/lnwire"
)

type notifyExitHopData struct {
	payHash       lntypes.Hash
	paidAmount    lnwire.MilliSatoshi
	hodlChan      chan<- interface{}
	expiry        uint32
	currentHeight int32
}

type mockRegistry struct {
	notifyChan       chan notifyExitHopData
	notifyErr        *er.ErrorCode
	notifyResolution invoices.HtlcResolution
}

func (r *mockRegistry) NotifyExitHopHtlc(payHash lntypes.Hash,
	paidAmount lnwire.MilliSatoshi, expiry uint32, currentHeight int32,
	circuitKey channeldb.CircuitKey, hodlChan chan<- interface{},
	payload invoices.Payload) (invoices.HtlcResolution, er.R) {
	r.notifyChan <- notifyExitHopData{
		hodlChan:      hodlChan,
		payHash:       payHash,
		paidAmount:    paidAmount,
		expiry:        expiry,
		currentHeight: currentHeight,
	}

	var e er.R
	if r.notifyErr != nil {
		e = r.notifyErr.Default()
	}

	return r.notifyResolution, e
}

func (r *mockRegistry) HodlUnsubscribeAll(subscriber chan<- interface{}) {}

func (r *mockRegistry) LookupInvoice(lntypes.Hash) (channeldb.Invoice,
	er.R) {
	return channeldb.Invoice{}, channeldb.ErrInvoiceNotFound.Default()
}

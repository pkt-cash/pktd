package invoicesrpc

import (
	"encoding/hex"

	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/btcutil/util"
	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/lnd/channeldb"
	"github.com/pkt-cash/pktd/lnd/lnrpc"
	"github.com/pkt-cash/pktd/lnd/lnwire"
	"github.com/pkt-cash/pktd/lnd/zpay32"
)

// decodePayReq decodes the invoice payment request if present. This is needed,
// because not all information is stored in dedicated invoice fields. If there
// is no payment request present, a dummy request will be returned. This can
// happen with just-in-time inserted keysend invoices.
func decodePayReq(invoice *channeldb.Invoice,
	activeNetParams *chaincfg.Params) (*zpay32.Invoice, er.R) {
	paymentRequest := string(invoice.PaymentRequest)
	if paymentRequest == "" {
		preimage := invoice.Terms.PaymentPreimage
		if preimage == nil {
			return nil, er.New("cannot reconstruct pay req")
		}
		hash := [32]byte(preimage.Hash())
		return &zpay32.Invoice{
			PaymentHash: &hash,
		}, nil
	}

	decoded, err := zpay32.Decode(paymentRequest, activeNetParams)
	if err != nil {
		return nil, er.Errorf("unable to decode payment "+
			"request: %v", err)
	}
	return decoded, nil
}

// CreateRPCInvoice creates an *lnrpc.Invoice from the *channeldb.Invoice.
func CreateRPCInvoice(invoice *channeldb.Invoice,
	activeNetParams *chaincfg.Params) (*lnrpc.Invoice, er.R) {
	decoded, err := decodePayReq(invoice, activeNetParams)
	if err != nil {
		return nil, err
	}

	var descHash []byte
	if decoded.DescriptionHash != nil {
		descHash = decoded.DescriptionHash[:]
	}

	fallbackAddr := ""
	if decoded.FallbackAddr != nil {
		fallbackAddr = decoded.FallbackAddr.String()
	}

	settleDate := int64(0)
	if !invoice.SettleDate.IsZero() {
		settleDate = invoice.SettleDate.Unix()
	}

	// Convert between the `lnrpc` and `routing` types.
	routeHints := CreateRPCRouteHints(decoded.RouteHints)

	preimage := invoice.Terms.PaymentPreimage
	satAmt := invoice.Terms.Value.ToSatoshis()
	satAmtPaid := invoice.AmtPaid.ToSatoshis()

	isSettled := invoice.State == channeldb.ContractSettled

	var state lnrpc.Invoice_InvoiceState
	switch invoice.State {
	case channeldb.ContractOpen:
		state = lnrpc.Invoice_OPEN
	case channeldb.ContractSettled:
		state = lnrpc.Invoice_SETTLED
	case channeldb.ContractCanceled:
		state = lnrpc.Invoice_CANCELED
	case channeldb.ContractAccepted:
		state = lnrpc.Invoice_ACCEPTED
	default:
		return nil, er.Errorf("unknown invoice state %v",
			invoice.State)
	}

	rpcHtlcs := make([]*lnrpc.InvoiceHTLC, 0, len(invoice.Htlcs))
	for key, htlc := range invoice.Htlcs {
		var state lnrpc.InvoiceHTLCState
		switch htlc.State {
		case channeldb.HtlcStateAccepted:
			state = lnrpc.InvoiceHTLCState_ACCEPTED
		case channeldb.HtlcStateSettled:
			state = lnrpc.InvoiceHTLCState_SETTLED
		case channeldb.HtlcStateCanceled:
			state = lnrpc.InvoiceHTLCState_CANCELED
		default:
			return nil, er.Errorf("unknown state %v", htlc.State)
		}

		rpcHtlc := lnrpc.InvoiceHTLC{
			ChanId:          key.ChanID.ToUint64(),
			HtlcIndex:       key.HtlcID,
			AcceptHeight:    int32(htlc.AcceptHeight),
			AcceptTime:      htlc.AcceptTime.Unix(),
			ExpiryHeight:    int32(htlc.Expiry),
			AmtMsat:         uint64(htlc.Amt),
			State:           state,
			CustomRecords:   htlc.CustomRecords,
			MppTotalAmtMsat: uint64(htlc.MppTotalAmt),
		}

		// Only report resolved times if htlc is resolved.
		if htlc.State != channeldb.HtlcStateAccepted {
			rpcHtlc.ResolveTime = htlc.ResolveTime.Unix()
		}

		rpcHtlcs = append(rpcHtlcs, &rpcHtlc)
	}

	rpcInvoice := &lnrpc.Invoice{
		Memo:            string(invoice.Memo[:]),
		RHash:           decoded.PaymentHash[:],
		Value:           int64(satAmt),
		ValueMsat:       int64(invoice.Terms.Value),
		CreationDate:    invoice.CreationDate.Unix(),
		SettleDate:      settleDate,
		Settled:         isSettled,
		PaymentRequest:  string(invoice.PaymentRequest),
		DescriptionHash: descHash,
		Expiry:          int64(invoice.Terms.Expiry.Seconds()),
		CltvExpiry:      uint64(invoice.Terms.FinalCltvDelta),
		FallbackAddr:    fallbackAddr,
		RouteHints:      routeHints,
		AddIndex:        invoice.AddIndex,
		Private:         len(routeHints) > 0,
		SettleIndex:     invoice.SettleIndex,
		AmtPaidSat:      int64(satAmtPaid),
		AmtPaidMsat:     int64(invoice.AmtPaid),
		AmtPaid:         int64(invoice.AmtPaid),
		State:           state,
		Htlcs:           rpcHtlcs,
		Features:        CreateRPCFeatures(invoice.Terms.Features),
		IsKeysend:       len(invoice.PaymentRequest) == 0,
	}

	if preimage != nil {
		rpcInvoice.RPreimage = preimage[:]
	}

	return rpcInvoice, nil
}

// CreateRPCFeatures maps a feature vector into a list of lnrpc.Features.
func CreateRPCFeatures(fv *lnwire.FeatureVector) map[uint32]*lnrpc.Feature {
	if fv == nil {
		return nil
	}

	features := fv.Features()
	rpcFeatures := make(map[uint32]*lnrpc.Feature, len(features))
	for bit := range features {
		rpcFeatures[uint32(bit)] = &lnrpc.Feature{
			Name:       fv.Name(bit),
			IsRequired: bit.IsRequired(),
			IsKnown:    fv.IsKnown(bit),
		}
	}

	return rpcFeatures
}

// CreateRPCRouteHints takes in the decoded form of an invoice's route hints
// and converts them into the lnrpc type.
func CreateRPCRouteHints(routeHints [][]zpay32.HopHint) []*lnrpc.RouteHint {
	var res []*lnrpc.RouteHint

	for _, route := range routeHints {
		hopHints := make([]*lnrpc.HopHint, 0, len(route))
		for _, hop := range route {
			pubKey := hex.EncodeToString(
				hop.NodeID.SerializeCompressed(),
			)

			hint := &lnrpc.HopHint{
				NodeId:                    pubKey,
				ChanId:                    hop.ChannelID,
				FeeBaseMsat:               hop.FeeBaseMSat,
				FeeProportionalMillionths: hop.FeeProportionalMillionths,
				CltvExpiryDelta:           uint32(hop.CLTVExpiryDelta),
			}

			hopHints = append(hopHints, hint)
		}

		routeHint := &lnrpc.RouteHint{HopHints: hopHints}
		res = append(res, routeHint)
	}

	return res
}

// CreateZpay32HopHints takes in the lnrpc form of route hints and converts them
// into an invoice decoded form.
func CreateZpay32HopHints(routeHints []*lnrpc.RouteHint) ([][]zpay32.HopHint, er.R) {
	var res [][]zpay32.HopHint
	for _, route := range routeHints {
		hopHints := make([]zpay32.HopHint, 0, len(route.HopHints))
		for _, hop := range route.HopHints {
			pubKeyBytes, err := util.DecodeHex(hop.NodeId)
			if err != nil {
				return nil, err
			}
			p, err := btcec.ParsePubKey(pubKeyBytes, btcec.S256())
			if err != nil {
				return nil, err
			}
			hopHints = append(hopHints, zpay32.HopHint{
				NodeID:                    p,
				ChannelID:                 hop.ChanId,
				FeeBaseMSat:               hop.FeeBaseMsat,
				FeeProportionalMillionths: hop.FeeProportionalMillionths,
				CLTVExpiryDelta:           uint16(hop.CltvExpiryDelta),
			})
		}
		res = append(res, hopHints)
	}
	return res, nil
}

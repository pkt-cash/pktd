package migration_01_to_11

import (
	"bytes"
	"io"

	litecoinCfg "github.com/ltcsuite/ltcd/chaincfg"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/btcutil/util"
	bitcoinCfg "github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/lnd/channeldb/kvdb"
	"github.com/pkt-cash/pktd/lnd/channeldb/migration_01_to_11/zpay32"
	"github.com/pkt-cash/pktd/lnd/lnwire"
	"github.com/pkt-cash/pktd/pktlog/log"
	"github.com/pkt-cash/pktd/wire"
)

// MigrateInvoices adds invoice htlcs and a separate cltv delta field to the
// invoices.
func MigrateInvoices(tx kvdb.RwTx) er.R {
	log.Infof("Migrating invoices to new invoice format")

	invoiceB := tx.ReadWriteBucket(invoiceBucket)
	if invoiceB == nil {
		return nil
	}

	// Iterate through the entire key space of the top-level invoice bucket.
	// If key with a non-nil value stores the next invoice ID which maps to
	// the corresponding invoice. Store those keys first, because it isn't
	// safe to modify the bucket inside a ForEach loop.
	var invoiceKeys [][]byte
	err := invoiceB.ForEach(func(k, v []byte) er.R {
		if v == nil {
			return nil
		}

		invoiceKeys = append(invoiceKeys, k)

		return nil
	})
	if err != nil {
		return err
	}

	nets := []*bitcoinCfg.Params{
		&bitcoinCfg.MainNetParams, &bitcoinCfg.SimNetParams,
		&bitcoinCfg.RegressionNetParams, &bitcoinCfg.TestNet3Params,
	}

	ltcNets := []*litecoinCfg.Params{
		&litecoinCfg.MainNetParams, &litecoinCfg.SimNetParams,
		&litecoinCfg.RegressionNetParams, &litecoinCfg.TestNet4Params,
	}
	for _, net := range ltcNets {
		var convertedNet bitcoinCfg.Params
		convertedNet.Bech32HRPSegwit = net.Bech32HRPSegwit
		nets = append(nets, &convertedNet)
	}

	// Iterate over all stored keys and migrate the invoices.
	for _, k := range invoiceKeys {
		v := invoiceB.Get(k)

		// Deserialize the invoice with the deserializing function that
		// was in use for this version of the database.
		invoiceReader := bytes.NewReader(v)
		invoice, err := deserializeInvoiceLegacy(invoiceReader)
		if err != nil {
			return err
		}

		if invoice.Terms.State == ContractAccepted {
			return er.Errorf("cannot upgrade with invoice(s) " +
				"in accepted state, see release notes")
		}

		// Try to decode the payment request for every possible net to
		// avoid passing a the active network to channeldb. This would
		// be a layering violation, while this migration is only running
		// once and will likely be removed in the future.
		var payReq *zpay32.Invoice
		for _, net := range nets {
			payReq, err = zpay32.Decode(
				string(invoice.PaymentRequest), net,
			)
			if err == nil {
				break
			}
		}
		if payReq == nil {
			return er.Errorf("cannot decode payreq")
		}
		invoice.FinalCltvDelta = int32(payReq.MinFinalCLTVExpiry())
		invoice.Expiry = payReq.Expiry()

		// Serialize the invoice in the new format and use it to replace
		// the old invoice in the database.
		var buf bytes.Buffer
		if err := serializeInvoice(&buf, &invoice); err != nil {
			return err
		}

		err = invoiceB.Put(k, buf.Bytes())
		if err != nil {
			return err
		}
	}

	log.Infof("Migration of invoices completed!")
	return nil
}

func deserializeInvoiceLegacy(r io.Reader) (Invoice, er.R) {
	var err er.R
	invoice := Invoice{}

	// TODO(roasbeef): use read full everywhere
	invoice.Memo, err = wire.ReadVarBytes(r, 0, MaxMemoSize, "")
	if err != nil {
		return invoice, err
	}
	invoice.Receipt, err = wire.ReadVarBytes(r, 0, MaxReceiptSize, "")
	if err != nil {
		return invoice, err
	}

	invoice.PaymentRequest, err = wire.ReadVarBytes(r, 0, MaxPaymentRequestSize, "")
	if err != nil {
		return invoice, err
	}

	birthBytes, err := wire.ReadVarBytes(r, 0, 300, "birth")
	if err != nil {
		return invoice, err
	}
	if err := invoice.CreationDate.UnmarshalBinary(birthBytes); err != nil {
		return invoice, er.E(err)
	}

	settledBytes, err := wire.ReadVarBytes(r, 0, 300, "settled")
	if err != nil {
		return invoice, err
	}
	if err := invoice.SettleDate.UnmarshalBinary(settledBytes); err != nil {
		return invoice, er.E(err)
	}

	if _, err := util.ReadFull(r, invoice.Terms.PaymentPreimage[:]); err != nil {
		return invoice, err
	}
	var scratch [8]byte
	if _, err := util.ReadFull(r, scratch[:]); err != nil {
		return invoice, err
	}
	invoice.Terms.Value = lnwire.MilliSatoshi(byteOrder.Uint64(scratch[:]))

	if err := util.ReadBin(r, byteOrder, &invoice.Terms.State); err != nil {
		return invoice, err
	}

	if err := util.ReadBin(r, byteOrder, &invoice.AddIndex); err != nil {
		return invoice, err
	}
	if err := util.ReadBin(r, byteOrder, &invoice.SettleIndex); err != nil {
		return invoice, err
	}
	if err := util.ReadBin(r, byteOrder, &invoice.AmtPaid); err != nil {
		return invoice, err
	}

	return invoice, nil
}

// serializeInvoiceLegacy serializes an invoice in the format of the previous db
// version.
func serializeInvoiceLegacy(w io.Writer, i *Invoice) er.R {
	if err := wire.WriteVarBytes(w, 0, i.Memo); err != nil {
		return err
	}
	if err := wire.WriteVarBytes(w, 0, i.Receipt); err != nil {
		return err
	}
	if err := wire.WriteVarBytes(w, 0, i.PaymentRequest); err != nil {
		return err
	}

	birthBytes, errr := i.CreationDate.MarshalBinary()
	if errr != nil {
		return er.E(errr)
	}

	if err := wire.WriteVarBytes(w, 0, birthBytes); err != nil {
		return err
	}

	settleBytes, errr := i.SettleDate.MarshalBinary()
	if errr != nil {
		return er.E(errr)
	}

	if err := wire.WriteVarBytes(w, 0, settleBytes); err != nil {
		return err
	}

	if _, err := util.Write(w, i.Terms.PaymentPreimage[:]); err != nil {
		return err
	}

	var scratch [8]byte
	byteOrder.PutUint64(scratch[:], uint64(i.Terms.Value))
	if _, err := util.Write(w, scratch[:]); err != nil {
		return err
	}

	if err := util.WriteBin(w, byteOrder, i.Terms.State); err != nil {
		return err
	}

	if err := util.WriteBin(w, byteOrder, i.AddIndex); err != nil {
		return err
	}
	if err := util.WriteBin(w, byteOrder, i.SettleIndex); err != nil {
		return err
	}
	if err := util.WriteBin(w, byteOrder, int64(i.AmtPaid)); err != nil {
		return err
	}

	return nil
}

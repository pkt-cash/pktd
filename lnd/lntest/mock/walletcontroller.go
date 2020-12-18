package mock

import (
	"sync/atomic"
	"time"

	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/btcutil/psbt"
	"github.com/pkt-cash/pktd/btcutil/util"
	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/pktwallet/wallet/txauthor"
	"github.com/pkt-cash/pktd/pktwallet/wtxmgr"
	"github.com/pkt-cash/pktd/wire"

	"github.com/pkt-cash/pktd/lnd/lnwallet"
	"github.com/pkt-cash/pktd/lnd/lnwallet/chainfee"
)

var CoinPkScript, _ = util.DecodeHex("001431df1bde03c074d0cf21ea2529427e1499b8f1de")

// WalletController is a mock implementation of the WalletController
// interface. It let's us mock the interaction with the bitcoin network.
type WalletController struct {
	RootKey               *btcec.PrivateKey
	PublishedTransactions chan *wire.MsgTx
	index                 uint32
	Utxos                 []*lnwallet.Utxo
}

// BackEnd returns "mock" to signify a mock wallet controller.
func (w *WalletController) BackEnd() string {
	return "mock"
}

// FetchInputInfo will be called to get info about the inputs to the funding
// transaction.
func (w *WalletController) FetchInputInfo(
	prevOut *wire.OutPoint) (*lnwallet.Utxo, er.R) {
	utxo := &lnwallet.Utxo{
		AddressType:   lnwallet.WitnessPubKey,
		Value:         10 * btcutil.UnitsPerCoin(),
		PkScript:      []byte("dummy"),
		Confirmations: 1,
		OutPoint:      *prevOut,
	}
	return utxo, nil
}

// ConfirmedBalance currently returns dummy values.
func (w *WalletController) ConfirmedBalance(confs int32) (btcutil.Amount, er.R) {
	return 0, nil
}

// NewAddress is called to get new addresses for delivery, change etc.
func (w *WalletController) NewAddress(addrType lnwallet.AddressType,
	change bool) (btcutil.Address, er.R) {
	addr, _ := btcutil.NewAddressPubKey(
		w.RootKey.PubKey().SerializeCompressed(), &chaincfg.MainNetParams,
	)
	return addr, nil
}

// LastUnusedAddress currently returns dummy values.
func (w *WalletController) LastUnusedAddress(addrType lnwallet.AddressType) (
	btcutil.Address, er.R) {
	return nil, nil
}

// IsOurAddress currently returns a dummy value.
func (w *WalletController) IsOurAddress(a btcutil.Address) bool {
	return false
}

// SendOutputs currently returns dummy values.
func (w *WalletController) SendOutputs(outputs []*wire.TxOut,
	_ chainfee.SatPerKWeight, _ int32, _ string) (*wire.MsgTx, er.R) {
	return nil, nil
}

// CreateSimpleTx currently returns dummy values.
func (w *WalletController) CreateSimpleTx(outputs []*wire.TxOut,
	_ chainfee.SatPerKWeight, _ bool) (*txauthor.AuthoredTx, er.R) {
	return nil, nil
}

// ListUnspentWitness is called by the wallet when doing coin selection. We just
// need one unspent for the funding transaction.
func (w *WalletController) ListUnspentWitness(minconfirms,
	maxconfirms int32) ([]*lnwallet.Utxo, er.R) {
	// If the mock already has a list of utxos, return it.
	if w.Utxos != nil {
		return w.Utxos, nil
	}

	// Otherwise create one to return.
	utxo := &lnwallet.Utxo{
		AddressType: lnwallet.WitnessPubKey,
		Value:       btcutil.Amount(10 * btcutil.UnitsPerCoin()),
		PkScript:    CoinPkScript,
		OutPoint: wire.OutPoint{
			Hash:  chainhash.Hash{},
			Index: w.index,
		},
	}
	atomic.AddUint32(&w.index, 1)
	var ret []*lnwallet.Utxo
	ret = append(ret, utxo)
	return ret, nil
}

// ListTransactionDetails currently returns dummy values.
func (w *WalletController) ListTransactionDetails(_,
	_ int32) ([]*lnwallet.TransactionDetail, er.R) {
	return nil, nil
}

// LockOutpoint currently does nothing.
func (w *WalletController) LockOutpoint(o wire.OutPoint) {}

// UnlockOutpoint currently does nothing.
func (w *WalletController) UnlockOutpoint(o wire.OutPoint) {}

// LeaseOutput returns the current time and a nil error.
func (w *WalletController) LeaseOutput(wtxmgr.LockID, wire.OutPoint) (time.Time,
	er.R) {
	return time.Now(), nil
}

// ReleaseOutput currently does nothing.
func (w *WalletController) ReleaseOutput(wtxmgr.LockID, wire.OutPoint) er.R {
	return nil
}

// FundPsbt currently does nothing.
func (w *WalletController) FundPsbt(_ *psbt.Packet,
	_ chainfee.SatPerKWeight) (int32, er.R) {
	return 0, nil
}

// FinalizePsbt currently does nothing.
func (w *WalletController) FinalizePsbt(_ *psbt.Packet) er.R {
	return nil
}

// PublishTransaction sends a transaction to the PublishedTransactions chan.
func (w *WalletController) PublishTransaction(tx *wire.MsgTx, _ string) er.R {
	w.PublishedTransactions <- tx
	return nil
}

// LabelTransaction currently does nothing.
func (w *WalletController) LabelTransaction(_ chainhash.Hash, _ string,
	_ bool) er.R {
	return nil
}

// SubscribeTransactions currently does nothing.
func (w *WalletController) SubscribeTransactions() (lnwallet.TransactionSubscription,
	er.R) {
	return nil, nil
}

// IsSynced currently returns dummy values.
func (w *WalletController) IsSynced() (bool, int64, er.R) {
	return true, int64(0), nil
}

// GetRecoveryInfo currently returns dummy values.
func (w *WalletController) GetRecoveryInfo() (bool, float64, er.R) {
	return true, float64(1), nil
}

// Start currently does nothing.
func (w *WalletController) Start() er.R {
	return nil
}

// Stop currently does nothing.
func (w *WalletController) Stop() er.R {
	return nil
}

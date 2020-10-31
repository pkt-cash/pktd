// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"bytes"
	"sync"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/pktwallet/waddrmgr"
	"github.com/pkt-cash/pktd/pktwallet/walletdb"
	"github.com/pkt-cash/pktd/pktwallet/wtxmgr"
	"github.com/pkt-cash/pktd/txscript"
)

// TODO: It would be good to send errors during notification creation to the rpc
// server instead of just logging them here so the client is aware that wallet
// isn't working correctly and notifications are missing.

// TODO: Anything dealing with accounts here is expensive because the database
// is not organized correctly for true account support, but do the slow thing
// instead of the easy thing since the db can be fixed later, and we want the
// api correct now.

// NotificationServer is a server that interested clients may hook into to
// receive notifications of changes in a wallet.  A client is created for each
// registered notification.  Clients are guaranteed to receive messages in the
// order wallet created them, but there is no guaranteed synchronization between
// different clients.
type NotificationServer struct {
	transactions   []chan *TransactionNotifications
	currentTxNtfn  *TransactionNotifications // coalesce this since wallet does not add mined txs together
	spentness      map[uint32][]chan *SpentnessNotifications
	accountClients []chan *AccountNotification
	mu             sync.Mutex // Only protects registered client channels
	wallet         *Wallet    // smells like hacks
}

func newNotificationServer(wallet *Wallet) *NotificationServer {
	return &NotificationServer{
		spentness: make(map[uint32][]chan *SpentnessNotifications),
		wallet:    wallet,
	}
}

func lookupInputAccount(dbtx walletdb.ReadTx, w *Wallet, details *wtxmgr.TxDetails, deb wtxmgr.DebitRecord) uint32 {
	addrmgrNs := dbtx.ReadBucket(waddrmgrNamespaceKey)
	txmgrNs := dbtx.ReadBucket(wtxmgrNamespaceKey)

	// TODO: Debits should record which account(s?) they
	// debit from so this doesn't need to be looked up.
	prevOP := &details.MsgTx.TxIn[deb.Index].PreviousOutPoint
	prev, err := w.TxStore.TxDetails(txmgrNs, &prevOP.Hash)
	if err != nil {
		log.Errorf("Cannot query previous transaction details for %v: %v", prevOP.Hash, err)
		return 0
	}
	if prev == nil {
		log.Errorf("Missing previous transaction %v", prevOP.Hash)
		return 0
	}
	prevOut := prev.MsgTx.TxOut[prevOP.Index]
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(prevOut.PkScript, w.chainParams)
	var inputAcct uint32
	if err == nil && len(addrs) > 0 {
		_, inputAcct, err = w.Manager.AddrAccount(addrmgrNs, addrs[0])
	}
	if err != nil {
		log.Errorf("Cannot fetch account for previous output %v: %v", prevOP, err)
		inputAcct = 0
	}
	return inputAcct
}

func lookupOutputChain(dbtx walletdb.ReadTx, w *Wallet, details *wtxmgr.TxDetails,
	cred wtxmgr.CreditRecord) (account uint32, internal bool) {

	addrmgrNs := dbtx.ReadBucket(waddrmgrNamespaceKey)

	output := details.MsgTx.TxOut[cred.Index]
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(output.PkScript, w.chainParams)
	var ma waddrmgr.ManagedAddress
	if err == nil && len(addrs) > 0 {
		ma, err = w.Manager.Address(addrmgrNs, addrs[0])
	}
	if err != nil {
		log.Errorf("Cannot fetch account for wallet output: %v", err)
	} else {
		account = ma.Account()
		internal = ma.Internal()
	}
	return
}

func makeTxSummary(dbtx walletdb.ReadTx, w *Wallet, details *wtxmgr.TxDetails) TransactionSummary {
	serializedTx := details.SerializedTx
	if serializedTx == nil {
		var buf bytes.Buffer
		err := details.MsgTx.Serialize(&buf)
		if err != nil {
			log.Errorf("Transaction serialization: %v", err)
		}
		serializedTx = buf.Bytes()
	}
	var fee btcutil.Amount
	if len(details.Debits) == len(details.MsgTx.TxIn) {
		for _, deb := range details.Debits {
			fee += deb.Amount
		}
		for _, txOut := range details.MsgTx.TxOut {
			fee -= btcutil.Amount(txOut.Value)
		}
	}
	var inputs []TransactionSummaryInput
	if len(details.Debits) != 0 {
		inputs = make([]TransactionSummaryInput, len(details.Debits))
		for i, d := range details.Debits {
			inputs[i] = TransactionSummaryInput{
				Index:           d.Index,
				PreviousAccount: lookupInputAccount(dbtx, w, details, d),
				PreviousAmount:  d.Amount,
			}
		}
	}
	outputs := make([]TransactionSummaryOutput, 0, len(details.MsgTx.TxOut))
	for i := range details.MsgTx.TxOut {
		credIndex := len(outputs)
		mine := len(details.Credits) > credIndex && details.Credits[credIndex].Index == uint32(i)
		if !mine {
			continue
		}
		acct, internal := lookupOutputChain(dbtx, w, details, details.Credits[credIndex])
		output := TransactionSummaryOutput{
			Index:    uint32(i),
			Account:  acct,
			Internal: internal,
		}
		outputs = append(outputs, output)
	}
	return TransactionSummary{
		Hash:        &details.Hash,
		Transaction: serializedTx,
		MyInputs:    inputs,
		MyOutputs:   outputs,
		Fee:         fee,
		Timestamp:   details.Received.Unix(),
	}
}

func totalBalances(dbtx walletdb.ReadTx, w *Wallet, m map[uint32]btcutil.Amount) er.R {
	addrmgrNs := dbtx.ReadBucket(waddrmgrNamespaceKey)
	unspent, err := w.TxStore.GetUnspentOutputs(dbtx.ReadBucket(wtxmgrNamespaceKey))
	if err != nil {
		return err
	}
	for i := range unspent {
		output := &unspent[i]
		var outputAcct uint32
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(
			output.PkScript, w.chainParams)
		if err == nil && len(addrs) > 0 {
			_, outputAcct, err = w.Manager.AddrAccount(addrmgrNs, addrs[0])
		}
		if err == nil {
			_, ok := m[outputAcct]
			if ok {
				m[outputAcct] += output.Amount
			}
		}
	}
	return nil
}

func flattenBalanceMap(m map[uint32]btcutil.Amount) []AccountBalance {
	s := make([]AccountBalance, 0, len(m))
	for k, v := range m {
		s = append(s, AccountBalance{Account: k, TotalBalance: v})
	}
	return s
}

func relevantAccounts(w *Wallet, m map[uint32]btcutil.Amount, txs []TransactionSummary) {
	for _, tx := range txs {
		for _, d := range tx.MyInputs {
			m[d.PreviousAccount] = 0
		}
		for _, c := range tx.MyOutputs {
			m[c.Account] = 0
		}
	}
}

func (s *NotificationServer) notifyUnminedTransaction(dbtx walletdb.ReadTx, details *wtxmgr.TxDetails) {
	// Sanity check: should not be currently coalescing a notification for
	// mined transactions at the same time that an unmined tx is notified.
	if s.currentTxNtfn != nil {
		log.Errorf("Notifying unmined tx notification (%s) while creating notification for blocks",
			details.Hash)
	}

	defer s.mu.Unlock()
	s.mu.Lock()
	clients := s.transactions
	if len(clients) == 0 {
		return
	}

	unminedTxs := []TransactionSummary{makeTxSummary(dbtx, s.wallet, details)}
	unminedHashes, err := s.wallet.TxStore.UnminedTxHashes(dbtx.ReadBucket(wtxmgrNamespaceKey))
	if err != nil {
		log.Errorf("Cannot fetch unmined transaction hashes: %v", err)
		return
	}
	bals := make(map[uint32]btcutil.Amount)
	relevantAccounts(s.wallet, bals, unminedTxs)
	err = totalBalances(dbtx, s.wallet, bals)
	if err != nil {
		log.Errorf("Cannot determine balances for relevant accounts: %v", err)
		return
	}
	n := &TransactionNotifications{
		UnminedTransactions:      unminedTxs,
		UnminedTransactionHashes: unminedHashes,
		NewBalances:              flattenBalanceMap(bals),
	}
	for _, c := range clients {
		c <- n
	}
}

func (s *NotificationServer) notifyDetachedBlock(hash *chainhash.Hash) {
	if s.currentTxNtfn == nil {
		s.currentTxNtfn = &TransactionNotifications{}
	}
	s.currentTxNtfn.DetachedBlocks = append(s.currentTxNtfn.DetachedBlocks, hash)
}

func (s *NotificationServer) notifyMinedTransaction(dbtx walletdb.ReadTx, details *wtxmgr.TxDetails, block *wtxmgr.BlockMeta) {
	if s.currentTxNtfn == nil {
		s.currentTxNtfn = &TransactionNotifications{}
	}
	n := len(s.currentTxNtfn.AttachedBlocks)
	if n == 0 || *s.currentTxNtfn.AttachedBlocks[n-1].Hash != block.Hash {
		s.currentTxNtfn.AttachedBlocks = append(s.currentTxNtfn.AttachedBlocks, Block{
			Hash:      &block.Hash,
			Height:    block.Height,
			Timestamp: block.Time.Unix(),
		})
		n++
	}
	txs := s.currentTxNtfn.AttachedBlocks[n-1].Transactions
	s.currentTxNtfn.AttachedBlocks[n-1].Transactions =
		append(txs, makeTxSummary(dbtx, s.wallet, details))
}

// TransactionNotifications is a notification of changes to the wallet's
// transaction set and the current chain tip that wallet is considered to be
// synced with.  All transactions added to the blockchain are organized by the
// block they were mined in.
//
// During a chain switch, all removed block hashes are included.  Detached
// blocks are sorted in the reverse order they were mined.  Attached blocks are
// sorted in the order mined.
//
// All newly added unmined transactions are included.  Removed unmined
// transactions are not explicitly included.  Instead, the hashes of all
// transactions still unmined are included.
//
// If any transactions were involved, each affected account's new total balance
// is included.
//
// TODO: Because this includes stuff about blocks and can be fired without any
// changes to transactions, it needs a better name.
type TransactionNotifications struct {
	AttachedBlocks           []Block
	DetachedBlocks           []*chainhash.Hash
	UnminedTransactions      []TransactionSummary
	UnminedTransactionHashes []*chainhash.Hash
	NewBalances              []AccountBalance
}

// Block contains the properties and all relevant transactions of an attached
// block.
type Block struct {
	Hash         *chainhash.Hash
	Height       int32
	Timestamp    int64
	Transactions []TransactionSummary
}

// TransactionSummary contains a transaction relevant to the wallet and marks
// which inputs and outputs were relevant.
type TransactionSummary struct {
	Hash        *chainhash.Hash
	Transaction []byte
	MyInputs    []TransactionSummaryInput
	MyOutputs   []TransactionSummaryOutput
	Fee         btcutil.Amount
	Timestamp   int64
}

// TransactionSummaryInput describes a transaction input that is relevant to the
// wallet.  The Index field marks the transaction input index of the transaction
// (not included here).  The PreviousAccount and PreviousAmount fields describe
// how much this input debits from a wallet account.
type TransactionSummaryInput struct {
	Index           uint32
	PreviousAccount uint32
	PreviousAmount  btcutil.Amount
}

// TransactionSummaryOutput describes wallet properties of a transaction output
// controlled by the wallet.  The Index field marks the transaction output index
// of the transaction (not included here).
type TransactionSummaryOutput struct {
	Index    uint32
	Account  uint32
	Internal bool
}

// AccountBalance associates a total (zero confirmation) balance with an
// account.  Balances for other minimum confirmation counts require more
// expensive logic and it is not clear which minimums a client is interested in,
// so they are not included.
type AccountBalance struct {
	Account      uint32
	TotalBalance btcutil.Amount
}

// SpentnessNotifications is a notification that is fired for transaction
// outputs controlled by some account's keys.  The notification may be about a
// newly added unspent transaction output or that a previously unspent output is
// now spent.  When spent, the notification includes the spending transaction's
// hash and input index.
type SpentnessNotifications struct {
	hash  *chainhash.Hash
	index uint32
}

// notifyUnspentOutput notifies registered clients of a new unspent output that
// is controlled by the wallet.
func (s *NotificationServer) notifyUnspentOutput(account uint32, hash *chainhash.Hash, index uint32) {
	defer s.mu.Unlock()
	s.mu.Lock()
	clients := s.spentness[account]
	if len(clients) == 0 {
		return
	}
	n := &SpentnessNotifications{
		hash:  hash,
		index: index,
	}
	for _, c := range clients {
		c <- n
	}
}

// AccountNotification contains properties regarding an account, such as its
// name and the number of derived and imported keys.  When any of these
// properties change, the notification is fired.
type AccountNotification struct {
	AccountNumber    uint32
	AccountName      string
	ExternalKeyCount uint32
	InternalKeyCount uint32
	ImportedKeyCount uint32
}

func (s *NotificationServer) notifyAccountProperties(props *waddrmgr.AccountProperties) {
	defer s.mu.Unlock()
	s.mu.Lock()
	clients := s.accountClients
	if len(clients) == 0 {
		return
	}
	n := &AccountNotification{
		AccountNumber:    props.AccountNumber,
		AccountName:      props.AccountName,
		ExternalKeyCount: props.ExternalKeyCount,
		InternalKeyCount: props.InternalKeyCount,
		ImportedKeyCount: props.ImportedKeyCount,
	}
	for _, c := range clients {
		c <- n
	}
}

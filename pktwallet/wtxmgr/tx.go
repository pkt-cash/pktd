// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wtxmgr

import (
	"bytes"
	"time"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/pktlog"
	"github.com/pkt-cash/pktd/txscript"

	"github.com/pkt-cash/pktd/blockchain"
	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/pktwallet/walletdb"
	"github.com/pkt-cash/pktd/wire"
)

// Block contains the minimum amount of data to uniquely identify any block on
// either the best or side chain.
type Block struct {
	Hash   chainhash.Hash
	Height int32
}

// BlockMeta contains the unique identification for a block and any metadata
// pertaining to the block.  At the moment, this additional metadata only
// includes the block time from the block header.
type BlockMeta struct {
	Block
	Time time.Time
}

// blockRecord is an in-memory representation of the block record saved in the
// database.
type blockRecord struct {
	Block
	Time         time.Time
	transactions []chainhash.Hash
}

// incidence records the block hash and blockchain height of a mined transaction.
// Since a transaction hash alone is not enough to uniquely identify a mined
// transaction (duplicate transaction hashes are allowed), the incidence is used
// instead.
type incidence struct {
	txHash chainhash.Hash
	block  Block
}

// indexedIncidence records the transaction incidence and an input or output
// index.
type indexedIncidence struct {
	incidence
	index uint32
}

// credit describes a transaction output which was or is spendable by wallet.
type credit struct {
	outPoint wire.OutPoint
	block    Block
	amount   btcutil.Amount
	change   bool
	spentBy  indexedIncidence // Index == ^uint32(0) if unspent
}

// TxRecord represents a transaction managed by the Store.
type TxRecord struct {
	MsgTx        wire.MsgTx
	Hash         chainhash.Hash
	Received     time.Time
	SerializedTx []byte // Optional: may be nil
}

// NewTxRecord creates a new transaction record that may be inserted into the
// store.  It uses memoization to save the transaction hash and the serialized
// transaction.
func NewTxRecord(serializedTx []byte, received time.Time) (*TxRecord, er.R) {
	rec := &TxRecord{
		Received:     received,
		SerializedTx: serializedTx,
	}
	err := rec.MsgTx.Deserialize(bytes.NewReader(serializedTx))
	if err != nil {
		str := "failed to deserialize transaction"
		return nil, storeError(ErrInput, str, err)
	}
	copy(rec.Hash[:], chainhash.DoubleHashB(serializedTx))
	return rec, nil
}

// NewTxRecordFromMsgTx creates a new transaction record that may be inserted
// into the store.
func NewTxRecordFromMsgTx(msgTx *wire.MsgTx, received time.Time) (*TxRecord, er.R) {
	buf := bytes.NewBuffer(make([]byte, 0, msgTx.SerializeSize()))
	err := msgTx.Serialize(buf)
	if err != nil {
		str := "failed to serialize transaction"
		return nil, storeError(ErrInput, str, err)
	}
	rec := &TxRecord{
		MsgTx:        *msgTx,
		Received:     received,
		SerializedTx: buf.Bytes(),
		Hash:         msgTx.TxHash(),
	}

	return rec, nil
}

// Credit is the type representing a transaction output which was spent or
// is still spendable by wallet.  A UTXO is an unspent Credit, but not all
// Credits are UTXOs.
type Credit struct {
	wire.OutPoint
	BlockMeta
	Amount       btcutil.Amount
	PkScript     []byte
	Received     time.Time
	FromCoinBase bool
}

// Store implements a transaction store for storing and managing wallet
// transactions.
type Store struct {
	chainParams *chaincfg.Params

	// Event callbacks.  These execute in the same goroutine as the wtxmgr
	// caller.
	NotifyUnspent func(hash *chainhash.Hash, index uint32)
}

// Open opens the wallet transaction store from a walletdb namespace.  If the
// store does not exist, ErrNoExist is returned.
func Open(ns walletdb.ReadBucket, chainParams *chaincfg.Params) (*Store, er.R) {
	// Open the store.
	err := openStore(ns)
	if err != nil {
		return nil, err
	}
	s := &Store{chainParams, nil} // TODO: set callbacks
	return s, nil
}

// Create creates a new persistent transaction store in the walletdb namespace.
// Creating the store when one already exists in this namespace will error with
// ErrAlreadyExists.
func Create(ns walletdb.ReadWriteBucket) er.R {
	return createStore(ns)
}

// updateMinedBalance updates the mined balance within the store, if changed,
// after processing the given transaction record.
func (s *Store) updateMinedBalance(ns walletdb.ReadWriteBucket, rec *TxRecord,
	block *BlockMeta) er.R {

	// Fetch the mined balance in case we need to update it.
	minedBalance, err := fetchMinedBalance(ns)
	if err != nil {
		return err
	}

	// Add a debit record for each unspent credit spent by this transaction.
	// The index is set in each iteration below.
	spender := indexedIncidence{
		incidence: incidence{
			txHash: rec.Hash,
			block:  block.Block,
		},
	}

	spentByAddress := map[string]btcutil.Amount{}

	newMinedBalance := minedBalance
	for i, input := range rec.MsgTx.TxIn {
		unspentKey, credKey := existsUnspent(ns, &input.PreviousOutPoint)
		if credKey == nil {
			// Debits for unmined transactions are not explicitly
			// tracked.  Instead, all previous outputs spent by any
			// unmined transaction are added to a map for quick
			// lookups when it must be checked whether a mined
			// output is unspent or not.
			//
			// Tracking individual debits for unmined transactions
			// could be added later to simplify (and increase
			// performance of) determining some details that need
			// the previous outputs (e.g. determining a fee), but at
			// the moment that is not done (and a db lookup is used
			// for those cases instead).  There is also a good
			// chance that all unmined transaction handling will
			// move entirely to the db rather than being handled in
			// memory for atomicity reasons, so the simplist
			// implementation is currently used.
			continue
		}

		prevAddr := "unknown"
		if prevPk, err := AddressForOutPoint(ns, &input.PreviousOutPoint); err != nil {
			log.Warnf("Error decoding address spent from because [%s]", err.String())
		} else if prevPk != nil {
			prevAddr = txscript.PkScriptToAddress(prevPk, s.chainParams).String()
		}

		// If this output is relevant to us, we'll mark the it as spent
		// and remove its amount from the store.
		spender.index = uint32(i)
		amt, err := spendCredit(ns, credKey, &spender)
		if err != nil {
			return err
		}
		err = putDebit(
			ns, &rec.Hash, uint32(i), amt, &block.Block, credKey,
		)
		if err != nil {
			return err
		}
		if err := deleteRawUnspent(ns, unspentKey); err != nil {
			return err
		}
		spentByAddress[prevAddr] += amt
		newMinedBalance -= amt
	}

	for addr, amt := range spentByAddress {
		log.Infof("📩 %s [%s] from [%s] tx [%s] @ [%s]",
			pktlog.GreenBg("Confirmed spend"),
			pktlog.Coins(amt.ToBTC()),
			pktlog.Address(addr),
			pktlog.Txid(rec.Hash.String()),
			pktlog.Height(block.Height))
	}

	// For each output of the record that is marked as a credit, if the
	// output is marked as a credit by the unconfirmed store, remove the
	// marker and mark the output as a credit in the db.
	//
	// Moved credits are added as unspents, even if there is another
	// unconfirmed transaction which spends them.
	cred := credit{
		outPoint: wire.OutPoint{Hash: rec.Hash},
		block:    block.Block,
		spentBy:  indexedIncidence{index: ^uint32(0)},
	}

	it := makeUnminedCreditIterator(ns, &rec.Hash)
	for it.next() {
		// TODO: This should use the raw apis.  The credit value (it.cv)
		// can be moved from unmined directly to the credits bucket.
		// The key needs a modification to include the block
		// height/hash.
		index, err := fetchRawUnminedCreditIndex(it.ck)
		if err != nil {
			return err
		}
		amount, change, err := fetchRawUnminedCreditAmountChange(it.cv)
		if err != nil {
			return err
		}

		cred.outPoint.Index = index
		cred.amount = amount
		cred.change = change

		if err := putUnspentCredit(ns, &cred); err != nil {
			return err
		}
		err = putUnspent(ns, &cred.outPoint, &block.Block)
		if err != nil {
			return err
		}

		newMinedBalance += amount
	}
	if it.err != nil {
		return it.err
	}
	for _, input := range rec.MsgTx.TxIn {
		prevOut := input.PreviousOutPoint
		k := canonicalOutPoint(&prevOut.Hash, prevOut.Index)
		if err := deleteRawUnminedInput(ns, k, rec.Hash); err != nil {
			return err
		}
	}

	// Update the balance if it has changed.
	if newMinedBalance != minedBalance {
		return putMinedBalance(ns, newMinedBalance)
	}

	return nil
}

// deleteUnminedTx deletes an unmined transaction from the store.
//
// NOTE: This should only be used once the transaction has been mined.
func (s *Store) deleteUnminedTx(ns walletdb.ReadWriteBucket, rec *TxRecord) er.R {
	for i := range rec.MsgTx.TxOut {
		k := canonicalOutPoint(&rec.Hash, uint32(i))
		if err := deleteRawUnminedCredit(ns, k); err != nil {
			return err
		}
	}

	return deleteRawUnmined(ns, rec.Hash[:])
}

// InsertTx records a transaction as belonging to a wallet's transaction
// history.  If block is nil, the transaction is considered unspent, and the
// transaction's index must be unset.
func (s *Store) InsertTx(ns walletdb.ReadWriteBucket, rec *TxRecord, block *BlockMeta) er.R {
	if block == nil {
		return s.insertMemPoolTx(ns, rec)
	}
	return s.insertMinedTx(ns, rec, block)
}

// RemoveUnminedTx attempts to remove an unmined transaction from the
// transaction store. This is to be used in the scenario that a transaction
// that we attempt to rebroadcast, turns out to double spend one of our
// existing inputs. This function we remove the conflicting transaction
// identified by the tx record, and also recursively remove all transactions
// that depend on it.
func (s *Store) RemoveUnminedTx(ns walletdb.ReadWriteBucket, rec *TxRecord) er.R {
	// As we already have a tx record, we can directly call the
	// removeConflict method. This will do the job of recursively removing
	// this unmined transaction, and any transactions that depend on it.
	return removeConflict(ns, rec)
}

// insertMinedTx inserts a new transaction record for a mined transaction into
// the database under the confirmed bucket. It guarantees that, if the
// tranasction was previously unconfirmed, then it will take care of cleaning up
// the unconfirmed state. All other unconfirmed double spend attempts will be
// removed as well.
func (s *Store) insertMinedTx(ns walletdb.ReadWriteBucket, rec *TxRecord,
	block *BlockMeta) er.R {

	// If a transaction record for this hash and block already exists, we
	// can exit early.
	if _, v := existsTxRecord(ns, &rec.Hash, &block.Block); v != nil {
		return nil
	}

	// If a block record does not yet exist for any transactions from this
	// block, insert a block record first. Otherwise, update it by adding
	// the transaction hash to the set of transactions from this block.
	var err er.R
	blockKey, blockValue := existsBlockRecord(ns, block.Height)
	if blockValue == nil {
		err = putBlockRecord(ns, block, &rec.Hash)
	} else {
		blockValue, err = appendRawBlockRecord(blockValue, &rec.Hash)
		if err != nil {
			return err
		}
		err = putRawBlockRecord(ns, blockKey, blockValue)
	}
	if err != nil {
		return err
	}
	if err := putTxRecord(ns, rec, &block.Block); err != nil {
		return err
	}

	// Determine if this transaction has affected our balance, and if so,
	// update it.
	if err := s.updateMinedBalance(ns, rec, block); err != nil {
		return err
	}

	// If this transaction previously existed within the store as unmined,
	// we'll need to remove it from the unmined bucket.
	if v := existsRawUnmined(ns, rec.Hash[:]); v != nil {
		log.Debugf("Marking unconfirmed transaction [%s] mined in block [%d]",
			rec.Hash.String(),
			block.Height)

		if err := s.deleteUnminedTx(ns, rec); err != nil {
			return err
		}
	}

	// As there may be unconfirmed transactions that are invalidated by this
	// transaction (either being duplicates, or double spends), remove them
	// from the unconfirmed set.  This also handles removing unconfirmed
	// transaction spend chains if any other unconfirmed transactions spend
	// outputs of the removed double spend.
	return s.removeDoubleSpends(ns, rec)
}

// AddCredit marks a transaction record as containing a transaction output
// spendable by wallet.  The output is added unspent, and is marked spent
// when a new transaction spending the output is inserted into the store.
//
// TODO(jrick): This should not be necessary.  Instead, pass the indexes
// that are known to contain credits when a transaction or merkleblock is
// inserted into the store.
func (s *Store) AddCredit2(ns walletdb.ReadWriteBucket, rec *TxRecord, block *BlockMeta, index uint32, change bool) (bool, er.R) {
	if int(index) >= len(rec.MsgTx.TxOut) {
		str := "transaction output does not exist"
		return false, storeError(ErrInput, str, nil)
	}

	isNew, err := s.addCredit(ns, rec, block, index, change)
	if err == nil && isNew && s.NotifyUnspent != nil {
		s.NotifyUnspent(&rec.Hash, index)
	}
	return isNew, err
}

func (s *Store) AddCredit(ns walletdb.ReadWriteBucket, rec *TxRecord, block *BlockMeta, index uint32, change bool) er.R {
	_, err := s.AddCredit2(ns, rec, block, index, change)
	return err
}

// addCredit is an AddCredit helper that runs in an update transaction.  The
// bool return specifies whether the unspent output is newly added (true) or a
// duplicate (false).
func (s *Store) addCredit(ns walletdb.ReadWriteBucket, rec *TxRecord, block *BlockMeta, index uint32, change bool) (bool, er.R) {
	if block == nil {
		// If the outpoint that we should mark as credit already exists
		// within the store, either as unconfirmed or confirmed, then we
		// have nothing left to do and can exit.
		k := canonicalOutPoint(&rec.Hash, index)
		if existsRawUnminedCredit(ns, k) != nil {
			return false, nil
		}
		if existsRawUnspent(ns, k) != nil {
			return false, nil
		}
		v := valueUnminedCredit(btcutil.Amount(rec.MsgTx.TxOut[index].Value), change)
		return true, putRawUnminedCredit(ns, k, v)
	}

	k, v := existsCredit(ns, &rec.Hash, index, &block.Block)
	if v != nil {
		return false, nil
	}

	txOutAmt := btcutil.Amount(rec.MsgTx.TxOut[index].Value)
	log.Debugf("Marking transaction %v output %d (%v) spendable",
		rec.Hash, index, txOutAmt)

	cred := credit{
		outPoint: wire.OutPoint{
			Hash:  rec.Hash,
			Index: index,
		},
		block:   block.Block,
		amount:  txOutAmt,
		change:  change,
		spentBy: indexedIncidence{index: ^uint32(0)},
	}
	v = valueUnspentCredit(&cred)
	err := putRawCredit(ns, k, v)
	if err != nil {
		return false, err
	}

	minedBalance, err := fetchMinedBalance(ns)
	if err != nil {
		return false, err
	}
	err = putMinedBalance(ns, minedBalance+txOutAmt)
	if err != nil {
		return false, err
	}

	return true, putUnspent(ns, &cred.outPoint, &block.Block)
}

// Rollback removes all blocks at height onwards, moving any transactions within
// each block to the unconfirmed pool.
func (s *Store) Rollback(ns walletdb.ReadWriteBucket, height int32) er.R {
	return s.rollback(ns, height)
}

func rollbackTransaction(
	ns walletdb.ReadWriteBucket,
	txHash *chainhash.Hash,
	block *Block,
	params *chaincfg.Params,
) (coins btcutil.Amount, err er.R) {
	coins = btcutil.Amount(0)

	recKey := keyTxRecord(txHash, block)
	recVal := existsRawTxRecord(ns, recKey)
	var rec TxRecord
	if err = readRawTxRecord(txHash, recVal, &rec); err != nil {
		return
	}

	if err = deleteTxRecord(ns, txHash, block); err != nil {
		return
	}

	// Handle coinbase transactions specially since they are
	// not moved to the unconfirmed store.  A coinbase cannot
	// contain any debits, but all credits should be removed
	// and the mined balance decremented.
	if blockchain.IsCoinBaseTx(&rec.MsgTx) {
		op := wire.OutPoint{Hash: rec.Hash}
		for i, output := range rec.MsgTx.TxOut {
			k, v := existsCredit(ns, &rec.Hash,
				uint32(i), block)
			if v == nil {
				continue
			}
			op.Index = uint32(i)

			addr := txscript.PkScriptToAddress(output.PkScript, params)
			log.Infof("😱 %s [%s] <- [%s] by rollback of [%s]",
				pktlog.BgYellow("Got UNPAID"),
				pktlog.Coins(btcutil.Amount(output.Value).ToBTC()),
				pktlog.Address(addr.String()),
				pktlog.Txid(rec.Hash.String()))

			// Delete the unspents from this coinbase
			unspentKey, credKey := existsUnspent(ns, &op)
			if credKey != nil {
				coins -= btcutil.Amount(output.Value)
				if err = deleteRawUnspent(ns, unspentKey); err != nil {
					return
				}
			}

			// Delete any credits, spent or otherwise
			if err = deleteRawCredit(ns, k); err != nil {
				return
			}

			// If there are any mined transactions which spent this one, well
			// then lets just assume that pktd does its job and those blocks are
			// going to be properly rolled back. But if there *unmined* transactions
			// in the mempool which spent these outputs, then we'd better clear them
			// out because otherwise we will just keep broadcasting them forever.
			opKey := canonicalOutPoint(&op.Hash, op.Index)
			unminedSpendTxHashKeys := fetchUnminedInputSpendTxHashes(ns, opKey)
			for _, unminedSpendTxHashKey := range unminedSpendTxHashKeys {
				unminedVal := existsRawUnmined(ns, unminedSpendTxHashKey[:])

				// If the spending transaction spends multiple outputs
				// from the same transaction, we'll find duplicate
				// entries within the store, so it's possible we're
				// unable to find it if the conflicts have already been
				// removed in a previous iteration.
				if unminedVal == nil {
					continue
				}

				var unminedRec TxRecord
				unminedRec.Hash = unminedSpendTxHashKey
				err = readRawTxRecord(&unminedRec.Hash, unminedVal, &unminedRec)
				if err != nil {
					return
				}

				log.Debugf("Transaction %v spends a removed coinbase "+
					"output -- removing as well", unminedRec.Hash)
				err = removeConflict(ns, &unminedRec)
				if err != nil {
					return
				}
			}
		}

		return
	}

	if err = putRawUnmined(ns, txHash[:], recVal); err != nil {
		return
	}

	// For each debit recorded for this transaction, mark
	// the credit it spends as unspent (as long as it still
	// exists) and delete the debit.  The previous output is
	// recorded in the unconfirmed store for every previous
	// output, not just debits.
	unspentByAddress := map[string]btcutil.Amount{}
	for i, input := range rec.MsgTx.TxIn {
		prevOut := &input.PreviousOutPoint
		prevOutKey := canonicalOutPoint(&prevOut.Hash,
			prevOut.Index)
		if err = putRawUnminedInput(ns, prevOutKey, rec.Hash[:]); err != nil {
			return
		}

		// If this input is a debit, remove the debit
		// record and mark the credit that it spent as
		// unspent, incrementing the mined balance.
		var debKey, credKey []byte
		if debKey, credKey, err = existsDebit(ns, &rec.Hash, uint32(i), block); err != nil {
			return
		}
		if debKey == nil {
			continue
		}

		// unspendRawCredit does not error in case the
		// no credit exists for this key, but this
		// behavior is correct.  Since blocks are
		// removed in increasing order, this credit
		// may have already been removed from a
		// previously removed transaction record in
		// this rollback.
		var amt btcutil.Amount
		if amt, err = unspendRawCredit(ns, credKey); err != nil {
			return
		}
		if err = deleteRawDebit(ns, debKey); err != nil {
			return
		}

		// If the credit was previously removed in the
		// rollback, the credit amount is zero.  Only
		// mark the previously spent credit as unspent
		// if it still exists.
		if amt == 0 {
			continue
		}
		var unspentVal []byte
		if unspentVal, err = fetchRawCreditUnspentValue(credKey); err != nil {
			return
		}
		coins += amt
		if err = putRawUnspent(ns, prevOutKey, unspentVal); err != nil {
			return
		}

		prevAddr := "unknown"
		if prevPk, err := AddressForOutPoint(ns, &input.PreviousOutPoint); err != nil {
			log.Warnf("Error decoding address spent from because [%s]", err.String())
		} else if prevPk != nil {
			prevAddr = txscript.PkScriptToAddress(prevPk, params).String()
		}
		unspentByAddress[prevAddr] += amt
	}

	for addr, amt := range unspentByAddress {
		log.Infof("⚠️ Spend unconfirmed [%s] <- [%s] by rollback of [%s]",
			pktlog.Coins(btcutil.Amount(amt).ToBTC()),
			pktlog.Address(addr),
			pktlog.Txid(rec.Hash.String()))
	}

	// For each detached non-coinbase credit, move the
	// credit output to unmined.  If the credit is marked
	// unspent, it is removed from the utxo set and the
	// mined balance is decremented.
	//
	// TODO: use a credit iterator
	unearnedByAddress := map[string]btcutil.Amount{}
	for i, output := range rec.MsgTx.TxOut {
		k, v := existsCredit(ns, &rec.Hash, uint32(i), block)
		if v == nil {
			continue
		}

		var amt btcutil.Amount
		var change bool
		if amt, change, err = fetchRawCreditAmountChange(v); err != nil {
			return
		}
		outPointKey := canonicalOutPoint(&rec.Hash, uint32(i))
		unminedCredVal := valueUnminedCredit(amt, change)
		if err = putRawUnminedCredit(ns, outPointKey, unminedCredVal); err != nil {
			return
		}

		if err = deleteRawCredit(ns, k); err != nil {
			return
		}

		credKey := existsRawUnspent(ns, outPointKey)
		if credKey != nil {
			coins -= btcutil.Amount(output.Value)
			if err = deleteRawUnspent(ns, outPointKey); err != nil {
				return
			}
			prevAddr := txscript.PkScriptToAddress(output.PkScript, params).String()
			unearnedByAddress[prevAddr] += btcutil.Amount(output.Value)
		}
	}
	for addr, amt := range unearnedByAddress {
		log.Infof("⚠️ Income unconfirmed [%s] <- [%s] by rollback of [%s]",
			pktlog.Coins(btcutil.Amount(amt).ToBTC()),
			pktlog.Address(addr),
			pktlog.Txid(rec.Hash.String()))
	}
	return
}

// RollbackTransaction kills off a transaction in case it is invalid or burned
// or the block is orphaned. The results are:
// coins: the amount to *add* to the balance
// err: any error which might occur
func RollbackTransaction(
	ns walletdb.ReadWriteBucket,
	txHash *chainhash.Hash,
	block *Block,
	params *chaincfg.Params,
) er.R {
	if minedBalance, err := fetchMinedBalance(ns); err != nil {
		return err
	} else if coins, err := rollbackTransaction(ns, txHash, block, params); err != nil {
		return err
	} else if err := putMinedBalance(ns, minedBalance+coins); err != nil {
		return err
	}
	return nil
}

func (s *Store) rollback(ns walletdb.ReadWriteBucket, height int32) er.R {
	minedBalance, err := fetchMinedBalance(ns)
	if err != nil {
		return err
	}

	var heightsToRemove []int32

	it := makeReverseBlockIterator(ns)
	for it.prev() {
		b := &it.elem
		if it.elem.Height < height {
			break
		}

		heightsToRemove = append(heightsToRemove, it.elem.Height)

		log.Warnf("ROLLBACK rolling back %d transactions from block %v height %d",
			len(b.transactions), b.Hash, b.Height)

		for i := range b.transactions {
			txHash := &b.transactions[i]
			log.Infof("Rolling back tx [%s]", pktlog.Txid(txHash.String()))
			coins, err := rollbackTransaction(ns, txHash, &b.Block, s.chainParams)
			if err != nil {
				return err
			}
			minedBalance += coins
		}

		// reposition cursor before deleting this k/v pair and advancing to the
		// previous.
		it.reposition(it.elem.Height)

		// Avoid cursor deletion until bolt issue #620 is resolved.
		// err = it.delete()
		// if err != nil {
		// 	return err
		// }
	}
	if it.err != nil {
		return it.err
	}

	// Delete the block records outside of the iteration since cursor deletion
	// is broken.
	for _, h := range heightsToRemove {
		err = deleteBlockRecord(ns, h)
		if err != nil {
			return err
		}
	}

	return putMinedBalance(ns, minedBalance)
}

// ForEachUnspentOutput runs the visitor over each unspent output (in undefined order)
// Any error type other than wtxmgr.Err or er.LoopBreak will be wrapped as wtxmgr.ErrDatabase
//
// beginKey can be used to skip entries for pagination, however beware there is no guarantee
// of never receiving duplicate entries in different pages. In particular, all unconfirmed
// outputs will be received after the final confirmed output with no regard for what you specify
// as the beginKey.
func (s *Store) ForEachUnspentOutput(
	ns walletdb.ReadBucket,
	beginKey []byte,
	visitor func(key []byte, c *Credit) er.R,
) er.R {
	var op wire.OutPoint
	var block Block
	bu := ns.NestedReadBucket(bucketUnspent)
	var lastKey []byte
	if err := bu.ForEachBeginningWith(beginKey, func(k, v []byte) er.R {
		lastKey = k
		err := readCanonicalOutPoint(k, &op)
		if err != nil {
			return err
		}
		if existsRawUnminedInput(ns, k) != nil {
			// Output is spent by an unmined transaction.
			// Skip this k/v pair.
			return nil
		}
		err = readUnspentBlock(v, &block)
		if err != nil {
			return err
		}

		blockTime, err := fetchBlockTime(ns, block.Height)
		if err != nil {
			return err
		}
		// TODO(jrick): reading the entire transaction should
		// be avoidable.  Creating the credit only requires the
		// output amount and pkScript.
		rec, err := fetchTxRecord(ns, &op.Hash, &block)
		if err != nil {
			return err
		} else if rec == nil {
			return er.New("fetchTxRecord() -> nil")
		}
		txOut := rec.MsgTx.TxOut[op.Index]
		cred := Credit{
			OutPoint: op,
			BlockMeta: BlockMeta{
				Block: block,
				Time:  blockTime,
			},
			Amount:       btcutil.Amount(txOut.Value),
			PkScript:     txOut.PkScript,
			Received:     rec.Received,
			FromCoinBase: blockchain.IsCoinBaseTx(&rec.MsgTx),
		}
		return visitor(k, &cred)
	}); err != nil {
		if er.IsLoopBreak(err) || Err.Is(err) {
			return err
		}
		return storeError(ErrDatabase, "failed iterating unspent bucket", err)
	}

	// There's no easy way to do ForEachBeginningWith because these entries
	// will appear out of order with the main unspents, but the amount of unconfirmed
	// credits will tend to be small anyway so we might as well just send them all
	// if the iterator gets to this stage.
	if err := ns.NestedReadBucket(bucketUnminedCredits).ForEach(func(k, v []byte) er.R {
		if existsRawUnminedInput(ns, k) != nil {
			// Output is spent by an unmined transaction.
			// Skip to next unmined credit.
			return nil
		}

		err := readCanonicalOutPoint(k, &op)
		if err != nil {
			return err
		}

		// TODO(jrick): Reading/parsing the entire transaction record
		// just for the output amount and script can be avoided.
		recVal := existsRawUnmined(ns, op.Hash[:])
		var rec TxRecord
		err = readRawTxRecord(&op.Hash, recVal, &rec)
		if err != nil {
			return err
		}

		txOut := rec.MsgTx.TxOut[op.Index]
		cred := Credit{
			OutPoint: op,
			BlockMeta: BlockMeta{
				Block: Block{Height: -1},
			},
			Amount:       btcutil.Amount(txOut.Value),
			PkScript:     txOut.PkScript,
			Received:     rec.Received,
			FromCoinBase: blockchain.IsCoinBaseTx(&rec.MsgTx),
		}
		// Use the final key to come from the main search loop so that further calls
		// will arrive here as quickly as possible.
		return visitor(lastKey, &cred)
	}); err != nil {
		if er.IsLoopBreak(err) || Err.Is(err) {
			return err
		}
		return storeError(ErrDatabase, "failed iterating unmined credits bucket", err)
	}

	return nil
}

// GetUnspentOutputs returns all unspent received transaction outputs.
// The order is undefined.
func (s *Store) GetUnspentOutputs(ns walletdb.ReadBucket) ([]Credit, er.R) {
	var unspent []Credit
	err := s.ForEachUnspentOutput(ns, nil, func(_ []byte, c *Credit) er.R {
		unspent = append(unspent, *c)
		return nil
	})
	return unspent, err
}

// Balance returns the spendable wallet balance (total value of all unspent
// transaction outputs) given a minimum of minConf confirmations, calculated
// at a current chain height of curHeight.  Coinbase outputs are only included
// in the balance if maturity has been reached.
//
// Balance may return unexpected results if syncHeight is lower than the block
// height of the most recent mined transaction in the store.
func (s *Store) Balance(ns walletdb.ReadBucket, minConf int32, syncHeight int32) (btcutil.Amount, er.R) {
	bal, err := fetchMinedBalance(ns)
	if err != nil {
		return 0, err
	}

	// Subtract the balance for each credit that is spent by an unmined
	// transaction.
	var op wire.OutPoint
	var block Block
	err = ns.NestedReadBucket(bucketUnspent).ForEach(func(k, v []byte) er.R {
		err := readCanonicalOutPoint(k, &op)
		if err != nil {
			return err
		}
		err = readUnspentBlock(v, &block)
		if err != nil {
			return err
		}
		if existsRawUnminedInput(ns, k) != nil {
			_, v := existsCredit(ns, &op.Hash, op.Index, &block)
			amt, err := fetchRawCreditAmount(v)
			if err != nil {
				return err
			}
			bal -= amt
		}
		return nil
	})
	if err != nil {
		if Err.Is(err) {
			return 0, err
		}
		str := "failed iterating unspent outputs"
		return 0, storeError(ErrDatabase, str, err)
	}

	// Decrement the balance for any unspent credit with less than
	// minConf confirmations and any (unspent) immature coinbase credit.
	coinbaseMaturity := int32(s.chainParams.CoinbaseMaturity)
	stopConf := minConf
	if coinbaseMaturity > stopConf {
		stopConf = coinbaseMaturity
	}
	lastHeight := syncHeight - stopConf
	blockIt := makeReadReverseBlockIterator(ns)
	for blockIt.prev() {
		block := &blockIt.elem

		if block.Height < lastHeight {
			break
		}

		for i := range block.transactions {
			txHash := &block.transactions[i]
			rec, err := fetchTxRecord(ns, txHash, &block.Block)
			if err != nil {
				return 0, err
			} else if rec == nil {
				// Stray block.transactions entry for transaction which
				// should have been deleted.
				continue
			}
			numOuts := uint32(len(rec.MsgTx.TxOut))
			for i := uint32(0); i < numOuts; i++ {
				// Avoid double decrementing the credit amount
				// if it was already removed for being spent by
				// an unmined tx.
				opKey := canonicalOutPoint(txHash, i)
				if existsRawUnminedInput(ns, opKey) != nil {
					continue
				}

				_, v := existsCredit(ns, txHash, i, &block.Block)
				if v == nil {
					continue
				}
				amt, spent, err := fetchRawCreditAmountSpent(v)
				if err != nil {
					return 0, err
				}
				if spent {
					continue
				}
				confs := syncHeight - block.Height + 1
				if confs < minConf || (blockchain.IsCoinBaseTx(&rec.MsgTx) &&
					confs < coinbaseMaturity) {
					bal -= amt
				}
			}
		}
	}
	if blockIt.err != nil {
		return 0, blockIt.err
	}

	// If unmined outputs are included, increment the balance for each
	// output that is unspent.
	if minConf == 0 {
		err = ns.NestedReadBucket(bucketUnminedCredits).ForEach(func(k, v []byte) er.R {
			if existsRawUnminedInput(ns, k) != nil {
				// Output is spent by an unmined transaction.
				// Skip to next unmined credit.
				return nil
			}

			amount, err := fetchRawUnminedCreditAmount(v)
			if err != nil {
				return err
			}
			bal += amount
			return nil
		})
		if err != nil {
			if Err.Is(err) {
				return 0, err
			}
			str := "failed to iterate over unmined credits bucket"
			return 0, storeError(ErrDatabase, str, err)
		}
	}

	return bal, nil
}

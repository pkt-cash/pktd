// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"encoding/hex"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/database"
	"github.com/pkt-cash/pktd/txscript"
)

func electionIsVoteAgainst(pkScript, networkSteward []byte) bool {
	_, va := txscript.ElectionGetVotesForAgainst(pkScript)
	return va != nil && bytes.Equal(va, networkSteward)
}

// ElectionState ...
type ElectionState struct {
	NetworkSteward []byte
	Disapproval    int64
}

// ElectionCandidate ...
type electionCandidate struct {
	approval    int64
	disapproval int64
	key         []byte
}
type election map[[80]byte]*electionCandidate

func (e *election) castBallot(pkScript []byte, value int64) {
	vfor, vagainst := txscript.ElectionGetVotesForAgainst(pkScript)
	if vfor != nil {
		var arr [80]byte
		copy(arr[:], vfor)
		x := (*e)[arr]
		if x == nil {
			x = &electionCandidate{key: vfor}
		}
		x.approval += value
		(*e)[arr] = x
	}

	if vagainst != nil {
		var arr [80]byte
		copy(arr[:], vagainst)
		x := (*e)[arr]
		if x == nil {
			x = &electionCandidate{key: vagainst}
		}
		x.disapproval += value
		(*e)[arr] = x
	}
}

// electionProcessBlock computes a new ElectionState based on the state at the
// tip of the best chain and the UtxoViewpoint, which is essentially a diff against
// that tip. Note that this can be called for a block which is not building on the
// chain tip, but the UtxoViewpoint should show as "spent" the transactions which
// were non-existant at the time that this is being considered against.
//
// First we try the easy way, take the winner and disapproval rating at the
// current tip and then add/subtract the transactions which were created/spent
// in the UtxoViewpoint to get the new disapproval rating. If the disapproval
// rating runs over 50% of all money, then we need to perform a full election which
// means finding the candidate which has the highest approval rating based on the
// utxo set in the database, then applying the difference of the UtxoViewpoint to
// that in order to make sure that it's valid for the chain state in question.
//
// This function is safe for concurrent access
func (b *BlockChain) electionProcessBlock(view *UtxoViewpoint, blockHeight int32) (*ElectionState, er.R) {
	// first easy
	b.stateLock.RLock()
	tipState := b.stateSnapshot.Elect
	hash := b.stateSnapshot.Hash
	b.stateLock.RUnlock()
	disapproval := tipState.Disapproval
	log.Tracef("electionProcessBlock(%v)", hex.EncodeToString(hash[:]))
	for _, e := range view.Entries() {
		if e == nil || !e.isModified() {
			continue
		}
		if electionIsVoteAgainst(e.PkScript(), tipState.NetworkSteward) {
			if e.IsSpent() {
				disapproval -= e.Amount()
			} else {
				disapproval += e.Amount()
			}
		}
	}
	log.Tracef("electionProcessBlock changed disapproval rating by [%v]", (disapproval - tipState.Disapproval))
	if len(tipState.NetworkSteward) > 0 && disapproval <= PktCalcTotalMoney(blockHeight)/2 {
		log.Tracef("electionProcessBlock no election required")
		es := ElectionState{
			NetworkSteward: tipState.NetworkSteward,
			Disapproval:    disapproval,
		}
		return &es, nil
	}
	log.Tracef("electionProcessBlock election required")
	// Ok, that didn't work, we need to have a full election
	// go to the database and walk the entire utxo set, then come back and update
	// the results based on the utxo viewpoint
	elect := make(election)
	err := b.db.View(func(dbTx database.Tx) er.R {
		utxoBucket := dbTx.Metadata().Bucket(utxoSetBucketName)
		return utxoBucket.ForEach(func(outPt, utxoBytes []byte) er.R {
			utxo, err := deserializeUtxoEntry(utxoBytes)
			if err != nil {
				return err
			}
			elect.castBallot(utxo.PkScript(), utxo.Amount())
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	for _, e := range view.Entries() {
		if e == nil || !e.isModified() {
			continue
		}
		amount := e.Amount()
		if e.IsSpent() {
			amount = -amount
		}
		elect.castBallot(e.PkScript(), amount)
	}
	winner := tipState.NetworkSteward
	approval := int64(0)
	log.Tracef("electionProcessBlock election has [%d] candidates", len(elect))
	for _, cand := range elect {
		log.Tracef("electionProcessBlock candidate [%s] has [%d] approval and [%d] disapproval",
			hex.EncodeToString(cand.key), cand.approval, cand.disapproval)
		if cand.approval > approval {
			winner = cand.key
			approval = cand.approval
			disapproval = cand.disapproval
		}
	}
	log.Tracef("electionProcessBlock winner is [%s] with [%d] approval and [%d] disapproval",
		hex.EncodeToString(winner), approval, disapproval)
	result := ElectionState{
		NetworkSteward: winner,
		Disapproval:    disapproval,
	}
	return &result, nil
}

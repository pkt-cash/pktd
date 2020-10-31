// Copyright (c) 2014 The btcsuite developers
// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcjson

import (
	"time"

	"github.com/pkt-cash/pktd/peer"
)

// GetTransactionDetailsResult models the details data from the gettransaction command.
//
// This models the "short" version of the ListTransactionsResult type, which
// excludes fields common to the transaction.  These common fields are instead
// part of the GetTransactionResult.
type GetTransactionDetailsResult struct {
	Account           string   `json:"account"`
	Address           string   `json:"address,omitempty"`
	Amount            float64  `json:"amount"`
	Category          string   `json:"category"`
	InvolvesWatchOnly bool     `json:"involveswatchonly,omitempty"`
	Fee               *float64 `json:"fee,omitempty"`
	Vout              uint32   `json:"vout"`
}

// GetTransactionResult models the data from the gettransaction command.
type GetTransactionResult struct {
	Amount          float64                       `json:"amount"`
	Fee             float64                       `json:"fee,omitempty"`
	Confirmations   int64                         `json:"confirmations"`
	BlockHash       string                        `json:"blockhash"`
	BlockIndex      int64                         `json:"blockindex"`
	BlockTime       int64                         `json:"blocktime"`
	TxID            string                        `json:"txid"`
	WalletConflicts []string                      `json:"walletconflicts"`
	Time            int64                         `json:"time"`
	TimeReceived    int64                         `json:"timereceived"`
	Details         []GetTransactionDetailsResult `json:"details"`
	Hex             string                        `json:"hex"`
}

// InfoWalletResult models the data returned by the wallet server getinfo
// command.
type InfoWalletResult struct {
	Version         int32   `json:"version"`
	ProtocolVersion int32   `json:"protocolversion"`
	WalletVersion   int32   `json:"walletversion"`
	Balance         float64 `json:"balance"`
	Blocks          int32   `json:"blocks"`
	TimeOffset      int64   `json:"timeoffset"`
	Connections     int32   `json:"connections"`
	Difficulty      float64 `json:"difficulty"`
	TestNet         bool    `json:"testnet"`
	KeypoolOldest   int64   `json:"keypoololdest"`
	KeypoolSize     int32   `json:"keypoolsize"`
	UnlockedUntil   int64   `json:"unlocked_until"`
	PaytxFee        float64 `json:"paytxfee"`
	RelayFee        float64 `json:"relayfee"`
	Errors          string  `json:"errors"`
}

type NeutrinoBan struct {
	Addr    string
	Reason  string
	EndTime string `json:"endtime"`
}

type NeutrinoQuery struct {
	Peer             string
	Command          string
	ReqNum           uint32
	CreateTime       uint32
	LastRequestTime  uint32
	LastResponseTime uint32
}

type NeutrinoInfo struct {
	Peers   []peer.PeerDesc
	Bans    []NeutrinoBan
	Queries []NeutrinoQuery
}

type WalletStats struct {
	MaintenanceInProgress       bool
	MaintenanceName             string
	MaintenanceCycles           int
	MaintenanceLastBlockVisited int
	TimeOfLastMaintenance       time.Time

	// If we're currently in a resync
	Syncing              bool
	SyncStarted          *time.Time
	SyncRemainingSeconds int64
	SyncCurrentBlock     int32
	SyncFrom             int32
	SyncTo               int32

	// General info
	BirthdayBlock int32
}

type WalletInfoResult struct {
	CurrentBlockHash      string
	CurrentHeight         int32
	CurrentBlockTimestamp time.Time
	BackendBlockHash      string
	BackendHeight         int32
	BackendBlockTimestamp time.Time
	IsSyncing             bool
	Backend               string
	WalletVersion         int32
	WalletStats           *WalletStats

	RPCInfo      *InfoWalletResult
	NeutrinoInfo *NeutrinoInfo
}

// ListTransactionsResult models the data from the listtransactions command.
type ListTransactionsResult struct {
	Abandoned         bool     `json:"abandoned"`
	Account           string   `json:"account"`
	Address           string   `json:"address,omitempty"`
	Amount            float64  `json:"amount"`
	BIP125Replaceable string   `json:"bip125-replaceable,omitempty"`
	BlockHash         string   `json:"blockhash,omitempty"`
	BlockIndex        *int64   `json:"blockindex,omitempty"`
	BlockTime         int64    `json:"blocktime,omitempty"`
	Category          string   `json:"category"`
	Confirmations     int64    `json:"confirmations"`
	Fee               *float64 `json:"fee,omitempty"`
	Generated         bool     `json:"generated,omitempty"`
	InvolvesWatchOnly bool     `json:"involveswatchonly,omitempty"`
	Time              int64    `json:"time"`
	TimeReceived      int64    `json:"timereceived"`
	Trusted           bool     `json:"trusted"`
	TxID              string   `json:"txid"`
	Vout              uint32   `json:"vout"`
	WalletConflicts   []string `json:"walletconflicts"`
	Comment           string   `json:"comment,omitempty"`
	OtherAccount      string   `json:"otheraccount,omitempty"`
}

// ListReceivedByAddressResult models the data from the listreceivedbyaddress
// command.
type ListReceivedByAddressResult struct {
	Account           string   `json:"account"`
	Address           string   `json:"address"`
	Amount            float64  `json:"amount"`
	Confirmations     uint64   `json:"confirmations"`
	TxIDs             []string `json:"txids,omitempty"`
	InvolvesWatchonly bool     `json:"involvesWatchonly,omitempty"`
}

// ListSinceBlockResult models the data from the listsinceblock command.
type ListSinceBlockResult struct {
	Transactions []ListTransactionsResult `json:"transactions"`
	LastBlock    string                   `json:"lastblock"`
}

// ListUnspentResult models a successful response from the listunspent request.
type ListUnspentResult struct {
	TxID          string  `json:"txid"`
	Vout          uint32  `json:"vout"`
	Address       string  `json:"address"`
	Account       string  `json:"account"`
	ScriptPubKey  string  `json:"scriptPubKey"`
	RedeemScript  string  `json:"redeemScript,omitempty"`
	Amount        float64 `json:"amount"`
	Confirmations int64   `json:"confirmations"`
	Height        int64   `json:"height"`
	BlockHash     string  `json:"blockHash"`
	Spendable     bool    `json:"spendable"`
}

// SignRawTransactionError models the data that contains script verification
// errors from the signrawtransaction request.
type SignRawTransactionError struct {
	TxID      string `json:"txid"`
	Vout      uint32 `json:"vout"`
	ScriptSig string `json:"scriptSig"`
	Sequence  uint32 `json:"sequence"`
	Error     string `json:"error"`
}

// SignRawTransactionResult models the data from the signrawtransaction
// command.
type SignRawTransactionResult struct {
	Hex      string                    `json:"hex"`
	Complete bool                      `json:"complete"`
	Errors   []SignRawTransactionError `json:"errors,omitempty"`
}

// ValidateAddressWalletResult models the data returned by the wallet server
// validateaddress command.
type ValidateAddressWalletResult struct {
	IsValid      bool     `json:"isvalid"`
	Address      string   `json:"address,omitempty"`
	IsMine       bool     `json:"ismine,omitempty"`
	IsWatchOnly  bool     `json:"iswatchonly,omitempty"`
	IsScript     bool     `json:"isscript,omitempty"`
	PubKey       string   `json:"pubkey,omitempty"`
	IsCompressed bool     `json:"iscompressed,omitempty"`
	Account      string   `json:"account,omitempty"`
	Addresses    []string `json:"addresses,omitempty"`
	Hex          string   `json:"hex,omitempty"`
	Script       string   `json:"script,omitempty"`
	SigsRequired int32    `json:"sigsrequired,omitempty"`
}

// GetBestBlockResult models the data from the getbestblock command.
type GetBestBlockResult struct {
	Hash   string `json:"hash"`
	Height int32  `json:"height"`
}

// SetNetworkStewardVoteResult is the result of the wallet command setnetworkstewardvote
type SetNetworkStewardVoteResult struct{}

// GetNetworkStewardVoteResult is the result of the wallet command getnetworkstewardvote
type GetNetworkStewardVoteResult struct {
	VoteFor     string `json:"votefor,omitempty"`
	VoteAgainst string `json:"voteagainst,omitempty"`
}

type GetAddressBalancesResult struct {
	Address string `json:"address"`

	Total  float64 `json:"total"`
	Stotal string  `json:"stotal"`

	Spendable  float64 `json:"spendable"`
	Sspendable string  `json:"sspendable"`

	ImmatureReward  float64 `json:"immaturereward"`
	SimmatureReward string  `json:"simmaturereward"`

	Unconfirmed  float64 `json:"unconfirmed"`
	Sunconfirmed string  `json:"sunconfirmed"`

	OutputCount int32 `json:"outputcount"`
}

type MaintenanceStats struct {
	// Burned           int
	// Orphaned         int
	// VisitedUtxos     int
	LastBlockVisited int
	Done             bool
	Name             string
}

type WalletMempoolItem struct {
	Txid     string
	Received string
}
type WalletMempoolRes []WalletMempoolItem

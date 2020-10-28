package chain

import (
	"time"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/pktwallet/waddrmgr"

	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/btcutil/gcs"
	"github.com/pkt-cash/pktd/btcutil/gcs/builder"
	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/neutrino"
	"github.com/pkt-cash/pktd/txscript"
	"github.com/pkt-cash/pktd/wire"
)

// NeutrinoClient is an implementation of the btcwalet chain.Interface interface.
type NeutrinoClient struct {
	CS          *neutrino.ChainService
	stop        chan struct{}
	chainParams *chaincfg.Params
}

// NewNeutrinoClient creates a new NeutrinoClient struct with a backing
// ChainService.
func NewNeutrinoClient(chainParams *chaincfg.Params,
	chainService *neutrino.ChainService) *NeutrinoClient {

	return &NeutrinoClient{
		CS:          chainService,
		stop:        make(chan struct{}),
		chainParams: chainParams,
	}
}

// BackEnd returns the name of the driver.
func (s *NeutrinoClient) BackEnd() string {
	return "neutrino"
}

// Start replicates the RPC client's Start method.
func (s *NeutrinoClient) Start() er.R {
	s.CS.Start()
	return nil
}

// Stop replicates the RPC client's Stop method.
func (s *NeutrinoClient) Stop() {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
}

// WaitForShutdown replicates the RPC client's WaitForShutdown method.
func (s *NeutrinoClient) WaitForShutdown() {
	<-s.stop
}

// GetBlock replicates the RPC client's GetBlock command.
func (s *NeutrinoClient) GetBlock(hash *chainhash.Hash) (*wire.MsgBlock, er.R) {
	// TODO(roasbeef): add a block cache?
	//  * which evication strategy? depends on use case
	//  Should the block cache be INSIDE neutrino instead of in pktwallet?
	block, err := s.CS.GetBlock(*hash)
	if err != nil {
		return nil, err
	}
	return block.MsgBlock(), nil
}

// GetBestBlock replicates the RPC client's GetBestBlock command.
func (s *NeutrinoClient) GetBestBlock() (*chainhash.Hash, int32, er.R) {
	chainTip, err := s.CS.BestBlock()
	if err != nil {
		return nil, 0, err
	}

	return &chainTip.Hash, chainTip.Height, nil
}

// BlockStamp returns the latest block notified by the client, or an error
// if the client has been shut down.
func (s *NeutrinoClient) BlockStamp() (*waddrmgr.BlockStamp, er.R) {
	if hash, height, err := s.GetBestBlock(); err != nil {
		return nil, err
	} else {
		return &waddrmgr.BlockStamp{Hash: *hash, Height: height}, nil
	}
}

// GetBlockHash returns the block hash for the given height, or an error if the
// client has been shut down or the hash at the block height doesn't exist or
// is unknown.
func (s *NeutrinoClient) GetBlockHash(height int64) (*chainhash.Hash, er.R) {
	return s.CS.GetBlockHash(height)
}

// GetBlockHeader returns the block header for the given block hash, or an error
// if the client has been shut down or the hash doesn't exist or is unknown.
func (s *NeutrinoClient) GetBlockHeader(
	blockHash *chainhash.Hash) (*wire.BlockHeader, er.R) {
	return s.CS.GetBlockHeader(blockHash)
}

// IsCurrent returns whether the chain backend considers its view of the network
// as "current".
func (s *NeutrinoClient) IsCurrent() bool {
	return s.CS.IsCurrent()
}

// SendRawTransaction replicates the RPC client's SendRawTransaction command.
func (s *NeutrinoClient) SendRawTransaction(tx *wire.MsgTx, allowHighFees bool) (
	*chainhash.Hash, er.R) {
	err := s.CS.SendTransaction(tx)
	if err != nil {
		return nil, err
	}
	hash := tx.TxHash()
	return &hash, nil
}

// FilterBlocks scans the blocks contained in the FilterBlocksRequest for any
// addresses of interest. For each requested block, the corresponding compact
// filter will first be checked for matches, skipping those that do not report
// anything. If the filter returns a positive match, the full block will be
// fetched and filtered. This method returns a FilterBlocksReponse for the first
// block containing a matching address. If no matches are found in the range of
// blocks requested, the returned response will be nil.
func (s *NeutrinoClient) FilterBlocks(
	req *FilterBlocksRequest) (*FilterBlocksResponse, er.R) {

	blockFilterer := NewBlockFilterer(s.chainParams, req)

	// Construct the watchlist using the addresses and outpoints contained
	// in the filter blocks request.
	watchList, err := buildFilterBlocksWatchList(req)
	if err != nil {
		return nil, err
	}

	// Iterate over the requested blocks, fetching the compact filter for
	// each one, and matching it against the watchlist generated above. If
	// the filter returns a positive match, the full block is then requested
	// and scanned for addresses using the block filterer.
	for i, blk := range req.Blocks {
		filter, err := s.pollCFilter(&blk.Hash)
		if err != nil {
			return nil, err
		}

		// Skip any empty filters.
		if filter == nil || filter.N() == 0 {
			continue
		}

		key := builder.DeriveKey(&blk.Hash)
		matched, err := filter.MatchAny(key, watchList)
		if err != nil {
			return nil, err
		} else if !matched {
			continue
		}

		log.Tracef("Fetching block height=%d hash=%v", blk.Height, blk.Hash)

		block, err := s.CS.GetBlock(blk.Hash, neutrino.Encoding(wire.BaseEncoding))
		if err != nil {
			return nil, err
		}
		rawBlock := block.MsgBlock()

		if !blockFilterer.FilterBlock(rawBlock) {
			continue
		}

		// If any external or internal addresses were detected in this
		// block, we return them to the caller so that the rescan
		// windows can widened with subsequent addresses. The
		// `BatchIndex` is returned so that the caller can compute the
		// *next* block from which to begin again.
		resp := &FilterBlocksResponse{
			BatchIndex:         uint32(i),
			BlockMeta:          blk,
			FoundExternalAddrs: blockFilterer.FoundExternal,
			FoundInternalAddrs: blockFilterer.FoundInternal,
			FoundOutPoints:     blockFilterer.FoundOutPoints,
			RelevantTxns:       blockFilterer.RelevantTxns,
		}

		return resp, nil
	}

	// No addresses were found for this range.
	return nil, nil
}

// buildFilterBlocksWatchList constructs a watchlist used for matching against a
// cfilter from a FilterBlocksRequest. The watchlist will be populated with all
// external addresses, internal addresses, and outpoints contained in the
// request.
func buildFilterBlocksWatchList(req *FilterBlocksRequest) ([][]byte, er.R) {
	// Construct a watch list containing the script addresses of all
	// internal and external addresses that were requested, in addition to
	// the set of outpoints currently being watched.
	watchListSize := len(req.ExternalAddrs) +
		len(req.InternalAddrs) +
		len(req.ImportedAddrs) +
		len(req.WatchedOutPoints)

	watchList := make([][]byte, 0, watchListSize)

	var err er.R
	add := func(a btcutil.Address) {
		if err != nil {
			return
		}
		p2shAddr, e := txscript.PayToAddrScript(a)
		if err != nil {
			err = e
			return
		}
		watchList = append(watchList, p2shAddr)
	}

	for _, addr := range req.ExternalAddrs {
		add(addr)
	}
	for _, addr := range req.InternalAddrs {
		add(addr)
	}
	for _, addr := range req.ImportedAddrs {
		add(addr)
	}
	for _, addr := range req.WatchedOutPoints {
		add(addr)
	}

	return watchList, err
}

// pollCFilter attempts to fetch a CFilter from the neutrino client. This is
// used to get around the fact that the filter headers may lag behind the
// highest known block header.
func (s *NeutrinoClient) pollCFilter(hash *chainhash.Hash) (*gcs.Filter, er.R) {
	var (
		filter *gcs.Filter
		err    er.R
		count  int
	)

	const maxFilterRetries = 50
	for count < maxFilterRetries {
		if count > 0 {
			time.Sleep(100 * time.Millisecond)
		}

		filter, err = s.CS.GetCFilter(*hash, wire.GCSFilterRegular, neutrino.OptimisticBatch())
		if err != nil {
			count++
			continue
		}

		return filter, nil
	}

	return nil, err
}

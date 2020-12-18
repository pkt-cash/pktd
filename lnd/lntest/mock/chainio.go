package mock

import (
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/wire"
)

// ChainIO is a mock implementation of the BlockChainIO interface.
type ChainIO struct {
	BestHeight int32
}

// GetBestBlock currently returns dummy values.
func (c *ChainIO) GetBestBlock() (*chainhash.Hash, int32, er.R) {
	return chaincfg.TestNet3Params.GenesisHash, c.BestHeight, nil
}

// GetUtxo currently returns dummy values.
func (c *ChainIO) GetUtxo(op *wire.OutPoint, _ []byte,
	heightHint uint32, _ <-chan struct{}) (*wire.TxOut, er.R) {
	return nil, nil
}

// GetBlockHash currently returns dummy values.
func (c *ChainIO) GetBlockHash(blockHeight int64) (*chainhash.Hash, er.R) {
	return nil, nil
}

// GetBlock currently returns dummy values.
func (c *ChainIO) GetBlock(blockHash *chainhash.Hash) (*wire.MsgBlock, er.R) {
	return nil, nil
}

// +build dev

package chainntnfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/btcjson"
	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/chaincfg"
	"github.com/pkt-cash/pktd/chaincfg/chainhash"
	"github.com/pkt-cash/pktd/integration/rpctest"
	"github.com/pkt-cash/pktd/neutrino"
	"github.com/pkt-cash/pktd/pktwallet/walletdb"
	"github.com/pkt-cash/pktd/txscript"
	"github.com/pkt-cash/pktd/txscript/params"
	"github.com/pkt-cash/pktd/wire"
)

// TrickleInterval is the interval at which the miner should trickle
// transactions to its peers. We'll set it small to ensure the miner
// propagates transactions quickly in the tests.
var TrickleInterval = 10 * time.Millisecond

var NetParams = &chaincfg.RegressionNetParams

// randPubKeyHashScript generates a P2PKH script that pays to the public key of
// a randomly-generated private key.
func randPubKeyHashScript() ([]byte, *btcec.PrivateKey, er.R) {
	privKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, err
	}

	pubKeyHash := btcutil.Hash160(privKey.PubKey().SerializeCompressed())
	addrScript, err := btcutil.NewAddressPubKeyHash(pubKeyHash, NetParams)
	if err != nil {
		return nil, nil, err
	}

	pkScript, err := txscript.PayToAddrScript(addrScript)
	if err != nil {
		return nil, nil, err
	}

	return pkScript, privKey, nil
}

// GetTestTxidAndScript generate a new test transaction and returns its txid and
// the script of the output being generated.
func GetTestTxidAndScript(h *rpctest.Harness) (*chainhash.Hash, []byte, er.R) {
	pkScript, _, err := randPubKeyHashScript()
	if err != nil {
		return nil, nil, er.Errorf("unable to generate pkScript: %v", err)
	}
	output := &wire.TxOut{Value: 2e8, PkScript: pkScript}
	txid, err := h.SendOutputs([]*wire.TxOut{output}, 10)
	if err != nil {
		return nil, nil, err
	}

	return txid, pkScript, nil
}

// WaitForMempoolTx waits for the txid to be seen in the miner's mempool.
func WaitForMempoolTx(miner *rpctest.Harness, txid *chainhash.Hash) er.R {
	timeout := time.After(10 * time.Second)
	trickle := time.After(2 * TrickleInterval)
	for {
		// Check for the harness' knowledge of the txid.
		tx, err := miner.Node.GetRawTransaction(txid)
		if err != nil {
			if btcjson.ErrRPCNoTxInfo.Is(err) {
				continue
			}
			return err
		}

		if tx != nil && tx.Hash().IsEqual(txid) {
			break
		}

		select {
		case <-time.After(100 * time.Millisecond):
		case <-timeout:
			return er.New("timed out waiting for tx")
		}
	}

	// To ensure any transactions propagate from the miner to the peers
	// before returning, ensure we have waited for at least
	// 2*trickleInterval before returning.
	select {
	case <-trickle:
	case <-timeout:
		return er.New("timeout waiting for trickle interval. " +
			"Trickle interval to large?")
	}

	return nil
}

// CreateSpendableOutput creates and returns an output that can be spent later
// on.
func CreateSpendableOutput(t *testing.T,
	miner *rpctest.Harness) (*wire.OutPoint, *wire.TxOut, *btcec.PrivateKey) {
	t.Helper()

	// Create a transaction that only has one output, the one destined for
	// the recipient.
	pkScript, privKey, err := randPubKeyHashScript()
	if err != nil {
		t.Fatalf("unable to generate pkScript: %v", err)
	}
	output := &wire.TxOut{Value: 2e8, PkScript: pkScript}
	txid, err := miner.SendOutputsWithoutChange([]*wire.TxOut{output}, 10)
	if err != nil {
		t.Fatalf("unable to create tx: %v", err)
	}

	// Mine the transaction to mark the output as spendable.
	if err := WaitForMempoolTx(miner, txid); err != nil {
		t.Fatalf("tx not relayed to miner: %v", err)
	}
	if _, err := miner.Node.Generate(1); err != nil {
		t.Fatalf("unable to generate single block: %v", err)
	}

	return wire.NewOutPoint(txid, 0), output, privKey
}

// CreateSpendTx creates a transaction spending the specified output.
func CreateSpendTx(t *testing.T, prevOutPoint *wire.OutPoint,
	prevOutput *wire.TxOut, privKey *btcec.PrivateKey) *wire.MsgTx {
	t.Helper()

	spendingTx := wire.NewMsgTx(1)
	spendingTx.AddTxIn(&wire.TxIn{PreviousOutPoint: *prevOutPoint})
	spendingTx.AddTxOut(&wire.TxOut{Value: 1e8, PkScript: prevOutput.PkScript})

	sigScript, err := txscript.SignatureScript(
		spendingTx, 0, prevOutput.PkScript, params.SigHashAll,
		privKey, true,
	)
	if err != nil {
		t.Fatalf("unable to sign tx: %v", err)
	}
	spendingTx.TxIn[0].SignatureScript = sigScript

	return spendingTx
}

// NewMiner spawns testing harness backed by a btcd node that can serve as a
// miner.
func NewMiner(t *testing.T, extraArgs []string, createChain bool,
	spendableOutputs uint32) (*rpctest.Harness, func()) {
	t.Helper()

	// Add the trickle interval argument to the extra args.
	trickle := fmt.Sprintf("--trickleinterval=%v", TrickleInterval)
	extraArgs = append(extraArgs, trickle, "--tls")

	node, err := rpctest.New(NetParams, nil, extraArgs)
	if err != nil {
		t.Fatalf("unable to create backend node: %v", err)
	}
	if err := node.SetUp(createChain, spendableOutputs); err != nil {
		node.TearDown()
		t.Fatalf("unable to set up backend node: %v", err)
	}

	return node, func() { node.TearDown() }
}

// NewNeutrinoBackend spawns a new neutrino node that connects to a miner at
// the specified address.
func NewNeutrinoBackend(t *testing.T, minerAddr string) (*neutrino.ChainService, func()) {
	t.Helper()

	spvDir, errr := ioutil.TempDir("", "neutrino")
	if errr != nil {
		t.Fatalf("unable to create temp dir: %v", errr)
	}

	dbName := filepath.Join(spvDir, "neutrino.db")
	spvDatabase, err := walletdb.Create("bdb", dbName, true)
	if err != nil {
		os.RemoveAll(spvDir)
		t.Fatalf("unable to create walletdb: %v", err)
	}

	// Create an instance of neutrino connected to the running btcd
	// instance.
	spvConfig := neutrino.Config{
		DataDir:      spvDir,
		Database:     spvDatabase,
		ChainParams:  *NetParams,
		ConnectPeers: []string{minerAddr},
	}
	spvNode, err := neutrino.NewChainService(spvConfig)
	if err != nil {
		os.RemoveAll(spvDir)
		spvDatabase.Close()
		t.Fatalf("unable to create neutrino: %v", err)
	}

	// We'll also wait for the instance to sync up fully to the chain
	// generated by the btcd instance.
	spvNode.Start()
	for !spvNode.IsCurrent() {
		time.Sleep(time.Millisecond * 100)
	}

	return spvNode, func() {
		spvNode.Stop()
		spvDatabase.Close()
		os.RemoveAll(spvDir)
	}
}

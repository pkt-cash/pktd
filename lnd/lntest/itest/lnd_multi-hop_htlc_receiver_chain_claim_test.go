package itest

import (
	"context"
	"time"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/btcutil/util"
	"github.com/pkt-cash/pktd/lnd"
	"github.com/pkt-cash/pktd/lnd/lncfg"
	"github.com/pkt-cash/pktd/lnd/lnrpc"
	"github.com/pkt-cash/pktd/lnd/lnrpc/invoicesrpc"
	"github.com/pkt-cash/pktd/lnd/lnrpc/routerrpc"
	"github.com/pkt-cash/pktd/lnd/lntest"
	"github.com/pkt-cash/pktd/lnd/lntest/wait"
	"github.com/pkt-cash/pktd/lnd/lntypes"
	"github.com/pkt-cash/pktd/wire"
	"github.com/stretchr/testify/require"
)

// testMultiHopReceiverChainClaim tests that in the multi-hop setting, if the
// receiver of an HTLC knows the preimage, but wasn't able to settle the HTLC
// off-chain, then it goes on chain to claim the HTLC uing the HTLC success
// transaction. In this scenario, the node that sent the outgoing HTLC should
// extract the preimage from the sweep transaction, and finish settling the
// HTLC backwards into the route.
func testMultiHopReceiverChainClaim(net *lntest.NetworkHarness, t *harnessTest,
	alice, bob *lntest.HarnessNode, c commitType) {
	ctxb := context.Background()

	// First, we'll create a three hop network: Alice -> Bob -> Carol, with
	// Carol refusing to actually settle or directly cancel any HTLC's
	// self.
	aliceChanPoint, bobChanPoint, carol := createThreeHopNetwork(
		t, net, alice, bob, false, c,
	)

	// Clean up carol's node when the test finishes.
	defer shutdownAndAssert(net, t, carol)

	// With the network active, we'll now add a new hodl invoice at Carol's
	// end. Make sure the cltv expiry delta is large enough, otherwise Bob
	// won't send out the outgoing htlc.

	const invoiceAmt = 100000
	preimage := lntypes.Preimage{1, 2, 4}
	payHash := preimage.Hash()
	invoiceReq := &invoicesrpc.AddHoldInvoiceRequest{
		Value:      invoiceAmt,
		CltvExpiry: 40,
		Hash:       payHash[:],
	}
	ctxt, cancel := context.WithTimeout(ctxb, defaultTimeout)
	defer cancel()
	carolInvoice, errr := carol.AddHoldInvoice(ctxt, invoiceReq)
	require.NoError(t.t, errr)

	// Now that we've created the invoice, we'll send a single payment from
	// Alice to Carol. We won't wait for the response however, as Carol
	// will not immediately settle the payment.
	ctx, cancel := context.WithCancel(ctxb)
	defer cancel()

	_, errr = alice.RouterClient.SendPaymentV2(
		ctx, &routerrpc.SendPaymentRequest{
			PaymentRequest: carolInvoice.PaymentRequest,
			TimeoutSeconds: 60,
			FeeLimitMsat:   noFeeLimitMsat,
		},
	)
	require.NoError(t.t, errr)

	// At this point, all 3 nodes should now have an active channel with
	// the created HTLC pending on all of them.
	nodes := []*lntest.HarnessNode{alice, bob, carol}
	err := wait.NoError(func() er.R {
		return assertActiveHtlcs(nodes, payHash[:])
	}, defaultTimeout)
	util.RequireNoErr(t.t, err)

	// Wait for carol to mark invoice as accepted. There is a small gap to
	// bridge between adding the htlc to the channel and executing the exit
	// hop logic.
	waitForInvoiceAccepted(t, carol, payHash)

	restartBob, err := net.SuspendNode(bob)
	util.RequireNoErr(t.t, err)

	// Settle invoice. This will just mark the invoice as settled, as there
	// is no link anymore to remove the htlc from the commitment tx. For
	// this test, it is important to actually settle and not leave the
	// invoice in the accepted state, because without a known preimage, the
	// channel arbitrator won't go to chain.
	ctx, cancel = context.WithTimeout(ctxb, defaultTimeout)
	defer cancel()
	_, errr = carol.SettleInvoice(ctx, &invoicesrpc.SettleInvoiceMsg{
		Preimage: preimage[:],
	})
	require.NoError(t.t, errr)

	// Increase the fee estimate so that the following force close tx will
	// be cpfp'ed.
	net.SetFeeEstimate(30000)

	// Now we'll mine enough blocks to prompt carol to actually go to the
	// chain in order to sweep her HTLC since the value is high enough.
	// TODO(roasbeef): modify once go to chain policy changes
	numBlocks := padCLTV(uint32(
		invoiceReq.CltvExpiry - lncfg.DefaultIncomingBroadcastDelta,
	))
	_, err = net.Miner.Node.Generate(numBlocks)
	util.RequireNoErr(t.t, err)

	// At this point, Carol should broadcast her active commitment
	// transaction in order to go to the chain and sweep her HTLC. If there
	// are anchors, Carol also sweeps hers.
	expectedTxes := 1
	if c == commitTypeAnchors {
		expectedTxes = 2
	}
	_, err = getNTxsFromMempool(
		net.Miner.Node, expectedTxes, minerMempoolTimeout,
	)
	util.RequireNoErr(t.t, err)

	bobFundingTxid, err := lnd.GetChanPointFundingTxid(bobChanPoint)
	util.RequireNoErr(t.t, err)

	carolFundingPoint := wire.OutPoint{
		Hash:  *bobFundingTxid,
		Index: bobChanPoint.OutputIndex,
	}

	// The commitment transaction should be spending from the funding
	// transaction.
	closingTx := getSpendingTxInMempool(
		t, net.Miner.Node, minerMempoolTimeout, carolFundingPoint,
	)
	closingTxid := closingTx.TxHash()

	// Confirm the commitment.
	mineBlocks(t, net, 1, expectedTxes)

	// Restart bob again.
	err = restartBob()
	util.RequireNoErr(t.t, err)

	// After the force close transaction is mined, Carol should broadcast
	// her second level HTLC transaction. Bob will broadcast a sweep tx to
	// sweep his output in the channel with Carol. When Bob notices Carol's
	// second level transaction in the mempool, he will extract the preimage
	// and settle the HTLC back off-chain. Bob will also sweep his anchor,
	// if present.
	expectedTxes = 2
	if c == commitTypeAnchors {
		expectedTxes = 3
	}
	txes, err := getNTxsFromMempool(
		net.Miner.Node, expectedTxes, minerMempoolTimeout,
	)
	util.RequireNoErr(t.t, err)

	// All transactions should be spending from the commitment transaction.
	assertAllTxesSpendFrom(t, txes, closingTxid)

	// We'll now mine an additional block which should confirm both the
	// second layer transactions.
	_, err = net.Miner.Node.Generate(1)
	util.RequireNoErr(t.t, err)

	time.Sleep(time.Second * 4)

	// TODO(roasbeef): assert bob pending state as well

	// Carol's pending channel report should now show two outputs under
	// limbo: her commitment output, as well as the second-layer claim
	// output.
	pendingChansRequest := &lnrpc.PendingChannelsRequest{}
	ctxt, _ = context.WithTimeout(ctxb, defaultTimeout)
	pendingChanResp, errr := carol.PendingChannels(ctxt, pendingChansRequest)
	require.NoError(t.t, errr)

	require.NotZero(t.t, len(pendingChanResp.PendingForceClosingChannels))
	forceCloseChan := pendingChanResp.PendingForceClosingChannels[0]
	require.NotZero(t.t, forceCloseChan.LimboBalance)

	// The pending HTLC carol has should also now be in stage 2.
	require.Len(t.t, forceCloseChan.PendingHtlcs, 1)
	require.Equal(t.t, uint32(2), forceCloseChan.PendingHtlcs[0].Stage)

	// Once the second-level transaction confirmed, Bob should have
	// extracted the preimage from the chain, and sent it back to Alice,
	// clearing the HTLC off-chain.
	nodes = []*lntest.HarnessNode{alice}
	err = wait.NoError(func() er.R {
		return assertNumActiveHtlcs(nodes, 0)
	}, defaultTimeout)
	util.RequireNoErr(t.t, err)

	// If we mine 4 additional blocks, then both outputs should now be
	// mature.
	_, err = net.Miner.Node.Generate(defaultCSV)
	util.RequireNoErr(t.t, err)

	// We should have a new transaction in the mempool.
	_, err = waitForTxInMempool(net.Miner.Node, minerMempoolTimeout)
	util.RequireNoErr(t.t, err)

	// Finally, if we mine an additional block to confirm these two sweep
	// transactions, Carol should not show a pending channel in her report
	// afterwards.
	_, err = net.Miner.Node.Generate(1)
	util.RequireNoErr(t.t, err)
	ctxt, _ = context.WithTimeout(ctxb, defaultTimeout)
	err = waitForNumChannelPendingForceClose(ctxt, carol, 0, nil)
	util.RequireNoErr(t.t, err)

	// The invoice should show as settled for Carol, indicating that it was
	// swept on-chain.
	invoicesReq := &lnrpc.ListInvoiceRequest{}
	invoicesResp, errr := carol.ListInvoices(ctxb, invoicesReq)
	require.NoError(t.t, errr)
	require.Len(t.t, invoicesResp.Invoices, 1)
	invoice := invoicesResp.Invoices[0]
	require.Equal(t.t, lnrpc.Invoice_SETTLED, invoice.State)
	require.Equal(t.t, int64(invoiceAmt), invoice.AmtPaidSat)

	// Finally, check that the Alice's payment is correctly marked
	// succeeded.
	ctxt, _ = context.WithTimeout(ctxt, defaultTimeout)
	err = checkPaymentStatus(
		ctxt, alice, preimage, lnrpc.Payment_SUCCEEDED,
	)
	util.RequireNoErr(t.t, err)

	// We'll close out the channel between Alice and Bob, then shutdown
	// carol to conclude the test.
	ctxt, _ = context.WithTimeout(ctxb, channelCloseTimeout)
	closeChannelAndAssertType(
		ctxt, t, net, alice, aliceChanPoint,
		false, false,
	)
}

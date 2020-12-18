package routing

import (
	"sync"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/lnd/channeldb"
	"github.com/pkt-cash/pktd/lnd/htlcswitch"
	"github.com/pkt-cash/pktd/lnd/lntypes"
	"github.com/pkt-cash/pktd/lnd/lnwire"
	"github.com/pkt-cash/pktd/lnd/routing/route"
)

type mockPaymentAttemptDispatcher struct {
	onPayment func(firstHop lnwire.ShortChannelID) ([32]byte, er.R)
	results   map[uint64]*htlcswitch.PaymentResult

	sync.Mutex
}

var _ PaymentAttemptDispatcher = (*mockPaymentAttemptDispatcher)(nil)

func (m *mockPaymentAttemptDispatcher) SendHTLC(firstHop lnwire.ShortChannelID,
	pid uint64,
	_ *lnwire.UpdateAddHTLC) er.R {
	if m.onPayment == nil {
		return nil
	}

	var result *htlcswitch.PaymentResult
	preimage, err := m.onPayment(firstHop)
	if err != nil {
		_, ok := er.Wrapped(err).(htlcswitch.ClearTextError)
		if !ok {
			return err
		}
		result = &htlcswitch.PaymentResult{
			Error: err,
		}
	} else {
		result = &htlcswitch.PaymentResult{Preimage: preimage}
	}

	m.Lock()
	if m.results == nil {
		m.results = make(map[uint64]*htlcswitch.PaymentResult)
	}

	m.results[pid] = result
	m.Unlock()

	return nil
}

func (m *mockPaymentAttemptDispatcher) GetPaymentResult(paymentID uint64,
	_ lntypes.Hash, _ htlcswitch.ErrorDecrypter) (
	<-chan *htlcswitch.PaymentResult, er.R) {
	c := make(chan *htlcswitch.PaymentResult, 1)

	m.Lock()
	res, ok := m.results[paymentID]
	m.Unlock()

	if !ok {
		return nil, htlcswitch.ErrPaymentIDNotFound.Default()
	}
	c <- res

	return c, nil
}

func (m *mockPaymentAttemptDispatcher) CleanStore(map[uint64]struct{}) er.R {
	return nil
}

func (m *mockPaymentAttemptDispatcher) setPaymentResult(
	f func(firstHop lnwire.ShortChannelID) ([32]byte, er.R)) {
	m.onPayment = f
}

type mockPaymentSessionSource struct {
	routes []*route.Route
}

var _ PaymentSessionSource = (*mockPaymentSessionSource)(nil)

func (m *mockPaymentSessionSource) NewPaymentSession(
	_ *LightningPayment) (PaymentSession, er.R) {
	return &mockPaymentSession{m.routes}, nil
}

func (m *mockPaymentSessionSource) NewPaymentSessionForRoute(
	preBuiltRoute *route.Route) PaymentSession {
	return nil
}

func (m *mockPaymentSessionSource) NewPaymentSessionEmpty() PaymentSession {
	return &mockPaymentSession{}
}

type mockMissionControl struct {
}

var _ MissionController = (*mockMissionControl)(nil)

func (m *mockMissionControl) ReportPaymentFail(paymentID uint64, rt *route.Route,
	failureSourceIdx *int, failure lnwire.FailureMessage) (
	*channeldb.FailureReason, er.R) {
	// Report a permanent failure if this is an error caused
	// by incorrect details.
	if failure.Code() == lnwire.CodeIncorrectOrUnknownPaymentDetails {
		reason := channeldb.FailureReasonPaymentDetails
		return &reason, nil
	}

	return nil, nil
}

func (m *mockMissionControl) ReportPaymentSuccess(paymentID uint64,
	rt *route.Route) er.R {
	return nil
}

func (m *mockMissionControl) GetProbability(fromNode, toNode route.Vertex,
	amt lnwire.MilliSatoshi) float64 {
	return 0
}

type mockPaymentSession struct {
	routes []*route.Route
}

var _ PaymentSession = (*mockPaymentSession)(nil)

func (m *mockPaymentSession) RequestRoute(_, _ lnwire.MilliSatoshi,
	_, height uint32) (*route.Route, er.R) {
	if len(m.routes) == 0 {
		return nil, er.E(errNoPathFound)
	}

	r := m.routes[0]
	m.routes = m.routes[1:]

	return r, nil
}

type mockPayer struct {
	sendResult       chan er.R
	paymentResultErr chan er.R
	paymentResult    chan *htlcswitch.PaymentResult
	quit             chan struct{}
}

var _ PaymentAttemptDispatcher = (*mockPayer)(nil)

func (m *mockPayer) SendHTLC(_ lnwire.ShortChannelID,
	paymentID uint64,
	_ *lnwire.UpdateAddHTLC) er.R {
	select {
	case res := <-m.sendResult:
		return res
	case <-m.quit:
		return er.Errorf("test quitting")
	}
}

func (m *mockPayer) GetPaymentResult(paymentID uint64, _ lntypes.Hash,
	_ htlcswitch.ErrorDecrypter) (<-chan *htlcswitch.PaymentResult, er.R) {
	select {
	case res := <-m.paymentResult:
		resChan := make(chan *htlcswitch.PaymentResult, 1)
		resChan <- res
		return resChan, nil
	case err := <-m.paymentResultErr:
		return nil, err
	case <-m.quit:
		return nil, er.Errorf("test quitting")
	}
}

func (m *mockPayer) CleanStore(pids map[uint64]struct{}) er.R {
	return nil
}

type initArgs struct {
	c *channeldb.PaymentCreationInfo
}

type registerAttemptArgs struct {
	a *channeldb.HTLCAttemptInfo
}

type settleAttemptArgs struct {
	preimg lntypes.Preimage
}

type failAttemptArgs struct {
	reason *channeldb.HTLCFailInfo
}

type failPaymentArgs struct {
	reason channeldb.FailureReason
}

type testPayment struct {
	info     channeldb.PaymentCreationInfo
	attempts []channeldb.HTLCAttempt
}

type mockControlTower struct {
	payments   map[lntypes.Hash]*testPayment
	successful map[lntypes.Hash]struct{}
	failed     map[lntypes.Hash]channeldb.FailureReason

	init            chan initArgs
	registerAttempt chan registerAttemptArgs
	settleAttempt   chan settleAttemptArgs
	failAttempt     chan failAttemptArgs
	failPayment     chan failPaymentArgs
	fetchInFlight   chan struct{}

	sync.Mutex
}

var _ ControlTower = (*mockControlTower)(nil)

func makeMockControlTower() *mockControlTower {
	return &mockControlTower{
		payments:   make(map[lntypes.Hash]*testPayment),
		successful: make(map[lntypes.Hash]struct{}),
		failed:     make(map[lntypes.Hash]channeldb.FailureReason),
	}
}

func (m *mockControlTower) InitPayment(phash lntypes.Hash,
	c *channeldb.PaymentCreationInfo) er.R {
	m.Lock()
	defer m.Unlock()

	if m.init != nil {
		m.init <- initArgs{c}
	}

	// Don't allow re-init a successful payment.
	if _, ok := m.successful[phash]; ok {
		return channeldb.ErrAlreadyPaid.Default()
	}

	_, failed := m.failed[phash]
	_, ok := m.payments[phash]

	// If the payment is known, only allow re-init if failed.
	if ok && !failed {
		return channeldb.ErrPaymentInFlight.Default()
	}

	delete(m.failed, phash)
	m.payments[phash] = &testPayment{
		info: *c,
	}

	return nil
}

func (m *mockControlTower) RegisterAttempt(phash lntypes.Hash,
	a *channeldb.HTLCAttemptInfo) er.R {
	m.Lock()
	defer m.Unlock()

	if m.registerAttempt != nil {
		m.registerAttempt <- registerAttemptArgs{a}
	}

	// Cannot register attempts for successful or failed payments.
	if _, ok := m.successful[phash]; ok {
		return channeldb.ErrPaymentAlreadySucceeded.Default()
	}

	if _, ok := m.failed[phash]; ok {
		return channeldb.ErrPaymentAlreadyFailed.Default()
	}

	p, ok := m.payments[phash]
	if !ok {
		return channeldb.ErrPaymentNotInitiated.Default()
	}

	p.attempts = append(p.attempts, channeldb.HTLCAttempt{
		HTLCAttemptInfo: *a,
	})
	m.payments[phash] = p

	return nil
}

func (m *mockControlTower) SettleAttempt(phash lntypes.Hash,
	pid uint64, settleInfo *channeldb.HTLCSettleInfo) (
	*channeldb.HTLCAttempt, er.R) {
	m.Lock()
	defer m.Unlock()

	if m.settleAttempt != nil {
		m.settleAttempt <- settleAttemptArgs{settleInfo.Preimage}
	}

	// Only allow setting attempts if the payment is known.
	p, ok := m.payments[phash]
	if !ok {
		return nil, channeldb.ErrPaymentNotInitiated.Default()
	}

	// Find the attempt with this pid, and set the settle info.
	for i, a := range p.attempts {
		if a.AttemptID != pid {
			continue
		}

		if a.Settle != nil {
			return nil, channeldb.ErrAttemptAlreadySettled.Default()
		}
		if a.Failure != nil {
			return nil, channeldb.ErrAttemptAlreadyFailed.Default()
		}

		p.attempts[i].Settle = settleInfo

		// Mark the payment successful on first settled attempt.
		m.successful[phash] = struct{}{}
		return &channeldb.HTLCAttempt{
			Settle: settleInfo,
		}, nil
	}

	return nil, er.Errorf("pid not found")
}

func (m *mockControlTower) FailAttempt(phash lntypes.Hash, pid uint64,
	failInfo *channeldb.HTLCFailInfo) (*channeldb.HTLCAttempt, er.R) {
	m.Lock()
	defer m.Unlock()

	if m.failAttempt != nil {
		m.failAttempt <- failAttemptArgs{failInfo}
	}

	// Only allow failing attempts if the payment is known.
	p, ok := m.payments[phash]
	if !ok {
		return nil, channeldb.ErrPaymentNotInitiated.Default()
	}

	// Find the attempt with this pid, and set the failure info.
	for i, a := range p.attempts {
		if a.AttemptID != pid {
			continue
		}

		if a.Settle != nil {
			return nil, channeldb.ErrAttemptAlreadySettled.Default()
		}
		if a.Failure != nil {
			return nil, channeldb.ErrAttemptAlreadyFailed.Default()
		}

		p.attempts[i].Failure = failInfo
		return &channeldb.HTLCAttempt{
			Failure: failInfo,
		}, nil
	}

	return nil, er.Errorf("pid not found")
}

func (m *mockControlTower) Fail(phash lntypes.Hash,
	reason channeldb.FailureReason) er.R {
	m.Lock()
	defer m.Unlock()

	if m.failPayment != nil {
		m.failPayment <- failPaymentArgs{reason}
	}

	// Payment must be known.
	if _, ok := m.payments[phash]; !ok {
		return channeldb.ErrPaymentNotInitiated.Default()
	}

	m.failed[phash] = reason

	return nil
}

func (m *mockControlTower) FetchPayment(phash lntypes.Hash) (
	*channeldb.MPPayment, er.R) {
	m.Lock()
	defer m.Unlock()

	p, ok := m.payments[phash]
	if !ok {
		return nil, channeldb.ErrPaymentNotInitiated.Default()
	}

	mp := &channeldb.MPPayment{
		Info: &p.info,
	}

	reason, ok := m.failed[phash]
	if ok {
		mp.FailureReason = &reason
	}

	// Return a copy of the current attempts.
	mp.HTLCs = append(mp.HTLCs, p.attempts...)

	return mp, nil
}

func (m *mockControlTower) FetchInFlightPayments() (
	[]*channeldb.InFlightPayment, er.R) {
	m.Lock()
	defer m.Unlock()

	if m.fetchInFlight != nil {
		m.fetchInFlight <- struct{}{}
	}

	// In flight are all payments not successful or failed.
	var fl []*channeldb.InFlightPayment
	for hash, p := range m.payments {
		if _, ok := m.successful[hash]; ok {
			continue
		}
		if _, ok := m.failed[hash]; ok {
			continue
		}

		ifl := channeldb.InFlightPayment{
			Info: &p.info,
		}

		fl = append(fl, &ifl)
	}

	return fl, nil
}

func (m *mockControlTower) SubscribePayment(paymentHash lntypes.Hash) (
	*ControlTowerSubscriber, er.R) {
	return nil, er.New("not implemented")
}

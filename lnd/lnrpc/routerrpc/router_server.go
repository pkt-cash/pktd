package routerrpc

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/lnd/channeldb"
	"github.com/pkt-cash/pktd/lnd/lnrpc"
	"github.com/pkt-cash/pktd/lnd/lntypes"
	"github.com/pkt-cash/pktd/lnd/lnwire"
	"github.com/pkt-cash/pktd/lnd/macaroons"
	"github.com/pkt-cash/pktd/lnd/routing"
	"github.com/pkt-cash/pktd/lnd/routing/route"
	"github.com/pkt-cash/pktd/pktlog/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/macaroon-bakery.v2/bakery"
)

const (
	// subServerName is the name of the sub rpc server. We'll use this name
	// to register ourselves, and we also require that the main
	// SubServerConfigDispatcher instance recognize as the name of our
	subServerName = "RouterRPC"
)

var (
	errServerShuttingDown = er.GenericErrorType.CodeWithDetail("errServerShuttingDown",
		"routerrpc server shutting down")

	// ErrInterceptorAlreadyExists is an error returned when the a new stream
	// is opened and there is already one active interceptor.
	// The user must disconnect prior to open another stream.
	ErrInterceptorAlreadyExists = er.GenericErrorType.CodeWithDetail("ErrInterceptorAlreadyExists",
		"interceptor already exists")

	// macaroonOps are the set of capabilities that our minted macaroon (if
	// it doesn't already exist) will have.
	macaroonOps = []bakery.Op{
		{
			Entity: "offchain",
			Action: "read",
		},
		{
			Entity: "offchain",
			Action: "write",
		},
	}

	// macPermissions maps RPC calls to the permissions they require.
	macPermissions = map[string][]bakery.Op{
		"/routerrpc.Router/SendPaymentV2": {{
			Entity: "offchain",
			Action: "write",
		}},
		"/routerrpc.Router/SendToRouteV2": {{
			Entity: "offchain",
			Action: "write",
		}},
		"/routerrpc.Router/SendToRoute": {{
			Entity: "offchain",
			Action: "write",
		}},
		"/routerrpc.Router/TrackPaymentV2": {{
			Entity: "offchain",
			Action: "read",
		}},
		"/routerrpc.Router/EstimateRouteFee": {{
			Entity: "offchain",
			Action: "read",
		}},
		"/routerrpc.Router/QueryMissionControl": {{
			Entity: "offchain",
			Action: "read",
		}},
		"/routerrpc.Router/QueryProbability": {{
			Entity: "offchain",
			Action: "read",
		}},
		"/routerrpc.Router/ResetMissionControl": {{
			Entity: "offchain",
			Action: "write",
		}},
		"/routerrpc.Router/BuildRoute": {{
			Entity: "offchain",
			Action: "read",
		}},
		"/routerrpc.Router/SubscribeHtlcEvents": {{
			Entity: "offchain",
			Action: "read",
		}},
		"/routerrpc.Router/SendPayment": {{
			Entity: "offchain",
			Action: "write",
		}},
		"/routerrpc.Router/TrackPayment": {{
			Entity: "offchain",
			Action: "read",
		}},
		"/routerrpc.Router/HtlcInterceptor": {{
			Entity: "offchain",
			Action: "write",
		}},
	}

	// DefaultRouterMacFilename is the default name of the router macaroon
	// that we expect to find via a file handle within the main
	// configuration file in this package.
	DefaultRouterMacFilename = "router.macaroon"
)

// Server is a stand alone sub RPC server which exposes functionality that
// allows clients to route arbitrary payment through the Lightning Network.
type Server struct {
	started                  int32 // To be used atomically.
	shutdown                 int32 // To be used atomically.
	forwardInterceptorActive int32 // To be used atomically.

	cfg *Config

	quit chan struct{}
}

// A compile time check to ensure that Server fully implements the RouterServer
// gRPC service.
var _ RouterServer = (*Server)(nil)

// New creates a new instance of the RouterServer given a configuration struct
// that contains all external dependencies. If the target macaroon exists, and
// we're unable to create it, then an error will be returned. We also return
// the set of permissions that we require as a server. At the time of writing
// of this documentation, this is the same macaroon as as the admin macaroon.
func New(cfg *Config) (*Server, lnrpc.MacaroonPerms, er.R) {
	// If the path of the router macaroon wasn't generated, then we'll
	// assume that it's found at the default network directory.
	if cfg.RouterMacPath == "" {
		cfg.RouterMacPath = filepath.Join(
			cfg.NetworkDir, DefaultRouterMacFilename,
		)
	}

	// Now that we know the full path of the router macaroon, we can check
	// to see if we need to create it or not. If stateless_init is set
	// then we don't write the macaroons.
	macFilePath := cfg.RouterMacPath
	if cfg.MacService != nil && !cfg.MacService.StatelessInit &&
		!lnrpc.FileExists(macFilePath) {

		log.Infof("Making macaroons for Router RPC Server at: %v",
			macFilePath)

		// At this point, we know that the router macaroon doesn't yet,
		// exist, so we need to create it with the help of the main
		// macaroon service.
		routerMac, err := cfg.MacService.NewMacaroon(
			context.Background(), macaroons.DefaultRootKeyID,
			macaroonOps...,
		)
		if err != nil {
			return nil, nil, err
		}
		routerMacBytes, errr := routerMac.M().MarshalBinary()
		if errr != nil {
			return nil, nil, er.E(errr)
		}
		errr = ioutil.WriteFile(macFilePath, routerMacBytes, 0o644)
		if errr != nil {
			_ = os.Remove(macFilePath)
			return nil, nil, er.E(errr)
		}
	}

	routerServer := &Server{
		cfg:  cfg,
		quit: make(chan struct{}),
	}

	return routerServer, macPermissions, nil
}

// Start launches any helper goroutines required for the rpcServer to function.
//
// NOTE: This is part of the lnrpc.SubServer interface.
func (s *Server) Start() er.R {
	if atomic.AddInt32(&s.started, 1) != 1 {
		return nil
	}

	return nil
}

// Stop signals any active goroutines for a graceful closure.
//
// NOTE: This is part of the lnrpc.SubServer interface.
func (s *Server) Stop() er.R {
	if atomic.AddInt32(&s.shutdown, 1) != 1 {
		return nil
	}

	close(s.quit)
	return nil
}

// Name returns a unique string representation of the sub-server. This can be
// used to identify the sub-server and also de-duplicate them.
//
// NOTE: This is part of the lnrpc.SubServer interface.
func (s *Server) Name() string {
	return subServerName
}

// RegisterWithRootServer will be called by the root gRPC server to direct a
// sub RPC server to register itself with the main gRPC root server. Until this
// is called, each sub-server won't be able to have requests routed towards it.
//
// NOTE: This is part of the lnrpc.SubServer interface.
func (s *Server) RegisterWithRootServer(grpcServer *grpc.Server) er.R {
	// We make sure that we register it with the main gRPC server to ensure
	// all our methods are routed properly.
	RegisterRouterServer(grpcServer, s)

	log.Debugf("Router RPC server successfully register with root gRPC " +
		"server")

	return nil
}

// RegisterWithRestServer will be called by the root REST mux to direct a sub
// RPC server to register itself with the main REST mux server. Until this is
// called, each sub-server won't be able to have requests routed towards it.
//
// NOTE: This is part of the lnrpc.SubServer interface.
func (s *Server) RegisterWithRestServer(ctx context.Context,
	mux *runtime.ServeMux, dest string, opts []grpc.DialOption) er.R {
	// We make sure that we register it with the main REST server to ensure
	// all our methods are routed properly.
	errr := RegisterRouterHandlerFromEndpoint(ctx, mux, dest, opts)
	if errr != nil {
		log.Errorf("Could not register Router REST server "+
			"with root REST server: %v", errr)
		return er.E(errr)
	}

	log.Debugf("Router REST server successfully registered with " +
		"root REST server")
	return nil
}

// SendPaymentV2 attempts to route a payment described by the passed
// PaymentRequest to the final destination. If we are unable to route the
// payment, or cannot find a route that satisfies the constraints in the
// PaymentRequest, then an error will be returned. Otherwise, the payment
// pre-image, along with the final route will be returned.
func (s *Server) SendPaymentV2(req *SendPaymentRequest,
	stream Router_SendPaymentV2Server) error {
	payment, err := s.cfg.RouterBackend.extractIntentFromSendRequest(req)
	if err != nil {
		return er.Native(err)
	}

	err = s.cfg.Router.SendPaymentAsync(payment)
	if err != nil {
		// Transform user errors to grpc code.
		if channeldb.ErrPaymentInFlight.Is(err) ||
			channeldb.ErrAlreadyPaid.Is(err) {

			log.Debugf("SendPayment async result for hash %x: %v",
				payment.PaymentHash, err)

			return status.Error(
				codes.AlreadyExists, err.String(),
			)
		}

		log.Errorf("SendPayment async error for hash %x: %v",
			payment.PaymentHash, err)

		return er.Native(err)
	}

	return s.trackPayment(payment.PaymentHash, stream, req.NoInflightUpdates)
}

// EstimateRouteFee allows callers to obtain a lower bound w.r.t how much it
// may cost to send an HTLC to the target end destination.
func (s *Server) EstimateRouteFee(ctx context.Context,
	req *RouteFeeRequest) (*RouteFeeResponse, error) {
	if len(req.Dest) != 33 {
		return nil, er.Native(er.New("invalid length destination key"))
	}
	var destNode route.Vertex
	copy(destNode[:], req.Dest)

	// Next, we'll convert the amount in satoshis to mSAT, which are the
	// native unit of LN.
	amtMsat := lnwire.NewMSatFromSatoshis(btcutil.Amount(req.AmtSat))

	// Pick a fee limit
	//
	// TODO: Change this into behavior that makes more sense.
	feeLimit := lnwire.NewMSatFromSatoshis(btcutil.UnitsPerCoin())

	// Finally, we'll query for a route to the destination that can carry
	// that target amount, we'll only request a single route. Set a
	// restriction for the default CLTV limit, otherwise we can find a route
	// that exceeds it and is useless to us.
	mc := s.cfg.RouterBackend.MissionControl
	route, err := s.cfg.Router.FindRoute(
		s.cfg.RouterBackend.SelfNode, destNode, amtMsat,
		&routing.RestrictParams{
			FeeLimit:          feeLimit,
			CltvLimit:         s.cfg.RouterBackend.MaxTotalTimelock,
			ProbabilitySource: mc.GetProbability,
		}, nil, nil, s.cfg.RouterBackend.DefaultFinalCltvDelta,
	)
	if err != nil {
		return nil, er.Native(err)
	}

	return &RouteFeeResponse{
		RoutingFeeMsat: int64(route.TotalFees()),
		TimeLockDelay:  int64(route.TotalTimeLock),
	}, nil
}

// SendToRouteV2 sends a payment through a predefined route. The response of this
// call contains structured error information.
func (s *Server) SendToRouteV2(ctx context.Context,
	req *SendToRouteRequest) (*lnrpc.HTLCAttempt, error) {
	if req.Route == nil {
		return nil, er.Native(er.Errorf("unable to send, no routes provided"))
	}

	route, err := s.cfg.RouterBackend.UnmarshalRoute(req.Route)
	if err != nil {
		return nil, er.Native(err)
	}

	hash, err := lntypes.MakeHash(req.PaymentHash)
	if err != nil {
		return nil, er.Native(err)
	}

	// Pass route to the router. This call returns the full htlc attempt
	// information as it is stored in the database. It is possible that both
	// the attempt return value and err are non-nil. This can happen when
	// the attempt was already initiated before the error happened. In that
	// case, we give precedence to the attempt information as stored in the
	// db.
	attempt, err := s.cfg.Router.SendToRoute(hash, route)
	if attempt != nil {
		rpcAttempt, err := s.cfg.RouterBackend.MarshalHTLCAttempt(
			*attempt,
		)
		if err != nil {
			return nil, er.Native(err)
		}
		return rpcAttempt, nil
	}

	// Transform user errors to grpc code.
	if channeldb.ErrPaymentInFlight.Is(err) ||
		channeldb.ErrAlreadyPaid.Is(err) {

		return nil, status.Error(codes.AlreadyExists, err.String())
	}

	return nil, er.Native(err)
}

// ResetMissionControl clears all mission control state and starts with a clean
// slate.
func (s *Server) ResetMissionControl(ctx context.Context,
	req *ResetMissionControlRequest) (*ResetMissionControlResponse, error) {
	err := s.cfg.RouterBackend.MissionControl.ResetHistory()
	if err != nil {
		return nil, er.Native(err)
	}

	return &ResetMissionControlResponse{}, nil
}

// QueryMissionControl exposes the internal mission control state to callers. It
// is a development feature.
func (s *Server) QueryMissionControl(ctx context.Context,
	req *QueryMissionControlRequest) (*QueryMissionControlResponse, error) {
	snapshot := s.cfg.RouterBackend.MissionControl.GetHistorySnapshot()

	rpcPairs := make([]*PairHistory, 0, len(snapshot.Pairs))
	for _, p := range snapshot.Pairs {
		// Prevent binding to loop variable.
		pair := p

		rpcPair := PairHistory{
			NodeFrom: pair.Pair.From[:],
			NodeTo:   pair.Pair.To[:],
			History:  toRPCPairData(&pair.TimedPairResult),
		}

		rpcPairs = append(rpcPairs, &rpcPair)
	}

	response := QueryMissionControlResponse{
		Pairs: rpcPairs,
	}

	return &response, nil
}

// toRPCPairData marshals mission control pair data to the rpc struct.
func toRPCPairData(data *routing.TimedPairResult) *PairData {
	rpcData := PairData{
		FailAmtSat:     int64(data.FailAmt.ToSatoshis()),
		FailAmtMsat:    int64(data.FailAmt),
		SuccessAmtSat:  int64(data.SuccessAmt.ToSatoshis()),
		SuccessAmtMsat: int64(data.SuccessAmt),
	}

	if !data.FailTime.IsZero() {
		rpcData.FailTime = data.FailTime.Unix()
	}

	if !data.SuccessTime.IsZero() {
		rpcData.SuccessTime = data.SuccessTime.Unix()
	}

	return &rpcData
}

// QueryProbability returns the current success probability estimate for a
// given node pair and amount.
func (s *Server) QueryProbability(ctx context.Context,
	req *QueryProbabilityRequest) (*QueryProbabilityResponse, error) {
	fromNode, err := route.NewVertexFromBytes(req.FromNode)
	if err != nil {
		return nil, er.Native(err)
	}

	toNode, err := route.NewVertexFromBytes(req.ToNode)
	if err != nil {
		return nil, er.Native(err)
	}

	amt := lnwire.MilliSatoshi(req.AmtMsat)

	mc := s.cfg.RouterBackend.MissionControl
	prob := mc.GetProbability(fromNode, toNode, amt)
	history := mc.GetPairHistorySnapshot(fromNode, toNode)

	return &QueryProbabilityResponse{
		Probability: prob,
		History:     toRPCPairData(&history),
	}, nil
}

// TrackPaymentV2 returns a stream of payment state updates. The stream is
// closed when the payment completes.
func (s *Server) TrackPaymentV2(request *TrackPaymentRequest,
	stream Router_TrackPaymentV2Server) error {
	paymentHash, err := lntypes.MakeHash(request.PaymentHash)
	if err != nil {
		return er.Native(err)
	}

	log.Debugf("TrackPayment called for payment %v", paymentHash)

	return s.trackPayment(paymentHash, stream, request.NoInflightUpdates)
}

// trackPayment writes payment status updates to the provided stream.
func (s *Server) trackPayment(paymentHash lntypes.Hash,
	stream Router_TrackPaymentV2Server, noInflightUpdates bool) error {
	router := s.cfg.RouterBackend

	// Subscribe to the outcome of this payment.
	subscription, err := router.Tower.SubscribePayment(
		paymentHash,
	)
	switch {
	case channeldb.ErrPaymentNotInitiated.Is(err):
		return status.Error(codes.NotFound, err.String())
	case err != nil:
		return er.Native(err)
	}
	defer subscription.Close()

	// Stream updates back to the client. The first update is always the
	// current state of the payment.
	for {
		select {
		case item, ok := <-subscription.Updates:
			if !ok {
				// No more payment updates.
				return nil
			}
			result := item.(*channeldb.MPPayment)

			// Skip in-flight updates unless requested.
			if noInflightUpdates &&
				result.Status == channeldb.StatusInFlight {

				continue
			}

			rpcPayment, err := router.MarshalPayment(result)
			if err != nil {
				return er.Native(err)
			}

			// Send event to the client.
			errr := stream.Send(rpcPayment)
			if errr != nil {
				return errr
			}

		case <-s.quit:
			return er.Native(errServerShuttingDown.Default())

		case <-stream.Context().Done():
			log.Debugf("Payment status stream %v canceled", paymentHash)
			return stream.Context().Err()
		}
	}
}

// BuildRoute builds a route from a list of hop addresses.
func (s *Server) BuildRoute(ctx context.Context,
	req *BuildRouteRequest) (*BuildRouteResponse, error) {
	// Unmarshal hop list.
	hops := make([]route.Vertex, len(req.HopPubkeys))
	for i, pubkeyBytes := range req.HopPubkeys {
		pubkey, err := route.NewVertexFromBytes(pubkeyBytes)
		if err != nil {
			return nil, er.Native(err)
		}
		hops[i] = pubkey
	}

	// Prepare BuildRoute call parameters from rpc request.
	var amt *lnwire.MilliSatoshi
	if req.AmtMsat != 0 {
		rpcAmt := lnwire.MilliSatoshi(req.AmtMsat)
		amt = &rpcAmt
	}

	var outgoingChan *uint64
	if req.OutgoingChanId != 0 {
		outgoingChan = &req.OutgoingChanId
	}

	// Build the route and return it to the caller.
	route, err := s.cfg.Router.BuildRoute(
		amt, hops, outgoingChan, req.FinalCltvDelta,
	)
	if err != nil {
		return nil, er.Native(err)
	}

	rpcRoute, err := s.cfg.RouterBackend.MarshalRoute(route)
	if err != nil {
		return nil, er.Native(err)
	}

	routeResp := &BuildRouteResponse{
		Route: rpcRoute,
	}

	return routeResp, nil
}

// SubscribeHtlcEvents creates a uni-directional stream from the server to
// the client which delivers a stream of htlc events.
func (s *Server) SubscribeHtlcEvents(req *SubscribeHtlcEventsRequest,
	stream Router_SubscribeHtlcEventsServer) error {
	htlcClient, err := s.cfg.RouterBackend.SubscribeHtlcEvents()
	if err != nil {
		return er.Native(err)
	}
	defer htlcClient.Cancel()

	for {
		select {
		case event := <-htlcClient.Updates():
			rpcEvent, err := rpcHtlcEvent(event)
			if err != nil {
				return er.Native(err)
			}

			if err := stream.Send(rpcEvent); err != nil {
				return err
			}

		// If the stream's context is canceled, return an error.
		case <-stream.Context().Done():
			log.Debugf("htlc event stream canceled")
			return stream.Context().Err()

		// If the subscribe client terminates, exit with an error.
		case <-htlcClient.Quit():
			return er.Native(er.New("htlc event subscription terminated"))

		// If the server has been signaled to shut down, exit.
		case <-s.quit:
			return er.Native(errServerShuttingDown.Default())
		}
	}
}

// HtlcInterceptor is a bidirectional stream for streaming interception
// requests to the caller.
// Upon connection it does the following:
// 1. Check if there is already a live stream, if yes it rejects the request.
// 2. Regsitered a ForwardInterceptor
// 3. Delivers to the caller every √√ and detect his answer.
// It uses a local implementation of holdForwardsStore to keep all the hold
// forwards and find them when manual resolution is later needed.
func (s *Server) HtlcInterceptor(stream Router_HtlcInterceptorServer) error {
	// We ensure there is only one interceptor at a time.
	if !atomic.CompareAndSwapInt32(&s.forwardInterceptorActive, 0, 1) {
		return er.Native(ErrInterceptorAlreadyExists.Default())
	}
	defer atomic.CompareAndSwapInt32(&s.forwardInterceptorActive, 1, 0)

	// run the forward interceptor.
	return er.Native(newForwardInterceptor(s, stream).run())
}

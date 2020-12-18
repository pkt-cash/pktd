package lnd

import (
	"fmt"
	"net"

	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/lnd/autopilot"
	"github.com/pkt-cash/pktd/lnd/chainreg"
	"github.com/pkt-cash/pktd/lnd/lncfg"
	"github.com/pkt-cash/pktd/lnd/lnwire"
	"github.com/pkt-cash/pktd/lnd/tor"
	"github.com/pkt-cash/pktd/pktlog/log"
	"github.com/pkt-cash/pktd/wire"
)

// validateAtplConfig is a helper method that makes sure the passed
// configuration is sane. Currently it checks that the heuristic configuration
// makes sense. In case the config is valid, it will return a list of
// WeightedHeuristics that can be combined for use with the autopilot agent.
func validateAtplCfg(cfg *lncfg.AutoPilot) ([]*autopilot.WeightedHeuristic,
	er.R) {
	var (
		heuristicsStr string
		sum           float64
		heuristics    []*autopilot.WeightedHeuristic
	)

	// Create a help text that we can return in case the config is not
	// correct.
	for _, a := range autopilot.AvailableHeuristics {
		heuristicsStr += fmt.Sprintf(" '%v' ", a.Name())
	}
	availStr := fmt.Sprintf("Available heuristics are: [%v]", heuristicsStr)

	// We'll go through the config and make sure all the heuristics exists,
	// and that the sum of their weights is 1.0.
	for name, weight := range cfg.Heuristic {
		a, ok := autopilot.AvailableHeuristics[name]
		if !ok {
			// No heuristic matching this config option was found.
			return nil, er.Errorf("heuristic %v not available. %v",
				name, availStr)
		}

		// If this heuristic was among the registered ones, we add it
		// to the list we'll give to the agent, and keep track of the
		// sum of weights.
		heuristics = append(
			heuristics,
			&autopilot.WeightedHeuristic{
				Weight:              weight,
				AttachmentHeuristic: a,
			},
		)
		sum += weight
	}

	// Check found heuristics. We must have at least one to operate.
	if len(heuristics) == 0 {
		return nil, er.Errorf("no active heuristics: %v", availStr)
	}

	if sum != 1.0 {
		return nil, er.Errorf("heuristic weights must sum to 1.0")
	}
	return heuristics, nil
}

// chanController is an implementation of the autopilot.ChannelController
// interface that's backed by a running lnd instance.
type chanController struct {
	server        *server
	private       bool
	minConfs      int32
	confTarget    uint32
	chanMinHtlcIn lnwire.MilliSatoshi
	netParams     chainreg.BitcoinNetParams
}

// OpenChannel opens a channel to a target peer, with a capacity of the
// specified amount. This function should un-block immediately after the
// funding transaction that marks the channel open has been broadcast.
func (c *chanController) OpenChannel(target *btcec.PublicKey,
	amt btcutil.Amount) er.R {
	// With the connection established, we'll now establish our connection
	// to the target peer, waiting for the first update before we exit.
	feePerKw, err := c.server.cc.FeeEstimator.EstimateFeePerKW(
		c.confTarget,
	)
	if err != nil {
		return err
	}

	// Construct the open channel request and send it to the server to begin
	// the funding workflow.
	req := &openChanReq{
		targetPubkey:     target,
		chainHash:        *c.netParams.GenesisHash,
		subtractFees:     true,
		localFundingAmt:  amt,
		pushAmt:          0,
		minHtlcIn:        c.chanMinHtlcIn,
		fundingFeePerKw:  feePerKw,
		private:          c.private,
		remoteCsvDelay:   0,
		minConfs:         c.minConfs,
		maxValueInFlight: 0,
	}

	updateStream, errChan := c.server.OpenChannel(req)
	select {
	case err := <-errChan:
		return err
	case <-updateStream:
		return nil
	case <-c.server.quit:
		return nil
	}
}

func (c *chanController) CloseChannel(chanPoint *wire.OutPoint) er.R {
	return nil
}

// A compile time assertion to ensure chanController meets the
// autopilot.ChannelController interface.
var _ autopilot.ChannelController = (*chanController)(nil)

// initAutoPilot initializes a new autopilot.ManagerCfg to manage an autopilot.
// Agent instance based on the passed configuration structs. The agent and all
// interfaces needed to drive it won't be launched before the Manager's
// StartAgent method is called.
func initAutoPilot(svr *server, cfg *lncfg.AutoPilot,
	chainCfg *lncfg.Chain, netParams chainreg.BitcoinNetParams) (
	*autopilot.ManagerCfg, er.R) {
	log.Infof("Instantiating autopilot with active=%v, "+
		"max_channels=%d, allocation=%f, min_chan_size=%d, "+
		"max_chan_size=%d, private=%t, min_confs=%d, conf_target=%d",
		cfg.Active, cfg.MaxChannels, cfg.Allocation, cfg.MinChannelSize,
		cfg.MaxChannelSize, cfg.Private, cfg.MinConfs, cfg.ConfTarget)

	// Set up the constraints the autopilot heuristics must adhere to.
	atplConstraints := autopilot.NewConstraints(
		btcutil.Amount(cfg.MinChannelSize),
		btcutil.Amount(cfg.MaxChannelSize),
		uint16(cfg.MaxChannels),
		10,
		cfg.Allocation,
	)
	heuristics, err := validateAtplCfg(cfg)
	if err != nil {
		return nil, err
	}

	weightedAttachment, err := autopilot.NewWeightedCombAttachment(
		heuristics...,
	)
	if err != nil {
		return nil, err
	}

	// With the heuristic itself created, we can now populate the remainder
	// of the items that the autopilot agent needs to perform its duties.
	self := svr.identityECDH.PubKey()
	pilotCfg := autopilot.Config{
		Self:      self,
		Heuristic: weightedAttachment,
		ChanController: &chanController{
			server:        svr,
			private:       cfg.Private,
			minConfs:      cfg.MinConfs,
			confTarget:    cfg.ConfTarget,
			chanMinHtlcIn: chainCfg.MinHTLCIn,
			netParams:     netParams,
		},
		WalletBalance: func() (btcutil.Amount, er.R) {
			return svr.cc.Wallet.ConfirmedBalance(cfg.MinConfs)
		},
		Graph:       autopilot.ChannelGraphFromDatabase(svr.localChanDB.ChannelGraph()),
		Constraints: atplConstraints,
		ConnectToPeer: func(target *btcec.PublicKey, addrs []net.Addr) (bool, er.R) {
			// First, we'll check if we're already connected to the
			// target peer. If we are, we can exit early. Otherwise,
			// we'll need to establish a connection.
			if _, err := svr.FindPeer(target); err == nil {
				return true, nil
			}

			// We can't establish a channel if no addresses were
			// provided for the peer.
			if len(addrs) == 0 {
				return false, er.New("no addresses specified")
			}

			log.Tracef("Attempting to connect to %x",
				target.SerializeCompressed())

			lnAddr := &lnwire.NetAddress{
				IdentityKey: target,
				ChainNet:    netParams.Net,
			}

			// We'll attempt to successively connect to each of the
			// advertised IP addresses until we've either exhausted
			// the advertised IP addresses, or have made a
			// connection.
			var connected bool
			for _, addr := range addrs {
				switch addr.(type) {
				case *net.TCPAddr, *tor.OnionAddr:
					lnAddr.Address = addr
				default:
					return false, er.Errorf("unknown "+
						"address type %T", addr)
				}

				err := svr.ConnectToPeer(
					lnAddr, false, svr.cfg.ConnectionTimeout,
				)
				if err != nil {
					// If we weren't able to connect to the
					// peer at this address, then we'll move
					// onto the next.
					continue
				}

				connected = true
				break
			}

			// If we weren't able to establish a connection at all,
			// then we'll error out.
			if !connected {
				return false, er.New("exhausted all " +
					"advertised addresses")
			}

			return false, nil
		},
		DisconnectPeer: svr.DisconnectPeer,
	}

	// Create and return the autopilot.ManagerCfg that administrates this
	// agent-pilot instance.
	return &autopilot.ManagerCfg{
		Self:     self,
		PilotCfg: &pilotCfg,
		ChannelState: func() ([]autopilot.LocalChannel, er.R) {
			// We'll fetch the current state of open
			// channels from the database to use as initial
			// state for the auto-pilot agent.
			activeChannels, err := svr.remoteChanDB.FetchAllChannels()
			if err != nil {
				return nil, err
			}
			chanState := make([]autopilot.LocalChannel,
				len(activeChannels))
			for i, channel := range activeChannels {
				localCommit := channel.LocalCommitment
				balance := localCommit.LocalBalance.ToSatoshis()

				chanState[i] = autopilot.LocalChannel{
					ChanID:  channel.ShortChanID(),
					Balance: balance,
					Node: autopilot.NewNodeID(
						channel.IdentityPub,
					),
				}
			}

			return chanState, nil
		},
		ChannelInfo: func(chanPoint wire.OutPoint) (
			*autopilot.LocalChannel, er.R) {
			channel, err := svr.remoteChanDB.FetchChannel(chanPoint)
			if err != nil {
				return nil, err
			}

			localCommit := channel.LocalCommitment
			return &autopilot.LocalChannel{
				ChanID:  channel.ShortChanID(),
				Balance: localCommit.LocalBalance.ToSatoshis(),
				Node:    autopilot.NewNodeID(channel.IdentityPub),
			}, nil
		},
		SubscribeTransactions: svr.cc.Wallet.SubscribeTransactions,
		SubscribeTopology:     svr.chanRouter.SubscribeTopology,
	}, nil
}

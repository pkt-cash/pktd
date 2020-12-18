package routing

import (
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/lnd/channeldb"
	"github.com/pkt-cash/pktd/lnd/channeldb/kvdb"
	"github.com/pkt-cash/pktd/lnd/lnwire"
	"github.com/pkt-cash/pktd/lnd/routing/route"
)

// routingGraph is an abstract interface that provides information about nodes
// and edges to pathfinding.
type routingGraph interface {
	// forEachNodeChannel calls the callback for every channel of the given node.
	forEachNodeChannel(nodePub route.Vertex,
		cb func(*channeldb.ChannelEdgeInfo, *channeldb.ChannelEdgePolicy,
			*channeldb.ChannelEdgePolicy) er.R) er.R

	// sourceNode returns the source node of the graph.
	sourceNode() route.Vertex

	// fetchNodeFeatures returns the features of the given node.
	fetchNodeFeatures(nodePub route.Vertex) (*lnwire.FeatureVector, er.R)
}

// dbRoutingTx is a routingGraph implementation that retrieves from the
// database.
type dbRoutingTx struct {
	graph  *channeldb.ChannelGraph
	tx     kvdb.RTx
	source route.Vertex
}

// newDbRoutingTx instantiates a new db-connected routing graph. It implictly
// instantiates a new read transaction.
func newDbRoutingTx(graph *channeldb.ChannelGraph) (*dbRoutingTx, er.R) {
	sourceNode, err := graph.SourceNode()
	if err != nil {
		return nil, err
	}

	tx, err := graph.Database().BeginReadTx()
	if err != nil {
		return nil, err
	}

	return &dbRoutingTx{
		graph:  graph,
		tx:     tx,
		source: sourceNode.PubKeyBytes,
	}, nil
}

// close closes the underlying db transaction.
func (g *dbRoutingTx) close() er.R {
	return g.tx.Rollback()
}

// forEachNodeChannel calls the callback for every channel of the given node.
//
// NOTE: Part of the routingGraph interface.
func (g *dbRoutingTx) forEachNodeChannel(nodePub route.Vertex,
	cb func(*channeldb.ChannelEdgeInfo, *channeldb.ChannelEdgePolicy,
		*channeldb.ChannelEdgePolicy) er.R) er.R {
	txCb := func(_ kvdb.RTx, info *channeldb.ChannelEdgeInfo,
		p1, p2 *channeldb.ChannelEdgePolicy) er.R {
		return cb(info, p1, p2)
	}

	return g.graph.ForEachNodeChannel(g.tx, nodePub[:], txCb)
}

// sourceNode returns the source node of the graph.
//
// NOTE: Part of the routingGraph interface.
func (g *dbRoutingTx) sourceNode() route.Vertex {
	return g.source
}

// fetchNodeFeatures returns the features of the given node. If the node is
// unknown, assume no additional features are supported.
//
// NOTE: Part of the routingGraph interface.
func (g *dbRoutingTx) fetchNodeFeatures(nodePub route.Vertex) (
	*lnwire.FeatureVector, er.R) {
	targetNode, err := g.graph.FetchLightningNode(g.tx, nodePub)
	switch {

	// If the node exists and has features, return them directly.
	case err == nil:
		return targetNode.Features, nil

	// If we couldn't find a node announcement, populate a blank feature
	// vector.
	case channeldb.ErrGraphNodeNotFound.Is(err):
		return lnwire.EmptyFeatureVector(), nil

	// Otherwise bubble the error up.
	default:
		return nil, err
	}
}

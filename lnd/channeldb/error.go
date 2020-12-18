package channeldb

import (
	"fmt"

	"github.com/pkt-cash/pktd/btcutil/er"
)

var (
	// Err ...
	Err = er.NewErrorType("lnd.channeldb")
	// ErrNoChanDBExists is returned when a channel bucket hasn't been
	// created.
	ErrNoChanDBExists = Err.CodeWithDetail("ErrNoChanDBExists",
		"channel db has not yet been created")

	// ErrNoHistoricalBucket is returned when the historical channel bucket
	// not been created yet.
	ErrNoHistoricalBucket = Err.CodeWithDetail("ErrNoHistoricalBucket",
		"historical channel bucket has not yet been created")

	// ErrDBReversion is returned when detecting an attempt to revert to a
	// prior database version.
	ErrDBReversion = Err.CodeWithDetail("ErrDBReversion",
		"channel db cannot revert to prior version")

	// ErrLinkNodesNotFound is returned when node info bucket hasn't been
	// created.
	ErrLinkNodesNotFound = Err.CodeWithDetail("ErrLinkNodesNotFound",
		"no link nodes exist")

	// ErrNoActiveChannels  is returned when there is no active (open)
	// channels within the database.
	ErrNoActiveChannels = Err.CodeWithDetail("ErrNoActiveChannels",
		"no active channels exist")

	// ErrNoPastDeltas is returned when the channel delta bucket hasn't been
	// created.
	ErrNoPastDeltas = Err.CodeWithDetail("ErrNoPastDeltas",
		"channel has no recorded deltas")

	// ErrInvoiceNotFound is returned when a targeted invoice can't be
	// found.
	ErrInvoiceNotFound = Err.CodeWithDetail("ErrInvoiceNotFound",
		"unable to locate invoice")

	// ErrNoInvoicesCreated is returned when we don't have invoices in
	// our database to return.
	ErrNoInvoicesCreated = Err.CodeWithDetail("ErrNoInvoicesCreated",
		"there are no existing invoices")

	// ErrDuplicateInvoice is returned when an invoice with the target
	// payment hash already exists.
	ErrDuplicateInvoice = Err.CodeWithDetail("ErrDuplicateInvoice",
		"invoice with payment hash already exists")

	// ErrDuplicatePayAddr is returned when an invoice with the target
	// payment addr already exists.
	ErrDuplicatePayAddr = Err.CodeWithDetail("ErrDuplicatePayAddr",
		"invoice with payemnt addr already exists")

	// ErrInvRefEquivocation is returned when an InvoiceRef targets
	// multiple, distinct invoices.
	ErrInvRefEquivocation = Err.CodeWithDetail("ErrInvRefEquivocation", "inv ref matches multiple invoices")

	// ErrNoPaymentsCreated is returned when bucket of payments hasn't been
	// created.
	ErrNoPaymentsCreated = Err.CodeWithDetail("ErrNoPaymentsCreated",
		"there are no existing payments")

	// ErrNodeNotFound is returned when node bucket exists, but node with
	// specific identity can't be found.
	ErrNodeNotFound = Err.CodeWithDetail("ErrNodeNotFound",
		"link node with target identity not found")

	// ErrChannelNotFound is returned when we attempt to locate a channel
	// for a specific chain, but it is not found.
	ErrChannelNotFound = Err.CodeWithDetail("ErrChannelNotFound",
		"channel not found")

	// ErrMetaNotFound is returned when meta bucket hasn't been
	// created.
	ErrMetaNotFound = Err.CodeWithDetail("ErrMetaNotFound",
		"unable to locate meta information")

	// ErrGraphNotFound is returned when at least one of the components of
	// graph doesn't exist.
	ErrGraphNotFound = Err.CodeWithDetail("ErrGraphNotFound",
		"graph bucket not initialized")

	// ErrGraphNeverPruned is returned when graph was never pruned.
	ErrGraphNeverPruned = Err.CodeWithDetail("ErrGraphNeverPruned",
		"graph never pruned")

	// ErrSourceNodeNotSet is returned if the source node of the graph
	// hasn't been added The source node is the center node within a
	// star-graph.
	ErrSourceNodeNotSet = Err.CodeWithDetail("ErrSourceNodeNotSet",
		"source node does not exist")

	// ErrGraphNodesNotFound is returned in case none of the nodes has
	// been added in graph node bucket.
	ErrGraphNodesNotFound = Err.CodeWithDetail("ErrGraphNodesNotFound",
		"no graph nodes exist")

	// ErrGraphNoEdgesFound is returned in case of none of the channel/edges
	// has been added in graph edge bucket.
	ErrGraphNoEdgesFound = Err.CodeWithDetail("ErrGraphNoEdgesFound",
		"no graph edges exist")

	// ErrGraphNodeNotFound is returned when we're unable to find the target
	// node.
	ErrGraphNodeNotFound = Err.CodeWithDetail("ErrGraphNodeNotFound",
		"unable to find node")

	// ErrEdgeNotFound is returned when an edge for the target chanID
	// can't be found.
	ErrEdgeNotFound = Err.CodeWithDetail("ErrEdgeNotFound",
		"edge not found")

	// ErrZombieEdge is an error returned when we attempt to look up an edge
	// but it is marked as a zombie within the zombie index.
	ErrZombieEdge = Err.CodeWithDetail("ErrZombieEdge", "edge marked as zombie")

	// ErrEdgeAlreadyExist is returned when edge with specific
	// channel id can't be added because it already exist.
	ErrEdgeAlreadyExist = Err.CodeWithDetail("ErrEdgeAlreadyExist",
		"edge already exist")

	// ErrNodeAliasNotFound is returned when alias for node can't be found.
	ErrNodeAliasNotFound = Err.CodeWithDetail("ErrNodeAliasNotFound",
		"alias for node not found")

	// ErrUnknownAddressType is returned when a node's addressType is not
	// an expected value.
	ErrUnknownAddressType = Err.CodeWithDetail("ErrUnknownAddressType",
		"address type cannot be resolved")

	// ErrNoClosedChannels is returned when a node is queries for all the
	// channels it has closed, but it hasn't yet closed any channels.
	ErrNoClosedChannels = Err.CodeWithDetail("ErrNoClosedChannels",
		"no channel have been closed yet")

	// ErrNoForwardingEvents is returned in the case that a query fails due
	// to the log not having any recorded events.
	ErrNoForwardingEvents = Err.CodeWithDetail("ErrNoForwardingEvents",
		"no recorded forwarding events")

	// ErrEdgePolicyOptionalFieldNotFound is an error returned if a channel
	// policy field is not found in the db even though its message flags
	// indicate it should be.
	ErrEdgePolicyOptionalFieldNotFound = Err.CodeWithDetail("ErrEdgePolicyOptionalFieldNotFound",
		"optional field not present")

	// ErrChanAlreadyExists is return when the caller attempts to create a
	// channel with a channel point that is already present in the
	// database.
	ErrChanAlreadyExists = Err.CodeWithDetail("ErrChanAlreadyExists",
		"channel already exists")

	// ErrTooManyExtraOpaqueBytes ...
	ErrTooManyExtraOpaqueBytes = Err.CodeWithDetail("ErrTooManyExtraOpaqueBytes",
		fmt.Sprintf("max allowed number of opaque bytes is %v", MaxAllowedExtraOpaqueBytes))
)

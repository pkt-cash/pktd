package verrpc

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/pktconfig/version"
	"github.com/pkt-cash/pktd/pktlog/log"
	"google.golang.org/grpc"
	"gopkg.in/macaroon-bakery.v2/bakery"
)

const subServerName = "VersionRPC"

var macPermissions = map[string][]bakery.Op{
	"/verrpc.Versioner/GetVersion": {{
		Entity: "info",
		Action: "read",
	}},
}

// Server is an rpc server that supports querying for information about the
// running binary.
type Server struct{}

// Start launches any helper goroutines required for the rpcServer to function.
//
// NOTE: This is part of the lnrpc.SubServer interface.
func (s *Server) Start() er.R {
	return nil
}

// Stop signals any active goroutines for a graceful closure.
//
// NOTE: This is part of the lnrpc.SubServer interface.
func (s *Server) Stop() er.R {
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
	RegisterVersionerServer(grpcServer, s)

	log.Debugf("Versioner RPC server successfully registered with root " +
		"gRPC server")

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
	err := RegisterVersionerHandlerFromEndpoint(ctx, mux, dest, opts)
	if err != nil {
		log.Errorf("Could not register Versioner REST server "+
			"with root REST server: %v", err)
		return er.E(err)
	}

	log.Debugf("Versioner REST server successfully registered with " +
		"root REST server")
	return nil
}

// GetVersion returns information about the compiled binary.
func (s *Server) GetVersion(_ context.Context,
	_ *VersionRequest) (*Version, error) {
	return &Version{
		Version:  version.Version(),
		AppMajor: uint32(version.AppMajorVersion()),
		AppMinor: uint32(version.AppMinorVersion()),
		AppPatch: uint32(version.AppPatchVersion()),
	}, nil
}

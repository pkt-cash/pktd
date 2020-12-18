package lnd

import (
	"context"

	"github.com/pkt-cash/pktd/pktlog/log"
	"google.golang.org/grpc"
)

// errorLogUnaryServerInterceptor is a simple UnaryServerInterceptor that will
// automatically log any errors that occur when serving a client's unary
// request.
func errorLogUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			// TODO(roasbeef): also log request details?
			log.Errorf("[%v]: %v", info.FullMethod, err)
		}

		return resp, err
	}
}

// errorLogStreamServerInterceptor is a simple StreamServerInterceptor that
// will log any errors that occur while processing a client or server streaming
// RPC.
func errorLogStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream,
		info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, ss)
		if err != nil {
			log.Errorf("[%v]: %v", info.FullMethod, err)
		}

		return err
	}
}

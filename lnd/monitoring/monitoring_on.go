// +build monitoring

package monitoring

import (
	"net/http"
	"sync"

	"google.golang.org/grpc"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/lnd/lncfg"
	"github.com/pkt-cash/pktd/pktlog/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var started sync.Once

// GetPromInterceptors returns the set of interceptors for Prometheus
// monitoring.
func GetPromInterceptors() ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		grpc_prometheus.UnaryServerInterceptor,
	}
	streamInterceptors := []grpc.StreamServerInterceptor{
		grpc_prometheus.StreamServerInterceptor,
	}
	return unaryInterceptors, streamInterceptors
}

// ExportPrometheusMetrics sets server options, registers gRPC metrics and
// launches the Prometheus exporter on the specified address.
func ExportPrometheusMetrics(grpcServer *grpc.Server, cfg lncfg.Prometheus) er.R {
	started.Do(func() {
		log.Infof("Prometheus exporter started on %v/metrics", cfg.Listen)

		grpc_prometheus.Register(grpcServer)

		http.Handle("/metrics", promhttp.Handler())
		go func() {
			http.ListenAndServe(cfg.Listen, nil)
		}()
	})

	return nil
}

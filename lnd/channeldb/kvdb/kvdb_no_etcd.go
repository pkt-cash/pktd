// +build !kvdb_etcd

package kvdb

import (
	"context"

	"github.com/pkt-cash/pktd/btcutil/er"
)

// TestBackend is conditionally set to bdb when the kvdb_etcd build tag is
// not defined, allowing testing our database code with bolt backend.
const TestBackend = BoltBackendName

var errEtcdNotAvailable = er.GenericErrorType.CodeWithDetail(
	"errEtcdNotAvailable",
	"etcd backend not available")

// GetEtcdBackend is a stub returning nil and errEtcdNotAvailable error.
func GetEtcdBackend(ctx context.Context, prefix string,
	etcdConfig *EtcdConfig) (Backend, er.R) {
	return nil, errEtcdNotAvailable.Default()
}

// GetTestEtcdBackend  is a stub returning nil, an empty closure and an
// errEtcdNotAvailable error.
func GetEtcdTestBackend(path, name string) (Backend, func(), er.R) {
	return nil, func() {}, errEtcdNotAvailable.Default()
}

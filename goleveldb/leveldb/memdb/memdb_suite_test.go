// +build long_tests

package memdb

import (
	"testing"

	"github.com/pkt-cash/pktd/goleveldb/leveldb/testutil"
)

func TestMemDB(t *testing.T) {
	testutil.RunSuite(t, "MemDB Suite")
}

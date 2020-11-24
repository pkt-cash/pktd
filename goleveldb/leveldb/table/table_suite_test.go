// +build long_tests

package table

import (
	"testing"

	"github.com/pkt-cash/pktd/goleveldb/leveldb/testutil"
)

func TestTable(t *testing.T) {
	testutil.RunSuite(t, "Table Suite")
}

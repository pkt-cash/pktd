// +build long_tests

package leveldb

import (
	"testing"

	"github.com/pkt-cash/pktd/goleveldb/leveldb/testutil"
)

func TestLevelDB(t *testing.T) {
	testutil.RunSuite(t, "LevelDB Suite")
}

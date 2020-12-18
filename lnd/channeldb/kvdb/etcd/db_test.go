// +build kvdb_etcd

package etcd

import (
	"bytes"
	"context"
	"testing"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/btcutil/util"
	"github.com/pkt-cash/pktd/pktwallet/walletdb"
	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	t.Parallel()

	f := NewEtcdTestFixture(t)
	defer f.Cleanup()

	db, err := newEtcdBackend(f.BackendConfig())
	util.RequireNoErr(t, err)

	err = db.Update(func(tx walletdb.ReadWriteTx) er.R {
		// "apple"
		apple, err := tx.CreateTopLevelBucket([]byte("apple"))
		util.RequireNoErr(t, err)
		require.NotNil(t, apple)

		util.RequireNoErr(t, apple.Put([]byte("key"), []byte("val")))
		return nil
	}, func() {})

	// Expect non-zero copy.
	var buf bytes.Buffer

	util.RequireNoErr(t, db.Copy(&buf))
	require.Greater(t, buf.Len(), 0)
	require.Nil(t, err)

	expected := map[string]string{
		bkey("apple"):        bval("apple"),
		vkey("key", "apple"): "val",
	}
	require.Equal(t, expected, f.Dump())
}

func TestAbortContext(t *testing.T) {
	t.Parallel()

	f := NewEtcdTestFixture(t)
	defer f.Cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	config := f.BackendConfig()
	config.Ctx = ctx

	// Pass abort context and abort right away.
	db, err := newEtcdBackend(config)
	util.RequireNoErr(t, err)
	cancel()

	// Expect that the update will fail.
	err = db.Update(func(tx walletdb.ReadWriteTx) er.R {
		_, err := tx.CreateTopLevelBucket([]byte("bucket"))
		util.RequireErr(t, err, "context canceled")

		return nil
	}, func() {})

	util.RequireErr(t, err, "context canceled")

	// No changes in the DB.
	require.Equal(t, map[string]string{}, f.Dump())
}

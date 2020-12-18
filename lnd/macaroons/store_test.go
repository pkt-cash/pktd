package macaroons_test

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/btcutil/util"
	"github.com/pkt-cash/pktd/lnd/channeldb/kvdb"
	"github.com/pkt-cash/pktd/lnd/macaroons"

	"github.com/pkt-cash/pktd/pktwallet/snacl"
	"github.com/stretchr/testify/require"
)

var defaultRootKeyIDContext = macaroons.ContextWithRootKeyID(
	context.Background(), macaroons.DefaultRootKeyID,
)

// newTestStore creates a new bolt DB in a temporary directory and then
// initializes a root key storage for that DB.
func newTestStore(t *testing.T) (string, func(), *macaroons.RootKeyStorage) {
	tempDir, err := ioutil.TempDir("", "macaroonstore-")
	require.NoError(t, err)

	cleanup, store := openTestStore(t, tempDir)
	cleanup2 := func() {
		cleanup()
		_ = os.RemoveAll(tempDir)
	}

	return tempDir, cleanup2, store
}

// openTestStore opens an existing bolt DB and then initializes a root key
// storage for that DB.
func openTestStore(t *testing.T, tempDir string) (func(),
	*macaroons.RootKeyStorage) {
	db, err := kvdb.Create(
		kvdb.BoltBackendName, path.Join(tempDir, "weks.db"), true,
	)
	util.RequireNoErr(t, err)

	store, errr := macaroons.NewRootKeyStorage(db)
	if errr != nil {
		_ = db.Close()
		t.Fatalf("Error creating root key store: %v", errr)
	}

	cleanup := func() {
		_ = store.Close()
	}

	return cleanup, store
}

// TestStore tests the normal use cases of the store like creating, unlocking,
// reading keys and closing it.
func TestStore(t *testing.T) {
	tempDir, cleanup, store := newTestStore(t)
	defer cleanup()

	_, _, errr := store.RootKey(context.TODO())
	require.True(t, macaroons.ErrStoreLocked.Is(er.E(errr)))

	_, errr = store.Get(context.TODO(), nil)
	require.True(t, macaroons.ErrStoreLocked.Is(er.E(errr)))

	pw := []byte("weks")
	err := store.CreateUnlock(&pw)
	util.RequireNoErr(t, err)

	// Check ErrContextRootKeyID is returned when no root key ID found in
	// context.
	_, _, errr = store.RootKey(context.TODO())
	require.True(t, macaroons.ErrContextRootKeyID.Is(er.E(errr)))

	// Check ErrMissingRootKeyID is returned when empty root key ID is used.
	emptyKeyID := make([]byte, 0)
	badCtx := macaroons.ContextWithRootKeyID(context.TODO(), emptyKeyID)
	_, _, errr = store.RootKey(badCtx)
	require.True(t, macaroons.ErrMissingRootKeyID.Is(er.E(errr)))

	// Create a context with illegal root key ID value.
	encryptedKeyID := []byte("enckey")
	badCtx = macaroons.ContextWithRootKeyID(context.TODO(), encryptedKeyID)
	_, _, errr = store.RootKey(badCtx)
	require.True(t, macaroons.ErrKeyValueForbidden.Is(er.E(errr)))

	// Create a context with root key ID value.
	key, id, errr := store.RootKey(defaultRootKeyIDContext)
	require.NoError(t, errr)

	rootID := id
	require.Equal(t, macaroons.DefaultRootKeyID, rootID)

	key2, errr := store.Get(defaultRootKeyIDContext, id)
	require.NoError(t, errr)
	require.Equal(t, key, key2)

	badpw := []byte("badweks")
	err = store.CreateUnlock(&badpw)
	require.True(t, macaroons.ErrAlreadyUnlocked.Is(err))

	_ = store.Close()

	// Between here and the re-opening of the store, it's possible to get
	// a double-close, but that's not such a big deal since the tests will
	// fail anyway in that case.
	_, store = openTestStore(t, tempDir)

	err = store.CreateUnlock(&badpw)
	require.True(t, snacl.ErrInvalidPassword.Is(err))

	err = store.CreateUnlock(nil)
	require.True(t, macaroons.ErrPasswordRequired.Is(err))

	_, _, errr = store.RootKey(defaultRootKeyIDContext)
	require.True(t, macaroons.ErrStoreLocked.Is(er.E(errr)))

	_, errr = store.Get(defaultRootKeyIDContext, nil)
	require.True(t, macaroons.ErrStoreLocked.Is(er.E(errr)))

	err = store.CreateUnlock(&pw)
	util.RequireNoErr(t, err)

	key, errr = store.Get(defaultRootKeyIDContext, rootID)
	require.NoError(t, errr)
	require.Equal(t, key, key2)

	key, id, errr = store.RootKey(defaultRootKeyIDContext)
	require.NoError(t, errr)
	require.Equal(t, key, key2)
	require.Equal(t, rootID, id)
}

// TestStoreGenerateNewRootKey tests that a root key can be replaced with a new
// one in the store without changing the password.
func TestStoreGenerateNewRootKey(t *testing.T) {
	_, cleanup, store := newTestStore(t)
	defer cleanup()

	// The store must be unlocked to replace the root key.
	err := store.GenerateNewRootKey()
	require.True(t, macaroons.ErrStoreLocked.Is(err))

	// Unlock the store and read the current key.
	pw := []byte("weks")
	err = store.CreateUnlock(&pw)
	util.RequireNoErr(t, err)
	oldRootKey, _, errr := store.RootKey(defaultRootKeyIDContext)
	require.NoError(t, errr)

	// Replace the root key with a new random key.
	err = store.GenerateNewRootKey()
	util.RequireNoErr(t, err)

	// Finally, read the root key from the DB and compare it to the one
	// we got returned earlier. This makes sure that the encryption/
	// decryption of the key in the DB worked as expected too.
	newRootKey, _, errr := store.RootKey(defaultRootKeyIDContext)
	require.NoError(t, errr)
	require.NotEqual(t, oldRootKey, newRootKey)
}

// TestStoreChangePassword tests that the password for the store can be changed
// without changing the root key.
func TestStoreChangePassword(t *testing.T) {
	tempDir, cleanup, store := newTestStore(t)
	defer cleanup()

	// The store must be unlocked to replace the root key.
	err := store.ChangePassword(nil, nil)
	require.True(t, macaroons.ErrStoreLocked.Is(err))

	// Unlock the DB and read the current root key. This will need to stay
	// the same after changing the password for the test to succeed.
	pw := []byte("weks")
	err = store.CreateUnlock(&pw)
	util.RequireNoErr(t, err)
	rootKey, _, errr := store.RootKey(defaultRootKeyIDContext)
	require.NoError(t, errr)

	// Both passwords must be set.
	err = store.ChangePassword(nil, nil)
	require.True(t, macaroons.ErrPasswordRequired.Is(err))

	// Make sure that an error is returned if we try to change the password
	// without the correct old password.
	wrongPw := []byte("wrong")
	newPw := []byte("newpassword")
	err = store.ChangePassword(wrongPw, newPw)
	require.True(t, snacl.ErrInvalidPassword.Is(err))

	// Now really do change the password.
	err = store.ChangePassword(pw, newPw)
	util.RequireNoErr(t, err)

	// Close the store. This will close the underlying DB and we need to
	// create a new store instance. Let's make sure we can't use it again
	// after closing.
	err = store.Close()
	util.RequireNoErr(t, err)

	err = store.CreateUnlock(&newPw)
	util.RequireErr(t, err)

	// Let's open it again and try unlocking with the new password.
	_, store = openTestStore(t, tempDir)
	err = store.CreateUnlock(&newPw)
	util.RequireNoErr(t, err)

	// Finally read the root key from the DB using the new password and
	// make sure the root key stayed the same.
	rootKeyDb, _, errr := store.RootKey(defaultRootKeyIDContext)
	require.NoError(t, errr)
	require.Equal(t, rootKey, rootKeyDb)
}

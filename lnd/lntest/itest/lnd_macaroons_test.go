package itest

import (
	"bytes"
	"context"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/btcutil/util"
	"github.com/pkt-cash/pktd/lnd/lnrpc"
	"github.com/pkt-cash/pktd/lnd/lntest"
	"github.com/pkt-cash/pktd/lnd/macaroons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/macaroon.v2"
)

// testMacaroonAuthentication makes sure that if macaroon authentication is
// enabled on the gRPC interface, no requests with missing or invalid
// macaroons are allowed. Further, the specific access rights (read/write,
// entity based) and first-party caveats are tested as well.
func testMacaroonAuthentication(net *lntest.NetworkHarness, ht *harnessTest) {
	var (
		infoReq    = &lnrpc.GetInfoRequest{}
		newAddrReq = &lnrpc.NewAddressRequest{
			Type: AddrTypeWitnessPubkeyHash,
		}
		testNode = net.Alice
	)

	testCases := []struct {
		name string
		run  func(ctxt context.Context, t *testing.T)
	}{{
		// First test: Make sure we get an error if we use no macaroons
		// but try to connect to a node that has macaroon authentication
		// enabled.
		name: "no macaroon",
		run: func(ctxt context.Context, t *testing.T) {
			conn, err := testNode.ConnectRPC(false)
			util.RequireNoErr(t, err)
			defer func() { _ = conn.Close() }()
			client := lnrpc.NewLightningClient(conn)
			_, errr := client.GetInfo(ctxt, infoReq)
			require.Error(t, errr)
			require.Contains(t, errr.Error(), "expected 1 macaroon")
		},
	}, {
		// Second test: Ensure that an invalid macaroon also triggers an
		// error.
		name: "invalid macaroon",
		run: func(ctxt context.Context, t *testing.T) {
			invalidMac, _ := macaroon.New(
				[]byte("dummy_root_key"), []byte("0"), "itest",
				macaroon.LatestVersion,
			)
			cleanup, client := macaroonClient(
				t, testNode, invalidMac,
			)
			defer cleanup()
			_, errr := client.GetInfo(ctxt, infoReq)
			require.Error(t, errr)
			require.Contains(t, errr.Error(), "cannot get macaroon")
		},
	}, {
		// Third test: Try to access a write method with read-only
		// macaroon.
		name: "read only macaroon",
		run: func(ctxt context.Context, t *testing.T) {
			readonlyMac, err := testNode.ReadMacaroon(
				testNode.ReadMacPath(), defaultTimeout,
			)
			util.RequireNoErr(t, err)
			cleanup, client := macaroonClient(
				t, testNode, readonlyMac,
			)
			defer cleanup()
			_, errr := client.NewAddress(ctxt, newAddrReq)
			require.Error(t, errr)
			require.Contains(t, errr.Error(), "permission denied")
		},
	}, {
		// Fourth test: Check first-party caveat with timeout that
		// expired 30 seconds ago.
		name: "expired macaroon",
		run: func(ctxt context.Context, t *testing.T) {
			readonlyMac, err := testNode.ReadMacaroon(
				testNode.ReadMacPath(), defaultTimeout,
			)
			util.RequireNoErr(t, err)
			timeoutMac, err := macaroons.AddConstraints(
				readonlyMac, macaroons.TimeoutConstraint(-30),
			)
			util.RequireNoErr(t, err)
			cleanup, client := macaroonClient(
				t, testNode, timeoutMac,
			)
			defer cleanup()
			_, errr := client.GetInfo(ctxt, infoReq)
			require.Error(t, errr)
			require.Contains(t, errr.Error(), "macaroon has expired")
		},
	}, {
		// Fifth test: Check first-party caveat with invalid IP address.
		name: "invalid IP macaroon",
		run: func(ctxt context.Context, t *testing.T) {
			readonlyMac, err := testNode.ReadMacaroon(
				testNode.ReadMacPath(), defaultTimeout,
			)
			util.RequireNoErr(t, err)
			invalidIPAddrMac, err := macaroons.AddConstraints(
				readonlyMac, macaroons.IPLockConstraint(
					"1.1.1.1",
				),
			)
			util.RequireNoErr(t, err)
			cleanup, client := macaroonClient(
				t, testNode, invalidIPAddrMac,
			)
			defer cleanup()
			_, errr := client.GetInfo(ctxt, infoReq)
			require.Error(t, errr)
			require.Contains(t, errr.Error(), "different IP address")
		},
	}, {
		// Sixth test: Make sure that if we do everything correct and
		// send the admin macaroon with first-party caveats that we can
		// satisfy, we get a correct answer.
		name: "correct macaroon",
		run: func(ctxt context.Context, t *testing.T) {
			adminMac, err := testNode.ReadMacaroon(
				testNode.AdminMacPath(), defaultTimeout,
			)
			util.RequireNoErr(t, err)
			adminMac, err = macaroons.AddConstraints(
				adminMac, macaroons.TimeoutConstraint(30),
				macaroons.IPLockConstraint("127.0.0.1"),
			)
			util.RequireNoErr(t, err)
			cleanup, client := macaroonClient(t, testNode, adminMac)
			defer cleanup()
			res, errr := client.NewAddress(ctxt, newAddrReq)
			require.NoError(t, errr, "get new address")
			assert.Contains(t, res.Address, "bcrt1")
		},
	}, {
		// Seventh test: Bake a macaroon that can only access exactly
		// two RPCs and make sure it works as expected.
		name: "custom URI permissions",
		run: func(ctxt context.Context, t *testing.T) {
			entity := macaroons.PermissionEntityCustomURI
			req := &lnrpc.BakeMacaroonRequest{
				Permissions: []*lnrpc.MacaroonPermission{{
					Entity: entity,
					Action: "/lnrpc.Lightning/GetInfo",
				}, {
					Entity: entity,
					Action: "/lnrpc.Lightning/List" +
						"Permissions",
				}},
			}
			bakeRes, errr := testNode.BakeMacaroon(ctxt, req)
			require.NoError(t, errr)

			// Create a connection that uses the custom macaroon.
			customMacBytes, err := util.DecodeHex(
				bakeRes.Macaroon,
			)
			util.RequireNoErr(t, err)
			customMac := &macaroon.Macaroon{}
			errr = customMac.UnmarshalBinary(customMacBytes)
			require.NoError(t, errr)
			cleanup, client := macaroonClient(
				t, testNode, customMac,
			)
			defer cleanup()

			// Call GetInfo which should succeed.
			_, errr = client.GetInfo(ctxt, infoReq)
			require.NoError(t, errr)

			// Call ListPermissions which should also succeed.
			permReq := &lnrpc.ListPermissionsRequest{}
			permRes, errr := client.ListPermissions(ctxt, permReq)
			require.NoError(t, errr)
			require.Greater(
				t, len(permRes.MethodPermissions), 10,
				"permissions",
			)

			// Try NewAddress which should be denied.
			_, errr = client.NewAddress(ctxt, newAddrReq)
			require.Error(t, errr)
			require.Contains(t, errr.Error(), "permission denied")
		},
	}}

	for _, tc := range testCases {
		tc := tc
		ht.t.Run(tc.name, func(tt *testing.T) {
			ctxt, cancel := context.WithTimeout(
				context.Background(), defaultTimeout,
			)
			defer cancel()

			tc.run(ctxt, tt)
		})
	}
}

// testBakeMacaroon checks that when creating macaroons, the permissions param
// in the request must be set correctly, and the baked macaroon has the intended
// permissions.
func testBakeMacaroon(net *lntest.NetworkHarness, t *harnessTest) {
	testNode := net.Alice

	testCases := []struct {
		name string
		run  func(ctxt context.Context, t *testing.T,
			adminClient lnrpc.LightningClient)
	}{{
		// First test: when the permission list is empty in the request,
		// an error should be returned.
		name: "no permission list",
		run: func(ctxt context.Context, t *testing.T,
			adminClient lnrpc.LightningClient) {
			req := &lnrpc.BakeMacaroonRequest{}
			_, errr := adminClient.BakeMacaroon(ctxt, req)
			require.Error(t, errr)
			assert.Contains(
				t, errr.Error(), "permission list cannot be "+
					"empty",
			)
		},
	}, {
		// Second test: when the action in the permission list is not
		// valid, an error should be returned.
		name: "invalid permission list",
		run: func(ctxt context.Context, t *testing.T,
			adminClient lnrpc.LightningClient) {
			req := &lnrpc.BakeMacaroonRequest{
				Permissions: []*lnrpc.MacaroonPermission{{
					Entity: "macaroon",
					Action: "invalid123",
				}},
			}
			_, errr := adminClient.BakeMacaroon(ctxt, req)
			require.Error(t, errr)
			assert.Contains(
				t, errr.Error(), "invalid permission action",
			)
		},
	}, {
		// Third test: when the entity in the permission list is not
		// valid, an error should be returned.
		name: "invalid permission entity",
		run: func(ctxt context.Context, t *testing.T,
			adminClient lnrpc.LightningClient) {
			req := &lnrpc.BakeMacaroonRequest{
				Permissions: []*lnrpc.MacaroonPermission{{
					Entity: "invalid123",
					Action: "read",
				}},
			}
			_, errr := adminClient.BakeMacaroon(ctxt, req)
			require.Error(t, errr)
			assert.Contains(
				t, errr.Error(), "invalid permission entity",
			)
		},
	}, {
		// Fourth test: check that when no root key ID is specified, the
		// default root keyID is used.
		name: "default root key ID",
		run: func(ctxt context.Context, t *testing.T,
			adminClient lnrpc.LightningClient) {
			req := &lnrpc.BakeMacaroonRequest{
				Permissions: []*lnrpc.MacaroonPermission{{
					Entity: "macaroon",
					Action: "read",
				}},
			}
			_, errr := adminClient.BakeMacaroon(ctxt, req)
			require.NoError(t, errr)

			listReq := &lnrpc.ListMacaroonIDsRequest{}
			resp, errr := adminClient.ListMacaroonIDs(ctxt, listReq)
			require.NoError(t, errr)
			require.Equal(t, resp.RootKeyIds[0], uint64(0))
		},
	}, {
		// Fifth test: create a macaroon use a non-default root key ID.
		name: "custom root key ID",
		run: func(ctxt context.Context, t *testing.T,
			adminClient lnrpc.LightningClient) {
			rootKeyID := uint64(4200)
			req := &lnrpc.BakeMacaroonRequest{
				RootKeyId: rootKeyID,
				Permissions: []*lnrpc.MacaroonPermission{{
					Entity: "macaroon",
					Action: "read",
				}},
			}
			_, errr := adminClient.BakeMacaroon(ctxt, req)
			require.NoError(t, errr)

			listReq := &lnrpc.ListMacaroonIDsRequest{}
			resp, errr := adminClient.ListMacaroonIDs(ctxt, listReq)
			require.NoError(t, errr)

			// the ListMacaroonIDs should give a list of two IDs,
			// the default ID 0, and the newly created ID. The
			// returned response is sorted to guarantee the order so
			// that we can compare them one by one.
			sort.Slice(resp.RootKeyIds, func(i, j int) bool {
				return resp.RootKeyIds[i] < resp.RootKeyIds[j]
			})
			require.Equal(t, resp.RootKeyIds[0], uint64(0))
			require.Equal(t, resp.RootKeyIds[1], rootKeyID)
		},
	}, {
		// Sixth test: check the baked macaroon has the intended
		// permissions. It should succeed in reading, and fail to write
		// a macaroon.
		name: "custom macaroon permissions",
		run: func(ctxt context.Context, t *testing.T,
			adminClient lnrpc.LightningClient) {
			rootKeyID := uint64(4200)
			req := &lnrpc.BakeMacaroonRequest{
				RootKeyId: rootKeyID,
				Permissions: []*lnrpc.MacaroonPermission{{
					Entity: "macaroon",
					Action: "read",
				}},
			}
			bakeResp, errr := adminClient.BakeMacaroon(ctxt, req)
			require.NoError(t, errr)

			newMac, err := readMacaroonFromHex(bakeResp.Macaroon)
			util.RequireNoErr(t, err)
			cleanup, readOnlyClient := macaroonClient(
				t, testNode, newMac,
			)
			defer cleanup()

			// BakeMacaroon requires a write permission, so this
			// call should return an error.
			_, errr = readOnlyClient.BakeMacaroon(ctxt, req)
			require.Error(t, errr)
			require.Contains(t, errr.Error(), "permission denied")

			// ListMacaroon requires a read permission, so this call
			// should succeed.
			listReq := &lnrpc.ListMacaroonIDsRequest{}
			_, errr = readOnlyClient.ListMacaroonIDs(ctxt, listReq)
			require.NoError(t, errr)

			// Current macaroon can only work on entity macaroon, so
			// a GetInfo request will fail.
			infoReq := &lnrpc.GetInfoRequest{}
			_, errr = readOnlyClient.GetInfo(ctxt, infoReq)
			require.Error(t, errr)
			require.Contains(t, errr.Error(), "permission denied")
		},
	}}

	for _, tc := range testCases {
		tc := tc
		t.t.Run(tc.name, func(tt *testing.T) {
			ctxt, cancel := context.WithTimeout(
				context.Background(), defaultTimeout,
			)
			defer cancel()

			adminMac, err := testNode.ReadMacaroon(
				testNode.AdminMacPath(), defaultTimeout,
			)
			util.RequireNoErr(tt, err)
			cleanup, client := macaroonClient(tt, testNode, adminMac)
			defer cleanup()

			tc.run(ctxt, tt, client)
		})
	}
}

// testDeleteMacaroonID checks that when deleting a macaroon ID, it removes the
// specified ID and invalidates all macaroons derived from the key with that ID.
// Also, it checks deleting the reserved marcaroon ID, DefaultRootKeyID or is
// forbidden.
func testDeleteMacaroonID(net *lntest.NetworkHarness, t *harnessTest) {
	var (
		ctxb     = context.Background()
		testNode = net.Alice
	)
	ctxt, cancel := context.WithTimeout(ctxb, defaultTimeout)
	defer cancel()

	// Use admin macaroon to create a connection.
	adminMac, err := testNode.ReadMacaroon(
		testNode.AdminMacPath(), defaultTimeout,
	)
	util.RequireNoErr(t.t, err)
	cleanup, client := macaroonClient(t.t, testNode, adminMac)
	defer cleanup()

	// Record the number of macaroon IDs before creation.
	listReq := &lnrpc.ListMacaroonIDsRequest{}
	listResp, errr := client.ListMacaroonIDs(ctxt, listReq)
	require.NoError(t.t, errr)
	numMacIDs := len(listResp.RootKeyIds)

	// Create macaroons for testing.
	rootKeyIDs := []uint64{1, 2, 3}
	macList := make([]string, 0, len(rootKeyIDs))
	for _, id := range rootKeyIDs {
		req := &lnrpc.BakeMacaroonRequest{
			RootKeyId: id,
			Permissions: []*lnrpc.MacaroonPermission{{
				Entity: "macaroon",
				Action: "read",
			}},
		}
		resp, errr := client.BakeMacaroon(ctxt, req)
		require.NoError(t.t, errr)
		macList = append(macList, resp.Macaroon)
	}

	// Check that the creation is successful.
	listReq = &lnrpc.ListMacaroonIDsRequest{}
	listResp, errr = client.ListMacaroonIDs(ctxt, listReq)
	require.NoError(t.t, errr)

	// The number of macaroon IDs should be increased by len(rootKeyIDs).
	require.Equal(t.t, numMacIDs+len(rootKeyIDs), len(listResp.RootKeyIds))

	// First test: check deleting the DefaultRootKeyID returns an error.
	defaultID, _ := strconv.ParseUint(
		string(macaroons.DefaultRootKeyID), 10, 64,
	)
	req := &lnrpc.DeleteMacaroonIDRequest{
		RootKeyId: defaultID,
	}
	_, errr = client.DeleteMacaroonID(ctxt, req)
	require.Error(t.t, errr)
	require.Contains(
		t.t, errr.Error(), macaroons.ErrDeletionForbidden.Detail,
	)

	// Second test: check deleting the customized ID returns success.
	req = &lnrpc.DeleteMacaroonIDRequest{
		RootKeyId: rootKeyIDs[0],
	}
	resp, errr := client.DeleteMacaroonID(ctxt, req)
	require.NoError(t.t, errr)
	require.True(t.t, resp.Deleted)

	// Check that the deletion is successful.
	listReq = &lnrpc.ListMacaroonIDsRequest{}
	listResp, errr = client.ListMacaroonIDs(ctxt, listReq)
	require.NoError(t.t, errr)

	// The number of macaroon IDs should be decreased by 1.
	require.Equal(t.t, numMacIDs+len(rootKeyIDs)-1, len(listResp.RootKeyIds))

	// Check that the deleted macaroon can no longer access macaroon:read.
	deletedMac, err := readMacaroonFromHex(macList[0])
	util.RequireNoErr(t.t, err)
	cleanup, client = macaroonClient(t.t, testNode, deletedMac)
	defer cleanup()

	// Because the macaroon is deleted, it will be treated as an invalid one.
	listReq = &lnrpc.ListMacaroonIDsRequest{}
	_, errr = client.ListMacaroonIDs(ctxt, listReq)
	require.Error(t.t, errr)
	require.Contains(t.t, errr.Error(), "cannot get macaroon")
}

// testStatelessInit checks that the stateless initialization of the daemon
// does not write any macaroon files to the daemon's file system and returns
// the admin macaroon in the response. It then checks that the password
// change of the wallet can also happen stateless.
func testStatelessInit(net *lntest.NetworkHarness, t *harnessTest) {
	var (
		initPw     = []byte("stateless")
		newPw      = []byte("stateless-new")
		newAddrReq = &lnrpc.NewAddressRequest{
			Type: AddrTypeWitnessPubkeyHash,
		}
	)

	// First, create a new node and request it to initialize stateless.
	// This should return us the binary serialized admin macaroon that we
	// can then use for further calls.
	carol, _, macBytes, err := net.NewNodeWithSeed(
		"Carol", nil, initPw, true,
	)
	util.RequireNoErr(t.t, err)
	if len(macBytes) == 0 {
		t.Fatalf("invalid macaroon returned in stateless init")
	}

	// Now make sure no macaroon files have been created by the node Carol.
	_, errr := os.Stat(carol.AdminMacPath())
	require.Error(t.t, errr)
	_, errr = os.Stat(carol.ReadMacPath())
	require.Error(t.t, errr)
	_, errr = os.Stat(carol.InvoiceMacPath())
	require.Error(t.t, errr)

	// Then check that we can unmarshal the binary serialized macaroon.
	adminMac := &macaroon.Macaroon{}
	errr = adminMac.UnmarshalBinary(macBytes)
	require.NoError(t.t, errr)

	// Find out if we can actually use the macaroon that has been returned
	// to us for a RPC call.
	conn, err := carol.ConnectRPCWithMacaroon(adminMac)
	util.RequireNoErr(t.t, err)
	defer conn.Close()
	adminMacClient := lnrpc.NewLightningClient(conn)
	ctxt, _ := context.WithTimeout(context.Background(), defaultTimeout)
	res, errr := adminMacClient.NewAddress(ctxt, newAddrReq)
	require.NoError(t.t, errr)
	if !strings.HasPrefix(res.Address, harnessNetParams.Bech32HRPSegwit) {
		t.Fatalf("returned address was not a regtest address")
	}

	// As a second part, shut down the node and then try to change the
	// password when we start it up again.
	if err := net.RestartNodeNoUnlock(carol, nil); err != nil {
		t.Fatalf("Node restart failed: %v", err)
	}
	changePwReq := &lnrpc.ChangePasswordRequest{
		CurrentPassword: initPw,
		NewPassword:     newPw,
		StatelessInit:   true,
	}
	ctxb := context.Background()
	response, err := carol.InitChangePassword(ctxb, changePwReq)
	util.RequireNoErr(t.t, err)

	// Again, make  sure no macaroon files have been created by the node
	// Carol.
	_, errr = os.Stat(carol.AdminMacPath())
	require.Error(t.t, errr)
	_, errr = os.Stat(carol.ReadMacPath())
	require.Error(t.t, errr)
	_, errr = os.Stat(carol.InvoiceMacPath())
	require.Error(t.t, errr)

	// Then check that we can unmarshal the new binary serialized macaroon
	// and that it really is a new macaroon.
	if errr = adminMac.UnmarshalBinary(response.AdminMacaroon); err != nil {
		t.Fatalf("unable to unmarshal macaroon: %v", errr)
	}
	if bytes.Equal(response.AdminMacaroon, macBytes) {
		t.Fatalf("expected new macaroon to be different")
	}

	// Finally, find out if we can actually use the new macaroon that has
	// been returned to us for a RPC call.
	conn2, err := carol.ConnectRPCWithMacaroon(adminMac)
	util.RequireNoErr(t.t, err)
	defer conn2.Close()
	adminMacClient = lnrpc.NewLightningClient(conn2)

	// Changing the password takes a while, so we use the default timeout
	// of 30 seconds to wait for the connection to be ready.
	ctxt, _ = context.WithTimeout(context.Background(), defaultTimeout)
	res, errr = adminMacClient.NewAddress(ctxt, newAddrReq)
	require.NoError(t.t, errr)
	if !strings.HasPrefix(res.Address, harnessNetParams.Bech32HRPSegwit) {
		t.Fatalf("returned address was not a regtest address")
	}
}

// readMacaroonFromHex loads a macaroon from a hex string.
func readMacaroonFromHex(macHex string) (*macaroon.Macaroon, er.R) {
	macBytes, err := util.DecodeHex(macHex)
	if err != nil {
		return nil, err
	}

	mac := &macaroon.Macaroon{}
	if errr := mac.UnmarshalBinary(macBytes); errr != nil {
		return nil, er.E(errr)
	}
	return mac, nil
}

func macaroonClient(t *testing.T, testNode *lntest.HarnessNode,
	mac *macaroon.Macaroon) (func(), lnrpc.LightningClient) {
	conn, err := testNode.ConnectRPCWithMacaroon(mac)
	util.RequireNoErr(t, err, "connect to alice")

	cleanup := func() {
		errr := conn.Close()
		require.NoError(t, errr, "close")
	}
	return cleanup, lnrpc.NewLightningClient(conn)
}

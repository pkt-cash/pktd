// Copyright (c) 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcjson_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/btcjson"
)

// TestUsageFlagStringer tests the stringized output for the UsageFlag type.
func TestUsageFlagStringer(t *testing.T) {
	tests := []struct {
		in   btcjson.UsageFlag
		want string
	}{
		{0, "0x0"},
		{btcjson.UFWalletOnly, "UFWalletOnly"},
		{btcjson.UFWebsocketOnly, "UFWebsocketOnly"},
		{btcjson.UFNotification, "UFNotification"},
		{
			btcjson.UFWalletOnly | btcjson.UFWebsocketOnly,
			"UFWalletOnly|UFWebsocketOnly",
		},
		{
			btcjson.UFWalletOnly | btcjson.UFWebsocketOnly | (1 << 31),
			"UFWalletOnly|UFWebsocketOnly|0x80000000",
		},
	}

	// Detect additional usage flags that don't have the stringer added.
	numUsageFlags := 0
	highestUsageFlagBit := btcjson.TstHighestUsageFlagBit
	for highestUsageFlagBit > 1 {
		numUsageFlags++
		highestUsageFlagBit >>= 1
	}
	if len(tests)-3 != numUsageFlags {
		t.Errorf("It appears a usage flag was added without adding " +
			"an associated stringer test")
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}

// TestRegisterCmdErrors ensures the RegisterCmd function returns the expected
// error when provided with invalid types.
func TestRegisterCmdErrors(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		cmdFunc func() interface{}
		flags   btcjson.UsageFlag
		err     er.R
	}{
		{
			name:   "duplicate method",
			method: "getblock",
			cmdFunc: func() interface{} {
				return struct{}{}
			},
			err: btcjson.ErrDuplicateMethod.Default(),
		},
		{
			name:   "invalid usage flags",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				return 0
			},
			flags: btcjson.TstHighestUsageFlagBit,
			err:   btcjson.ErrInvalidUsageFlags.Default(),
		},
		{
			name:   "invalid type",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				return 0
			},
			err: btcjson.ErrInvalidType.Default(),
		},
		{
			name:   "invalid type 2",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				return &[]string{}
			},
			err: btcjson.ErrInvalidType.Default(),
		},
		{
			name:   "embedded field",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				//lint:ignore U1001 this field must be present for the test
				type test struct{ int }
				return (*test)(nil)
			},
			err: btcjson.ErrEmbeddedType.Default(),
		},
		{
			name:   "unexported field",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				//lint:ignore U1001 this field must be present for the test
				type test struct{ a int }
				return (*test)(nil)
			},
			err: btcjson.ErrUnexportedField.Default(),
		},
		{
			name:   "unsupported field type 1",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct{ A **int }
				return (*test)(nil)
			},
			err: btcjson.ErrUnsupportedFieldType.Default(),
		},
		{
			name:   "unsupported field type 2",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct{ A chan int }
				return (*test)(nil)
			},
			err: btcjson.ErrUnsupportedFieldType.Default(),
		},
		{
			name:   "unsupported field type 3",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct{ A complex64 }
				return (*test)(nil)
			},
			err: btcjson.ErrUnsupportedFieldType.Default(),
		},
		{
			name:   "unsupported field type 4",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct{ A complex128 }
				return (*test)(nil)
			},
			err: btcjson.ErrUnsupportedFieldType.Default(),
		},
		{
			name:   "unsupported field type 5",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct{ A func() }
				return (*test)(nil)
			},
			err: btcjson.ErrUnsupportedFieldType.Default(),
		},
		{
			name:   "unsupported field type 6",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct{ A interface{} }
				return (*test)(nil)
			},
			err: btcjson.ErrUnsupportedFieldType.Default(),
		},
		{
			name:   "required after optional",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct {
					A *int
					B int
				}
				return (*test)(nil)
			},
			err: btcjson.ErrNonOptionalField.Default(),
		},
		{
			name:   "non-optional with default",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct {
					A int `jsonrpcdefault:"1"`
				}
				return (*test)(nil)
			},
			err: btcjson.ErrNonOptionalDefault.Default(),
		},
		{
			name:   "mismatched default",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct {
					A *int `jsonrpcdefault:"1.7"`
				}
				return (*test)(nil)
			},
			err: btcjson.ErrMismatchedDefault.Default(),
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		err := btcjson.RegisterCmd(test.method, test.cmdFunc(),
			test.flags)
		if !er.FuzzyEquals(err, test.err) {
			t.Errorf("Test #%d (%s) wrong error - got %T, "+
				"want %T", i, test.name, err, test.err)
			continue
		}
	}
}

// TestMustRegisterCmdPanic ensures the MustRegisterCmd function panics when
// used to register an invalid type.
func TestMustRegisterCmdPanic(t *testing.T) {
	// Setup a defer to catch the expected panic to ensure it actually
	// paniced.
	defer func() {
		if err := recover(); err == nil {
			t.Error("MustRegisterCmd did not panic as expected")
		}
	}()

	// Intentionally try to register an invalid type to force a panic.
	btcjson.MustRegisterCmd("panicme", 0, 0)
}

// TestRegisteredCmdMethods tests the RegisteredCmdMethods function ensure it
// works as expected.
func TestRegisteredCmdMethods(t *testing.T) {
	// Ensure the registered methods are returned.
	methods := btcjson.RegisteredCmdMethods()
	if len(methods) == 0 {
		t.Fatal("RegisteredCmdMethods: no methods")
	}

	// Ensure the returned methods are sorted.
	sortedMethods := make([]string, len(methods))
	copy(sortedMethods, methods)
	sort.Strings(sortedMethods)
	if !reflect.DeepEqual(sortedMethods, methods) {
		t.Fatal("RegisteredCmdMethods: methods are not sorted")
	}
}

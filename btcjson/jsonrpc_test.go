// Copyright (c) 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcjson_test

import (
	"bytes"
	"testing"

	"github.com/pkt-cash/pktd/btcjson"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/pktconfig/version"
)

// TestIsValidIDType ensures the IsValidIDType function behaves as expected.
func TestIsValidIDType(t *testing.T) {
	tests := []struct {
		name    string
		id      interface{}
		isValid bool
	}{
		{"int", int(1), true},
		{"int8", int8(1), true},
		{"int16", int16(1), true},
		{"int32", int32(1), true},
		{"int64", int64(1), true},
		{"uint", uint(1), true},
		{"uint8", uint8(1), true},
		{"uint16", uint16(1), true},
		{"uint32", uint32(1), true},
		{"uint64", uint64(1), true},
		{"string", "1", true},
		{"nil", nil, true},
		{"float32", float32(1), true},
		{"float64", float64(1), true},
		{"bool", true, false},
		{"chan int", make(chan int), false},
		{"complex64", complex64(1), false},
		{"complex128", complex128(1), false},
		{"func", func() {}, false},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		if btcjson.IsValidIDType(test.id) != test.isValid {
			t.Errorf("Test #%d (%s) valid mismatch - got %v, "+
				"want %v", i, test.name, !test.isValid,
				test.isValid)
			continue
		}
	}
}

// TestMarshalResponse ensures the MarshalResponse function works as expected.
func TestMarshalResponse(t *testing.T) {
	testID := 1
	tests := []struct {
		name     string
		result   interface{}
		jsonErr  er.R
		expected []byte
	}{
		{
			name:     "ordinary bool result with no error",
			result:   true,
			jsonErr:  nil,
			expected: []byte(`{"result":true,"error":null,"id":1}`),
		},
		{
			name:   "result with error",
			result: nil,
			jsonErr: func() er.R {
				return btcjson.NewRPCError(btcjson.ErrRPCBlockNotFound, "123 not found", nil)
			}(),
			expected: []byte(`{"result":null,"error":{"code":-5,"message":"` +
				version.Version() + ` ErrRPCBlockNotFound(-5):`),
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		_, _ = i, test
		marshaled, err := btcjson.MarshalResponse(testID, test.result, test.jsonErr)
		if err != nil {
			t.Errorf("Test #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !bytes.Contains(marshaled, test.expected) {
			t.Errorf("Test #%d (%s) mismatched result - got %s, "+
				"want %s", i, test.name, marshaled,
				test.expected)
		}
	}
}

// TestMiscErrors tests a few error conditions not covered elsewhere.
func TestMiscErrors(t *testing.T) {
	// Force an error in NewRequest by giving it a parameter type that is
	// not supported.
	_, err := btcjson.NewRequest(nil, "test", []interface{}{make(chan int)})
	if err == nil {
		t.Error("NewRequest: did not receive error")
		return
	}

	// Force an error in MarshalResponse by giving it an id type that is not
	// supported.
	wantErr := btcjson.ErrInvalidType.Default()
	_, err = btcjson.MarshalResponse(make(chan int), nil, nil)
	if !er.FuzzyEquals(err, wantErr) {
		t.Errorf("MarshalResult: did not receive expected error - got "+
			"%v (%[1]T), want %v (%[2]T)", err, wantErr)
		return
	}

	// Force an error in MarshalResponse by giving it a result type that
	// can't be marshaled.
	/*	_, err = btcjson.MarshalResponse(1, make(chan int), nil)
		if _, ok := er.Wrapped(err).(*json.UnsupportedTypeError); !ok {
			wantErr := &json.UnsupportedTypeError{}
			t.Errorf("MarshalResult: did not receive expected error - got "+
				"%v (%[1]T), want %T", err, wantErr)
			return
		} XXX -trn */
}

// TestRPCError tests the error output for the RPCError type.
func TestRPCError(t *testing.T) {
	tests := []struct {
		in   er.R
		want string
	}{
		{
			btcjson.ErrRPCInvalidRequest.Default(),
			"ErrRPCInvalidRequest(-32600)",
		},
		{
			btcjson.ErrRPCMethodNotFound.Default(),
			"ErrRPCMethodNotFound(-32601)",
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		if test.in.Message() != test.want {
			t.Errorf("Error #%d\n got: %s want: %s", i, test.in.Message(),
				test.want)
			continue
		}
	}
}

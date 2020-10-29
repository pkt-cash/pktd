// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package globalcfg

import (
	"math"

	"github.com/pkt-cash/pktd/btcutil/er"
)

// Copyright (c) 2013, 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.


// round converts a floating point number, which may or may not be representable
// as an integer, to the Amount integer type by rounding to the nearest integer.
// This is performed by adding or subtracting 0.5 depending on the sign, and
// relying on integer truncation to round the value to the nearest Amount.
func round(f float64) int64 {
	if f < 0 {
		return int64(f - 0.5)
	}
	return int64(f + 0.5)
}

// NewAmount creates an Amount from a floating point value representing
// some value in PKT.  NewAmount errors if f is NaN or +-Infinity, but
// does not check that the amount is within the total amount of PKT Cash
// producible as f may not refer to an amount at a single moment in time.
//
// NewAmount is for specifically for converting PKT to Units.
// For creating a new Amount with an int64 value which denotes a quantity
// of Units, do a simple type conversion from type int64 to Amount.
func NewAmount(f float64) (int64, er.R) {
	// The amount is only considered invalid if it cannot be represented
	// as an integer type.  This may happen if f is NaN or +-Infinity.
	switch {
	case math.IsNaN(f):
		fallthrough
	case math.IsInf(f, 1):
		fallthrough
	case math.IsInf(f, -1):
		return 0, er.New("invalid bitcoin amount")
	}

	return round(f * float64(UnitsPerCoin())), nil
}

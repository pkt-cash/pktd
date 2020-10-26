// Copyright (c) 2013, 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcutil

import (
	"math"
	"strconv"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/chaincfg/globalcfg"
)

// Amount represents the base PKT monetary unit (colloquially referred
// to as a `Satoshi').  A single Amount is equal to 1e-8 of a PKT.
type Amount int64

// round converts a floating point number, which may or may not be representable
// as an integer, to the Amount integer type by rounding to the nearest integer.
// This is performed by adding or subtracting 0.5 depending on the sign, and
// relying on integer truncation to round the value to the nearest Amount.
func round(f float64) Amount {
	if f < 0 {
		return Amount(f - 0.5)
	}
	return Amount(f + 0.5)
}

// NewAmount creates an Amount from a floating point value representing
// some value in PKT.  NewAmount errors if f is NaN or +-Infinity, but
// does not check that the amount is within the total amount of PKT Cash
// producible as f may not refer to an amount at a single moment in time.
//
// NewAmount is for specifically for converting PKT to Satoshi.
// For creating a new Amount with an int64 value which denotes a quantity of Satoshi,
// do a simple type conversion from type int64 to Amount.
func NewAmount(f float64) (Amount, er.R) {
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

	return round(f * float64(globalcfg.SatoshiPerPKT())), nil
}

// ToUnit converts a monetary amount counted in PKT base units to a
// floating point value representing an amount of PKT.
func (a Amount) ToUnit(uname string) (float64, er.R) {
	units := globalcfg.AmountUnits()
	for _, u := range units {
		if u.Name != uname {
			continue
		}
		return float64(a) / float64(u.Units), nil
	}
	return math.NaN(), er.Errorf("%s is not a valid unit", uname)
}

// ToPKT is the equivalent of calling ToUnit with AmountPKT.
func (a Amount) ToPKT() float64 {
	out, err := a.ToUnit(globalcfg.AmountUnits()[0].Name)
	if err != nil {
		panic("ToUnit failed with default unit")
	}
	return out
}

// Format formats a monetary amount counted in PKT base units as a
// string for a given unit.  The conversion will succeed for any unit,
// however, known units will be formated with an appended label describing
// the units with SI notation, or "Satoshi" for the base unit.
func (a Amount) Format(uname string) (string, er.R) {
	units := globalcfg.AmountUnits()
	for _, u := range units {
		if u.Name != uname {
			continue
		}
		res := float64(a) / float64(u.Units)
		n := u.Name
		if u.ProperName != "" {
			n = u.ProperName
		}
		units := " " + n
		return strconv.FormatFloat(res, 'f', -u.Zeros, 64) + units, nil
	}
	return "", er.Errorf("%s is not a valid unit", uname)
}

// String is the equivalent of calling Format with AmountPKT.
func (a Amount) String() string {
	out, err := a.Format(globalcfg.AmountUnits()[0].Name)
	if err != nil {
		panic("Format failed with default unit")
	}
	return out
}

// MulF64 multiplies an Amount by a floating point value.  While this is not
// an operation that must typically be done by a full node or wallet, it is
// useful for services that build on top of PKT (for example, calculating
// a fee by multiplying by a percentage).
func (a Amount) MulF64(f float64) Amount {
	return round(float64(a) * f)
}

// MaxUnits returns the maximum number of atomic units of currency
func MaxUnits() Amount {
	return Amount(globalcfg.MaxUnitsI64())
}

// UnitsPerCoin returns the maximum number of atomic units per "coin"
func UnitsPerCoin() Amount {
	return Amount(globalcfg.UnitsPerCoinI64())
}

// Copyright 2020 Jeffrey H. Johnson.
// Copyright 2020 Gridfinity, LLC.
// Copyright 2019 The Go Authors.
// All rights reserved.
// Use of this source code is governed by the BSD-style
// license that can be found in the LICENSE file.

package cryptocycle_test

import (
	"fmt"
	mrand "math/rand"
	"testing"
	"time"

	goc25519sm "go.gridfinity.dev/goc25519sm"
)

func benchmarkOldScalarBaseMult(
	x int,
	b *testing.B,
) {
	var in, out [goc25519sm.X25519Size]byte
	for bSetup := 0; bSetup < 32; bSetup = (bSetup + 2) {
		in[bSetup] = ((byte(bSetup) + 1) + byte(x))
		in[bSetup+1] = (in[bSetup] + byte(x))
	}
	var err error
	b.SetBytes(
		goc25519sm.X25519Size,
	)
	for i := 0; i < b.N; i++ {
		err = goc25519sm.OldScalarBaseMult(
			&out,
			&in,
		)
		if err != nil {
			b.Fatal(
				fmt.Sprintf(
					"\ngoc25519sm_test.benchmarkOldScalarBaseMult.OldScalarBaseMult FAILURE:\n	input=%v\n	output=%v\n	%v",
					in,
					out,
					err,
				),
			)
		}
	}
	// Overwrite ExamplePointA with the bench output, and invoke
	// OldScalarVerifyBasepoint to ensure it correctly detects
	// that this output is NOT the Basepoint; this is mostly to
	// ensure the benchmark is not aggressively optimized away
	// by performing actual (constant time) work on the output.
	goc25519sm.ExamplePointA = out
	err = goc25519sm.OldScalarVerifyBasepoint(
		goc25519sm.ExamplePointA,
	)
	if err == nil {
		b.Fatal(
			fmt.Sprintf(
				"\ngoc25519sm_test.benchmarkOldScalarBaseMult.OldScalarVerifyBasepoint FAILURE:\n	ERROR: false positive detected checking basepoint: %v",
				err,
			),
		)
	}
}

// Setup multiple iterations with randomized inputs. Use
// of the CSPRNG is not needed here for simple benchmark
// testing, but should always be used in production code.
func BenchmarkOldScalarBaseMult_01(
	b *testing.B,
) {
	mrand.Seed(
		time.Now().UnixNano(),
	)
	z := mrand.Intn(
		((((1 << 8) - 1 - 1) - 1) + 1),
	)
	benchmarkOldScalarBaseMult(
		(z + 1),
		b,
	)
}

func BenchmarkOldScalarBaseMult_02(
	b *testing.B,
) {
	mrand.Seed(
		time.Now().UnixNano(),
	)
	z := mrand.Intn(
		((((1 << 8) - 2 - 1) - 2) + 2),
	)
	benchmarkOldScalarBaseMult(
		z+2,
		b,
	)
}

func BenchmarkOldScalarBaseMult_04(
	b *testing.B,
) {
	mrand.Seed(
		time.Now().UnixNano(),
	)
	z := mrand.Intn(
		((((1 << 8) - 4 - 1) - 4) + 4),
	)
	benchmarkOldScalarBaseMult(
		z+4,
		b,
	)
}

func BenchmarkOldScalarBaseMult_08(
	b *testing.B,
) {
	mrand.Seed(
		time.Now().UnixNano(),
	)
	z := mrand.Intn(
		((((1 << 8) - 8 - 1) - 8) + 8),
	)
	benchmarkOldScalarBaseMult(
		(z + 8),
		b,
	)
}

func BenchmarkOldScalarBaseMult_16(
	b *testing.B,
) {
	mrand.Seed(
		time.Now().UnixNano(),
	)
	z := mrand.Intn(
		((((1 << 8) - 16 - 1) - 16) + 16),
	)
	benchmarkOldScalarBaseMult(
		(z + 16),
		b,
	)
}

func BenchmarkOldScalarBaseMult_32(
	b *testing.B,
) {
	mrand.Seed(
		time.Now().UnixNano(),
	)
	z := mrand.Intn(
		((((1 << 8) - 32 - 1) - 32) + 32),
	)
	benchmarkOldScalarBaseMult(
		(z + 32),
		b,
	)
}

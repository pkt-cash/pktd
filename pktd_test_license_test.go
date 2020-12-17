// Copyright 2020 Jeffrey H. Johnson.
// Copyright 2020 Gridfinity, LLC.
// Copyright 2012 The Go Authors.
// All rights reserved.
// Use of this source code is governed by the BSD-style
// license that can be found in the LICENSE file.

package main_test

import (
	"fmt"
	"testing"

	u "github.com/pkt-cash/pktd/blockchain/packetcrypt/cryptocycle/testutil"
	licn "go4.org/legal"
)

func TestLicense(
	t *testing.T,
) {
	defer u.LeakVerifyNone(
		t,
	)
	licenses := licn.Licenses()
	if len(
		licenses,
	) == 0 {
		t.Fatal(
			"\npktd_test_license_test.TestLicense.Licenses FAILURE",
		)
	} else {
		t.Log(
			fmt.Sprintf(
				"\n%v\n",
				licenses,
			),
		)
	}
}

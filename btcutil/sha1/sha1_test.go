package sha1

import (
	gosha1 "crypto/sha1"
	csubtle "crypto/subtle"
	x0 "encoding/hex"
	stest "testing"
	tq "testing/quick"

	gl "go.uber.org/goleak"
)

//go:generate go run asm.go -out sha1.s -stubs stub.go

func TestASHA1Conform(
	t *stest.T,
) {
	defer gl.VerifyNone(
		t,
	)
	cases := []struct {
		Data      string
		HexDigest string
	}{
		{
			"",
			"da39a3ee5e6b4b0d3255bfef95601890afd80709",
		},
		{
			"The quick brown fox jumps over the lazy dog",
			"2fd4e1c67a2d28fced849ee1bb76e7391b93eb12",
		},
		{
			"The quick brown fox jumps over the lazy cog",
			"de9f2c7fd25e1b3afad3e85a0bd17d9b100db4b3",
		},
	}
	for _, c := range cases {
		digest := Sum(
			[]byte(
				c.Data,
			),
		)
		got := x0.EncodeToString(
			digest[:],
		)
		if got != c.HexDigest {
			t.Errorf(
				"sha1 failure: Sum(%#v) = %s; expect %s",
				c.Data,
				got,
				c.HexDigest,
			)
		}
	}
}

func TestASHA1Compare(
	t *stest.T,
) {
	defer gl.VerifyNone(
		t,
	)
	if err := tq.CheckEqual(
		Sum,
		gosha1.Sum,
		nil,
	); err != nil {
		t.Fatal(
			err,
		)
	}
}

func TestASHA1Length(
	t *stest.T,
) {
	defer gl.VerifyNone(
		t,
	)
	data := make(
		[]byte,
		blocksize,
	)
	for n := 0; n <= blocksize; n++ {
		got := Sum(
			data[:n],
		)
		expect := gosha1.Sum(
			data[:n],
		)
		if csubtle.ConstantTimeCompare(
			got[:],
			expect[:],
		) != 1 {
			t.Errorf(
				"sha1 fail len=%d",
				n,
			)
		}
	}
}

// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package cryptocycle

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/aead/chacha20/chacha"

	goc25519sm "github.com/johnsonjh/goc25519sm"
	"golang.org/x/crypto/poly1305"

	"github.com/pkt-cash/pktd/blockchain/packetcrypt/randhash/interpret"

	"github.com/pkt-cash/pktd/blockchain/packetcrypt/pcutil"
	"github.com/pkt-cash/pktd/blockchain/packetcrypt/randhash/randgen"
)

type State struct {
	Bytes [2048]byte
}

func (s *State) Nonce() []byte { return s.Bytes[:12] }
func (s *State) Data() []byte  { return s.Bytes[12:16] }
func (s *State) Key() []byte   { return s.Bytes[16:48] }
func (s *State) Poly() []byte  { return s.Bytes[16:32] }

func (s *State) SetBitRange(offset, width, _v byte) {
	v := uint32(_v) & ((1 << width) - 1)
	x := binary.LittleEndian.Uint32(s.Data())
	x &^= (((1 << width) - 1) << offset)
	x |= (v << offset)
	binary.LittleEndian.PutUint32(s.Data(), x)
}

func (s *State) GetBitRange(offset, width byte) byte {
	return byte(binary.LittleEndian.Uint32(s.Data()) >> offset & ((1 << width) - 1))
}

func (s *State) SetVersion(v byte)         { s.SetBitRange(25, 7, v) }
func (s *State) SetFailed(v bool)          { s.SetBitRange(24, 1, byte(pcutil.Bint(v))) }
func (s *State) SetLength(v byte)          { s.SetBitRange(17, 7, v) }
func (s *State) SetTruncated(v bool)       { s.SetBitRange(16, 1, byte(pcutil.Bint(v))) }
func (s *State) SetAddLen(v byte)          { s.SetBitRange(13, 3, v) }
func (s *State) SetDecrypt(v bool)         { s.SetBitRange(12, 1, byte(pcutil.Bint(v))) }
func (s *State) SetTrailingZeros(v byte)   { s.SetBitRange(8, 4, v) }
func (s *State) SetAdditionalZeros(v byte) { s.SetBitRange(0, 4, v) }

func (s *State) GetVersion() byte         { return s.GetBitRange(25, 7) }
func (s *State) IsFailed() bool           { return s.GetBitRange(24, 1) != 0 }
func (s *State) GetLength() byte          { return s.GetBitRange(17, 7) }
func (s *State) GetAddLen() byte          { return s.GetBitRange(13, 3) }
func (s *State) IsDecrypt() bool          { return s.GetBitRange(12, 1) != 0 }
func (s *State) GetTrailingZeros() byte   { return s.GetBitRange(8, 4) }
func (s *State) GetAdditionalZeros() byte { return s.GetBitRange(0, 4) }

// func (s *State) IsTruncated() bool        { return s.GetBitRange(16, 1) != 0 }

func GetItemNo(state *State) uint64 {
	return binary.LittleEndian.Uint64(state.Bytes[16:24])
}

func (s *State) MakeFuzzable() {
	copy(s.Data(), s.Poly()[:4])
	s.SetVersion(0)
	s.SetFailed(false)
	s.SetLength(s.GetLength() | 32)
}

type Context struct {
	Progbuf [2048]uint32
}

const (
	hdrSz         int = 48
	authTrailerSz int = 16
)

func getLengthAndTruncate(s *State) int {
	l := int(s.GetLength())
	max := 125 - int(s.GetAddLen())
	finalLen := l
	if l > max {
		finalLen = max
	}
	s.SetTruncated(finalLen != l)
	s.SetLength(byte(finalLen))
	return finalLen
}

func CryptoCycle(s *State) {
	if s.GetVersion() != 0 || s.IsFailed() {
		s.SetFailed(true)
		return
	}
	c, err := chacha.NewCipher(s.Nonce(), s.Key(), 20)
	if err != nil {
		panic("chacha20.NewCipher()")
	}
	var polyKey [32]byte
	c.XORKeyStream(polyKey[:], polyKey[:])

	c.SetCounter(1)

	aeadLen := int(s.GetAddLen()) * 16
	msgLen := getLengthAndTruncate(s) * 16
	tzc := int(s.GetTrailingZeros())
	azc := int(s.GetAdditionalZeros())
	decrypt := s.IsDecrypt()

	toVerify := make([]byte, aeadLen+msgLen+authTrailerSz)
	copy(toVerify[:aeadLen], s.Bytes[hdrSz:hdrSz+aeadLen])

	paddedContent := s.Bytes[hdrSz+aeadLen : hdrSz+aeadLen+msgLen]
	content := s.Bytes[hdrSz+aeadLen : hdrSz+aeadLen+msgLen-tzc]
	if decrypt {
		// If there are non-zero trailing bytes, include them in the auth
		copy(toVerify[aeadLen:], paddedContent)
	}

	c.XORKeyStream(paddedContent, paddedContent)

	if !decrypt {
		// for a fast implementation, it should be easier to zero the bits at the end
		// after encryption and before authentication rather than making the encryption
		// or authentication aware that it needs to do a partial block.
		pcutil.Zero(s.Bytes[hdrSz+aeadLen+msgLen-tzc : hdrSz+aeadLen+msgLen])

		copy(toVerify[aeadLen:], content)
	}

	binary.LittleEndian.PutUint64(toVerify[aeadLen+msgLen:][:8], uint64(aeadLen)-uint64(azc))
	binary.LittleEndian.PutUint64(toVerify[aeadLen+msgLen+8:], uint64(msgLen-tzc))

	var tag [poly1305.TagSize]byte
	poly1305.Sum(&tag, toVerify, &polyKey)
	copy(s.Poly(), tag[:])
}

func Init(s *State, seed []byte, nonce uint64) {
	pcutil.HashExpand(s.Bytes[:], seed, 0)
	binary.LittleEndian.PutUint64(s.Nonce(), nonce)
	s.MakeFuzzable()
}

func Update(state *State, item, contentBlock []byte, randHashCycles int, progBuf *Context) bool {
	if randHashCycles > 0 {
		prog, err := randgen.Generate(item[32*31:])
		if err != nil {
			return false
		}
		if interpret.Interpret(prog, state.Bytes[:], item, randHashCycles) != nil {
			return false
		}
	}
	copy(state.Bytes[32:], item)
	if contentBlock != nil {
		copy(state.Bytes[32+1024:], contentBlock)
	}
	state.MakeFuzzable()
	CryptoCycle(state)
	if state.IsFailed() {
		panic("CryptoCycle went into a failed state, should not happen")
	}
	// fmt.Printf("%v.\n", hex.EncodeToString(state.Bytes[0:64]))
	return true
}

// Smul does a scalar mult cycle
func Smul(s *State) {
	// while this does not overcome secret-dependent branch and memory access
	// timing-attack possibilities, it does add a bit more noise that will
	// make a localized or colocated adversary do more work when executing a
	// predictive timing attack, for additional background on the subject:
	// https://fahrplan.events.ccc.de/congress/2012/Fahrplan/attachments/2235_29c3-schinzel.pdf
	// crypto.stanford.edu/~dabo/papers/ssl-timing.pdf ... etc.
	time.Sleep(10 * time.Nanosecond)
	var err error
	var a, b, c [32]byte
	copy(a[:], s.Bytes[32:][:32])
	err = goc25519sm.OldScalarBaseMult(
		&b,
		&a,
	)
	if err != nil {
		panic(
			fmt.Sprintf(
				"\nCryptoCycle.goc25519sm.OldScalarBaseMult failure:\n	%v",
				err,
			),
		)
	}
	copy(a[:], s.Bytes[:32])
	err = goc25519sm.OldScalarMult(
		&c,
		&a,
		&b,
	)
	if err != nil {
		panic(
			fmt.Sprintf(
				"\nCryptoCycle.goc25519sm.OldScalarMult failure:\n	%v",
				err,
			),
		)
	}
	copy(s.Bytes[64:][:32], c[:])
}

/*
void CryptoCycle_smul(CryptoCycle_State_t* restrict state) {
    uint8_t pubkey[crypto_scalarmult_curve25519_BYTES];
    assert(!crypto_scalarmult_curve25519_base(pubkey, state->thirtytwos[1].bytes));
    assert(!crypto_scalarmult_curve25519(
        state->thirtytwos[2].bytes, state->thirtytwos[0].bytes, pubkey));
}*/

// Final will hash the whole buffer and put the result at the beginning
func Final(s *State) {
	pcutil.HashCompress(s.Bytes[:], s.Bytes[:])
}

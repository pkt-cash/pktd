// Copyright (c) 2020 Anode LLC
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package seedwords

import (
	"crypto/rand"
	"encoding/binary"
	"math/big"
	"strings"
	"time"

	"github.com/dchest/blake2b"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/pktwallet/internal/zero"
	"golang.org/x/crypto/argon2"
)

type wordsDesc struct {
	words  [2048]string
	rwords map[string]int16
	lang   string
}

//go:generate go run -tags words genwords.go
var allWords = make(map[string]*wordsDesc)

/**
 * Seed layout:
 *     0               1               2               3
 *     0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
 *    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 *  0 |  U  |  Ver  |E|   Checksum    |           Birthday            |
 *    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 *  4 |                                                               |
 *    +                                                               +
 *  8 |                                                               |
 *    +                               Seed                            +
 * 12 |                                                               |
 *    +                                                               +
 * 16 |                                                               |
 *    +               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 * 20 |               |
 *    +-+-+-+-+-+-+-+-+
 *
 * U: unused
 * Ver: 0
 * E: 1 if there is a passphrase encrypting the seed, 0 otherwise
 * Checksum: first byte of blake2b of structure with Checksum and Unused cleared
 * Birthday (encrypted): when the wallet was created, unix time divided by 60*60*24, big endian
 * Seed (encrypted): 32 byte seed content
 */
const seedVersion = 0
const wordCount = 15
const encByteLen = 21

// The salt is fixed because:
// 1. The password should normally be a strong one
// 2. Wallet seeds are something one is unlikely to encounter in large quantity
// 3. The resulting seed must be compact
var argonSalt = []byte("pktwallet seed 0")

const argonIterations = 32
const argonMemory = 256 * 1024 // 256MB
const argonThreads = 8

// The value of the unused section should be 1 after decoding
const expectUnused = 1

type SeedEnc struct {
	Bytes [encByteLen]byte
}

/*func (s *SeedEnc) putUnused(u byte) {
	s.Bytes[0] &= 0x1f
	s.Bytes[0] |= ((u & 0x07) << 5)
}*/
func (s *SeedEnc) getUnused() byte {
	return s.Bytes[0] >> 5
}

func (s *SeedEnc) putBday(bday time.Time) {
	day := uint16(bday.Unix() / (60 * 60 * 24))
	binary.BigEndian.PutUint16(s.Bytes[2:4], day)
}
func (s *SeedEnc) getBday() time.Time {
	day := binary.BigEndian.Uint16(s.Bytes[2:4])
	return time.Unix(int64(day)*(60*60*24), 0)
}

func (s *SeedEnc) putVer(ver byte) {
	s.Bytes[0] = (s.Bytes[0] & 0x01) | ((ver & 0x0f) << 1)
}
func (s *SeedEnc) getVer() byte {
	return (s.Bytes[0] >> 1) & 0x0f
}

func (s *SeedEnc) putE(e bool) {
	s.Bytes[0] &= 0x1e
	if e {
		s.Bytes[0] |= 0x01
	}
}
func (s *SeedEnc) getE() bool {
	return s.Bytes[0]&0x01 == 0x01
}

func (s *SeedEnc) putCsum(csum byte) {
	s.Bytes[1] = csum
}
func (s *SeedEnc) getCsum() byte {
	return s.Bytes[1]
}

/*func (s *SeedEnc) putSeed(seed *[17]byte) {
	copy(s.Bytes[4:], seed[:])
}
func (s *SeedEnc) getSeed(seedOut *[17]byte) {
	copy(seedOut[:], s.Bytes[4:])
}*/

func (s *SeedEnc) computeCsum() byte {
	// Pull the csum
	sb0 := s.getCsum()
	s.putCsum(0)

	// Clear the unused bits
	s.Bytes[0] &= 0x1f

	csum := blake2b.Sum256(s.Bytes[:])

	// put back the csum if there was one
	s.putCsum(sb0)

	return csum[0]
}

// If the passphrase is emptystring then cipher() simply copies the content
// from one to the other.
func cipher(outBytes *[encByteLen]byte, inBytes *[encByteLen]byte, passphrase []byte) {
	if len(passphrase) == 0 {
		copy(outBytes[2:], inBytes[2:])
	}
	// bytes 0 and 1 are not enciphered
	key := argon2.IDKey(
		passphrase,
		argonSalt,
		argonIterations,
		argonMemory,
		argonThreads,
		encByteLen-2)
	for i, b := range key {
		outBytes[i+2] = inBytes[i+2] ^ b
	}
	zero.Bytes(key)
}

func (s *SeedEnc) toBig(b *big.Int) {
	b.SetBytes(s.Bytes[:])
}

/*func (s *SeedEnc) fromBig(b *big.Int) {
	b.Lsh(b, 16)
	bytes := b.Bytes()
	copy(s.Bytes[:], bytes[:])
	zero.Bytes(bytes)
}*/

func zeroStr(s []string) {
	x := s[0:cap(s)]
	for i := range x {
		x[i] = ""
	}
}

func zeroNums(n []int16) {
	x := n[0:cap(n)]
	for i := range x {
		x[i] = 0
	}
}

// Unencrypted seed, used for deriving keys, this should be cleared when
// not needed using seed.Zero().
type Seed struct {
	// Contains a decrypted SeedEnc, the first two bytes are undefined
	seedBin SeedEnc
}

// Generate a random seed with a birthday of right now
func RandomSeed() (*Seed, er.R) {
	out := Seed{}
	_, err := rand.Read(out.seedBin.Bytes[4:])
	if err != nil {
		return nil, er.E(err)
	}
	out.seedBin.putBday(time.Now())
	return &out, nil
}

// Zero wipes the data of this seed from memory.
func (s *Seed) Zero() {
	s.seedBin.Zero()
}

// Provide the day when the seed was first created, useful for determining
// how far back in time you must search for possible payments to the wallet.
func (s *Seed) Birthday() time.Time {
	return s.seedBin.getBday()
}

// Bytes returns the secret seed, 19 bytes long including the birthday bytes
// which provide a small amount of additional entropy.
func (s *Seed) Bytes() []byte {
	return s.seedBin.Bytes[2:]
}

// Encrypt converts a Seed to a SeedEnc using a passphrase, if the passphrase
// is the empty string then the SeedEnc will be flagged as not requiring a
// passphrase to decrypt.
func (s *Seed) Encrypt(passphrase []byte) *SeedEnc {
	out := SeedEnc{}
	cipher(&out.Bytes, &s.seedBin.Bytes, passphrase)
	out.putVer(seedVersion)
	out.putE(len(passphrase) > 0)
	out.putCsum(out.computeCsum())
	return &out
}

// Words converts the encrypted seed to it's representation as a list of words.
func (s *SeedEnc) Words(lang string) (string, er.R) {
	wd, ok := allWords[lang]
	if !ok {
		return "", er.Errorf("Language [%s] is not supported", lang)
	}
	words := make([]string, 0, wordCount)
	defer zeroStr(words)
	b := big.NewInt(0)
	defer zero.BigInt(b)
	b_ := big.NewInt(0)
	defer zero.BigInt(b_)
	b2047 := big.NewInt(2047)

	// set unused to 0b001 to create a guard bit for bigint
	s.Bytes[0] &= 0x1f
	s.Bytes[0] |= 0x20
	s.toBig(b)
	for i := 0; i < wordCount; i++ {
		b_.And(b, b2047)
		words = append(words, wd.words[b_.Uint64()])
		b.Rsh(b, 11)
	}
	if !b.IsUint64() || b.Uint64() != 1 {
		panic("Internal error: bignum should have resulted in 1")
	}
	return strings.Join(words, " "), nil
}

func fromNums(nums [wordCount]int16) (*SeedEnc, er.R) {
	b := big.NewInt(1)
	defer zero.BigInt(b)
	b_ := big.NewInt(0)
	defer zero.BigInt(b_)
	for i := len(nums) - 1; i >= 0; i-- {
		b_.SetInt64(int64(nums[i]))
		b.Lsh(b, 11)
		b.Add(b, b_)
	}
	bytes := b.Bytes()
	defer zero.Bytes(bytes)
	if len(bytes) != encByteLen {
		return nil, er.New("Invalid seed: Unexpected byte length")
	}
	s := SeedEnc{}
	copy(s.Bytes[:], bytes)
	var err er.R
	if s.getUnused() != expectUnused {
		err = er.New("Invalid seed: Wrong bit pattern")
	} else if s.getVer() != 0 {
		err = er.Errorf("Invalid seed: Unknown version [%d]", s.getVer())
	} else if s.getCsum() != s.computeCsum() {
		err = er.New("Invalid seed: Checksum mismatch")
	} else {
		return &s, nil
	}
	s.Zero()
	return nil, err
}

// SeedFromWords creates an encrypted seed from a set of words, the language
// is auto-detected.
func SeedFromWords(words string) (*SeedEnc, er.R) {
	splitWords := strings.Split(words, " ")
	defer zeroStr(splitWords)
	if len(splitWords) != wordCount {
		return nil, er.Errorf("Expected a %d word seed", wordCount)
	}
	nums := [wordCount]int16{}
	defer zeroNums(nums[:])
LANGUAGE:
	for _, wd := range allWords {
		for i, word := range splitWords {
			if num, ok := wd.rwords[word]; ok {
				if !ok {
					continue LANGUAGE
				}
				nums[i] = num
			}
		}
		return fromNums(nums)
	}
	return nil, er.New("Could not decode the words provided, check for typos")
}

// Zero wipes the data from this seed from memory.
func (s *SeedEnc) Zero() {
	zero.Bytes(s.Bytes[:])
}

// NeedsPassphrase returns true if the seed requires a passphrase in order
// to decrypt.
func (s *SeedEnc) NeedsPassphrase() bool {
	return s.getE()
}

// Decrypt converts a SeedEnc into a Seed (decrypted)
// If no passphrase is required by the seed (see: seed.NeedsPassphrase()),
// the passphrase is not used.
// If the decrypted seed birthday is in the future or so far in the past
// as to be before the seed derivation code was written, this function will
// result in an error, unless force is true.
func (s *SeedEnc) Decrypt(passphrase []byte, force bool) (*Seed, er.R) {
	out := Seed{}
	if !s.getE() {
		passphrase = nil
	}
	cipher(&out.seedBin.Bytes, &s.Bytes, passphrase)
	var err er.R
	bday := out.Birthday()
	if time.Now().Before(bday) && !force {
		err = er.Errorf("The birthday of this seed appears to be "+
			"[%s] which is in the future, the seed is probably invalid",
			bday.String())
	} else if time.Unix(1586276691, 0).After(bday) {
		err = er.Errorf("The birthday of this seed appears to be "+
			"[%s] which is before this code was written so the seed is "+
			"probably invalid", bday.String())
	} else {
		return &out, nil
	}
	out.Zero()
	return nil, err
}

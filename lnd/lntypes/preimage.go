package lntypes

import (
	"encoding/hex"

	sha256 "github.com/minio/sha256-simd"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/btcutil/util"
)

// PreimageSize of array used to store preimagees.
const PreimageSize = 32

// Preimage is used in several of the lightning messages and common structures.
// It represents a payment preimage.
type Preimage [PreimageSize]byte

// String returns the Preimage as a hexadecimal string.
func (p Preimage) String() string {
	return hex.EncodeToString(p[:])
}

// MakePreimage returns a new Preimage from a bytes slice. An error is returned
// if the number of bytes passed in is not PreimageSize.
func MakePreimage(newPreimage []byte) (Preimage, er.R) {
	nhlen := len(newPreimage)
	if nhlen != PreimageSize {
		return Preimage{}, er.Errorf("invalid preimage length of %v, "+
			"want %v", nhlen, PreimageSize)
	}

	var preimage Preimage
	copy(preimage[:], newPreimage)

	return preimage, nil
}

// MakePreimageFromStr creates a Preimage from a hex preimage string.
func MakePreimageFromStr(newPreimage string) (Preimage, er.R) {
	// Return error if preimage string is of incorrect length.
	if len(newPreimage) != PreimageSize*2 {
		return Preimage{}, er.Errorf("invalid preimage string length "+
			"of %v, want %v", len(newPreimage), PreimageSize*2)
	}

	preimage, err := util.DecodeHex(newPreimage)
	if err != nil {
		return Preimage{}, err
	}

	return MakePreimage(preimage)
}

// Hash returns the sha256 hash of the preimage.
func (p *Preimage) Hash() Hash {
	return Hash(sha256.Sum256(p[:]))
}

// Matches returns whether this preimage is the preimage of the given hash.
func (p *Preimage) Matches(h Hash) bool {
	return h == p.Hash()
}

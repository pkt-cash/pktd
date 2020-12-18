package sha1

import "encoding/binary"

const (
	size      = 20
	blocksize = 64
)

// Sum computes a SHA1 hash
func Sum(
	data []byte,
) [size]byte {
	n := len(
		data,
	)
	h := [5]uint32{
		0x67452301,
		0xefcdab89,
		0x98badcfe,
		0x10325476,
		0xc3d2e1f0,
	}
	for len(
		data,
	) >= blocksize {
		block(
			&h,
			data,
		)
		data = data[blocksize:]
	}
	tmp := make(
		[]byte,
		blocksize,
	)
	copy(
		tmp,
		data,
	)
	tmp[len(
		data,
	)] = 0x80
	if len(
		data,
	) >= 56 {
		block(
			&h,
			tmp,
		)
		for i := 0; i < blocksize; i++ {
			tmp[i] = 0
		}
	}
	binary.BigEndian.PutUint64(
		tmp[56:],
		uint64(8*n),
	)
	block(
		&h,
		tmp,
	)
	var digest [size]byte
	for i := 0; i < 5; i++ {
		binary.BigEndian.PutUint32(
			digest[4*i:],
			h[i],
		)
	}
	return digest
}

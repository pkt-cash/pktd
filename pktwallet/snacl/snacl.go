// Copyright (c) 2014-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package snacl

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"io"
	"runtime/debug"

	"github.com/pkt-cash/pktd/btcutil/er"
	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/scrypt"

	"github.com/pkt-cash/pktd/pktwallet/internal/zero"
)

var (
	prng = rand.Reader
)

var Err er.ErrorType = er.NewErrorType("snacl.Err")

// Error types and messages.
var (
	ErrInvalidPassword = Err.Code("ErrInvalidPassword")
	ErrMalformed       = Err.Code("ErrMalformed")
	ErrDecryptFailed   = Err.Code("ErrDecryptFailed")
)

// Various constants needed for encryption scheme.
const (
	// Expose secretbox's Overhead const here for convenience.
	Overhead  = secretbox.Overhead
	KeySize   = 32
	NonceSize = 24
	DefaultN  = 16384 // 2^14
	DefaultR  = 8
	DefaultP  = 1
)

// CryptoKey represents a secret key which can be used to encrypt and decrypt
// data.
type CryptoKey [KeySize]byte

// Encrypt encrypts the passed data.
func (ck *CryptoKey) Encrypt(in []byte) ([]byte, er.R) {
	var nonce [NonceSize]byte
	_, err := io.ReadFull(prng, nonce[:])
	if err != nil {
		return nil, er.E(err)
	}
	blob := secretbox.Seal(nil, in, &nonce, (*[KeySize]byte)(ck))
	return append(nonce[:], blob...), nil
}

// Decrypt decrypts the passed data.  The must be the output of the Encrypt
// function.
func (ck *CryptoKey) Decrypt(in []byte) ([]byte, er.R) {
	if len(in) < NonceSize {
		return nil, ErrMalformed.New("nonce too short", nil)
	}

	var nonce [NonceSize]byte
	copy(nonce[:], in[:NonceSize])
	blob := in[NonceSize:]

	opened, ok := secretbox.Open(nil, blob, &nonce, (*[KeySize]byte)(ck))
	if !ok {
		return nil, ErrDecryptFailed.Default()
	}

	return opened, nil
}

// Zero clears the key by manually zeroing all memory.  This is for security
// conscience application which wish to zero the memory after they've used it
// rather than waiting until it's reclaimed by the garbage collector.  The
// key is no longer usable after this call.
func (ck *CryptoKey) Zero() {
	zero.Bytea32((*[KeySize]byte)(ck))
}

// GenerateCryptoKey generates a new crypotgraphically random key.
func GenerateCryptoKey() (*CryptoKey, er.R) {
	var key CryptoKey
	_, err := io.ReadFull(prng, key[:])
	if err != nil {
		return nil, er.E(err)
	}

	return &key, nil
}

// Parameters are not secret and can be stored in plain text.
type Parameters struct {
	Salt   [KeySize]byte
	Digest [sha256.Size]byte
	N      int
	R      int
	P      int
}

// SecretKey houses a crypto key and the parameters needed to derive it from a
// passphrase.  It should only be used in memory.
type SecretKey struct {
	Key        *CryptoKey
	Parameters Parameters
}

// deriveKey fills out the Key field.
func (sk *SecretKey) deriveKey(password *[]byte) er.R {
	key, err := scrypt.Key(*password, sk.Parameters.Salt[:],
		sk.Parameters.N,
		sk.Parameters.R,
		sk.Parameters.P,
		len(sk.Key))
	if err != nil {
		return er.E(err)
	}
	copy(sk.Key[:], key)
	zero.Bytes(key)

	// I'm not a fan of forced garbage collections, but scrypt allocates a
	// ton of memory and calling it back to back without a GC cycle in
	// between means you end up needing twice the amount of memory.  For
	// example, if your scrypt parameters are such that you require 1GB and
	// you call it twice in a row, without this you end up allocating 2GB
	// since the first GB probably hasn't been released yet.
	debug.FreeOSMemory()

	return nil
}

// Marshal returns the Parameters field marshaled into a format suitable for
// storage.  This result of this can be stored in clear text.
func (sk *SecretKey) Marshal() []byte {
	params := &sk.Parameters

	// The marshaled format for the the params is as follows:
	//   <salt><digest><N><R><P>
	//
	// KeySize + sha256.Size + N (8 bytes) + R (8 bytes) + P (8 bytes)
	marshaled := make([]byte, KeySize+sha256.Size+24)

	b := marshaled
	copy(b[:KeySize], params.Salt[:])
	b = b[KeySize:]
	copy(b[:sha256.Size], params.Digest[:])
	b = b[sha256.Size:]
	binary.LittleEndian.PutUint64(b[:8], uint64(params.N))
	b = b[8:]
	binary.LittleEndian.PutUint64(b[:8], uint64(params.R))
	b = b[8:]
	binary.LittleEndian.PutUint64(b[:8], uint64(params.P))

	return marshaled
}

// Unmarshal unmarshalls the parameters needed to derive the secret key from a
// passphrase into sk.
func (sk *SecretKey) Unmarshal(marshaled []byte) er.R {
	if sk.Key == nil {
		sk.Key = (*CryptoKey)(&[KeySize]byte{})
	}

	// The marshaled format for the the params is as follows:
	//   <salt><digest><N><R><P>
	//
	// KeySize + sha256.Size + N (8 bytes) + R (8 bytes) + P (8 bytes)
	if len(marshaled) != KeySize+sha256.Size+24 {
		return ErrMalformed.New("wrong size secret key", nil)
	}

	params := &sk.Parameters
	copy(params.Salt[:], marshaled[:KeySize])
	marshaled = marshaled[KeySize:]
	copy(params.Digest[:], marshaled[:sha256.Size])
	marshaled = marshaled[sha256.Size:]
	params.N = int(binary.LittleEndian.Uint64(marshaled[:8]))
	marshaled = marshaled[8:]
	params.R = int(binary.LittleEndian.Uint64(marshaled[:8]))
	marshaled = marshaled[8:]
	params.P = int(binary.LittleEndian.Uint64(marshaled[:8]))

	return nil
}

// Zero zeroes the underlying secret key while leaving the parameters intact.
// This effectively makes the key unusable until it is derived again via the
// DeriveKey function.
func (sk *SecretKey) Zero() {
	sk.Key.Zero()
}

// DeriveKey derives the underlying secret key and ensures it matches the
// expected digest.  This should only be called after previously calling the
// Zero function or on an initial Unmarshal.
func (sk *SecretKey) DeriveKey(password *[]byte) er.R {
	if err := sk.deriveKey(password); err != nil {
		return err
	}

	// verify password
	digest := sha256.Sum256(sk.Key[:])
	if subtle.ConstantTimeCompare(digest[:], sk.Parameters.Digest[:]) != 1 {
		return ErrInvalidPassword.Default()
	}

	return nil
}

// Encrypt encrypts in bytes and returns a JSON blob.
func (sk *SecretKey) Encrypt(in []byte) ([]byte, er.R) {
	return sk.Key.Encrypt(in)
}

// Decrypt takes in a JSON blob and returns it's decrypted form.
func (sk *SecretKey) Decrypt(in []byte) ([]byte, er.R) {
	return sk.Key.Decrypt(in)
}

// NewSecretKey returns a SecretKey structure based on the passed parameters.
func NewSecretKey(password *[]byte, N, r, p int) (*SecretKey, er.R) {
	sk := SecretKey{
		Key: (*CryptoKey)(&[KeySize]byte{}),
	}
	// setup parameters
	sk.Parameters.N = N
	sk.Parameters.R = r
	sk.Parameters.P = p
	_, errr := io.ReadFull(prng, sk.Parameters.Salt[:])
	if errr != nil {
		return nil, er.E(errr)
	}

	// derive key
	err := sk.deriveKey(password)
	if err != nil {
		return nil, err
	}

	// store digest
	sk.Parameters.Digest = sha256.Sum256(sk.Key[:])

	return &sk, nil
}

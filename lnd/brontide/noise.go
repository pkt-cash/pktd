package brontide

import (
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"math"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"

	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/btcutil/util"
	"github.com/pkt-cash/pktd/lnd/keychain"
)

const (
	// protocolName is the precise instantiation of the Noise protocol
	// handshake at the center of Brontide. This value will be used as part
	// of the prologue. If the initiator and responder aren't using the
	// exact same string for this value, along with prologue of the Bitcoin
	// network, then the initial handshake will fail.
	protocolName = "Noise_XK_secp256k1_ChaChaPoly_SHA256"

	// macSize is the length in bytes of the tags generated by poly1305.
	macSize = 16

	// lengthHeaderSize is the number of bytes used to prefix encode the
	// length of a message payload.
	lengthHeaderSize = 2

	// encHeaderSize is the number of bytes required to hold an encrypted
	// header and it's MAC.
	encHeaderSize = lengthHeaderSize + macSize

	// keyRotationInterval is the number of messages sent on a single
	// cipher stream before the keys are rotated forwards.
	keyRotationInterval = 1000

	// handshakeReadTimeout is a read timeout that will be enforced when
	// waiting for data payloads during the various acts of Brontide. If
	// the remote party fails to deliver the proper payload within this
	// time frame, then we'll fail the connection.
	handshakeReadTimeout = time.Second * 5
)

var (
	Err = er.NewErrorType("lnd.brontide")
	// ErrMaxMessageLengthExceeded is returned when a message to be written to
	// the cipher session exceeds the maximum allowed message payload.
	ErrMaxMessageLengthExceeded = Err.CodeWithDetail("ErrMaxMessageLengthExceeded",
		"the generated payload exceeds the max allowed message length of (2^16)-1")

	// ErrMessageNotFlushed signals that the connection cannot accept a new
	// message because the prior message has not been fully flushed.
	ErrMessageNotFlushed = Err.CodeWithDetail("ErrMessageNotFlushed", "prior message not flushed")

	// lightningPrologue is the noise prologue that is used to initialize
	// the brontide noise handshake.
	lightningPrologue = []byte("lightning")

	// ephemeralGen is the default ephemeral key generator, used to derive a
	// unique ephemeral key for each brontide handshake.
	ephemeralGen = func() (*btcec.PrivateKey, er.R) {
		return btcec.NewPrivateKey(btcec.S256())
	}
)

// TODO(roasbeef): free buffer pool?

// ecdh performs an ECDH operation between pub and priv. The returned value is
// the sha256 of the compressed shared point.
func ecdh(pub *btcec.PublicKey, priv keychain.SingleKeyECDH) ([]byte, er.R) {
	hash, err := priv.ECDH(pub)
	return hash[:], err
}

// cipherState encapsulates the state for the AEAD which will be used to
// encrypt+authenticate any payloads sent during the handshake, and messages
// sent once the handshake has completed.
type cipherState struct {
	// nonce is the nonce passed into the chacha20-poly1305 instance for
	// encryption+decryption. The nonce is incremented after each successful
	// encryption/decryption.
	//
	// TODO(roasbeef): this should actually be 96 bit
	nonce uint64

	// secretKey is the shared symmetric key which will be used to
	// instantiate the cipher.
	//
	// TODO(roasbeef): m-lock??
	secretKey [32]byte

	// salt is an additional secret which is used during key rotation to
	// generate new keys.
	salt [32]byte

	// cipher is an instance of the ChaCha20-Poly1305 AEAD construction
	// created using the secretKey above.
	cipher cipher.AEAD
}

// Encrypt returns a ciphertext which is the encryption of the plainText
// observing the passed associatedData within the AEAD construction.
func (c *cipherState) Encrypt(associatedData, cipherText, plainText []byte) []byte {
	defer func() {
		c.nonce++

		if c.nonce == keyRotationInterval {
			c.rotateKey()
		}
	}()

	var nonce [12]byte
	binary.LittleEndian.PutUint64(nonce[4:], c.nonce)

	return c.cipher.Seal(cipherText, nonce[:], plainText, associatedData)
}

// Decrypt attempts to decrypt the passed ciphertext observing the specified
// associatedData within the AEAD construction. In the case that the final MAC
// check fails, then a non-nil error will be returned.
func (c *cipherState) Decrypt(associatedData, plainText, cipherText []byte) ([]byte, er.R) {
	defer func() {
		c.nonce++

		if c.nonce == keyRotationInterval {
			c.rotateKey()
		}
	}()

	var nonce [12]byte
	binary.LittleEndian.PutUint64(nonce[4:], c.nonce)

	o, e := c.cipher.Open(plainText, nonce[:], cipherText, associatedData)
	return o, er.E(e)
}

// InitializeKey initializes the secret key and AEAD cipher scheme based off of
// the passed key.
func (c *cipherState) InitializeKey(key [32]byte) {
	c.secretKey = key
	c.nonce = 0

	// Safe to ignore the error here as our key is properly sized
	// (32-bytes).
	c.cipher, _ = chacha20poly1305.New(c.secretKey[:])
}

// InitializeKeyWithSalt is identical to InitializeKey however it also sets the
// cipherState's salt field which is used for key rotation.
func (c *cipherState) InitializeKeyWithSalt(salt, key [32]byte) {
	c.salt = salt
	c.InitializeKey(key)
}

// rotateKey rotates the current encryption/decryption key for this cipherState
// instance. Key rotation is performed by ratcheting the current key forward
// using an HKDF invocation with the cipherState's salt as the salt, and the
// current key as the input.
func (c *cipherState) rotateKey() {
	var (
		info    []byte
		nextKey [32]byte
	)

	oldKey := c.secretKey
	h := hkdf.New(sha256.New, oldKey[:], c.salt[:], info)

	// hkdf(ck, k, zero)
	// |
	// | \
	// |  \
	// ck  k'
	h.Read(c.salt[:])
	h.Read(nextKey[:])

	c.InitializeKey(nextKey)
}

// symmetricState encapsulates a cipherState object and houses the ephemeral
// handshake digest state. This struct is used during the handshake to derive
// new shared secrets based off of the result of ECDH operations. Ultimately,
// the final key yielded by this struct is the result of an incremental
// Triple-DH operation.
type symmetricState struct {
	cipherState

	// chainingKey is used as the salt to the HKDF function to derive a new
	// chaining key as well as a new tempKey which is used for
	// encryption/decryption.
	chainingKey [32]byte

	// tempKey is the latter 32 bytes resulted from the latest HKDF
	// iteration. This key is used to encrypt/decrypt any handshake
	// messages or payloads sent until the next DH operation is executed.
	tempKey [32]byte

	// handshakeDigest is the cumulative hash digest of all handshake
	// messages sent from start to finish. This value is never transmitted
	// to the other side, but will be used as the AD when
	// encrypting/decrypting messages using our AEAD construction.
	handshakeDigest [32]byte
}

// mixKey implements a basic HKDF-based key ratchet. This method is called
// with the result of each DH output generated during the handshake process.
// The first 32 bytes extract from the HKDF reader is the next chaining key,
// then latter 32 bytes become the temp secret key using within any future AEAD
// operations until another DH operation is performed.
func (s *symmetricState) mixKey(input []byte) {
	var info []byte

	secret := input
	salt := s.chainingKey
	h := hkdf.New(sha256.New, secret, salt[:], info)

	// hkdf(ck, input, zero)
	// |
	// | \
	// |  \
	// ck  k
	h.Read(s.chainingKey[:])
	h.Read(s.tempKey[:])

	// cipher.k = temp_key
	s.InitializeKey(s.tempKey)
}

// mixHash hashes the passed input data into the cumulative handshake digest.
// The running result of this value (h) is used as the associated data in all
// decryption/encryption operations.
func (s *symmetricState) mixHash(data []byte) {
	h := sha256.New()
	h.Write(s.handshakeDigest[:])
	h.Write(data)

	copy(s.handshakeDigest[:], h.Sum(nil))
}

// EncryptAndHash returns the authenticated encryption of the passed plaintext.
// When encrypting the handshake digest (h) is used as the associated data to
// the AEAD cipher.
func (s *symmetricState) EncryptAndHash(plaintext []byte) []byte {
	ciphertext := s.Encrypt(s.handshakeDigest[:], nil, plaintext)

	s.mixHash(ciphertext)

	return ciphertext
}

// DecryptAndHash returns the authenticated decryption of the passed
// ciphertext. When encrypting the handshake digest (h) is used as the
// associated data to the AEAD cipher.
func (s *symmetricState) DecryptAndHash(ciphertext []byte) ([]byte, er.R) {
	plaintext, err := s.Decrypt(s.handshakeDigest[:], nil, ciphertext)
	if err != nil {
		return nil, err
	}

	s.mixHash(ciphertext)

	return plaintext, nil
}

// InitializeSymmetric initializes the symmetric state by setting the handshake
// digest (h) and the chaining key (ck) to protocol name.
func (s *symmetricState) InitializeSymmetric(protocolName []byte) {
	var empty [32]byte

	s.handshakeDigest = sha256.Sum256(protocolName)
	s.chainingKey = s.handshakeDigest
	s.InitializeKey(empty)
}

// handshakeState encapsulates the symmetricState and keeps track of all the
// public keys (static and ephemeral) for both sides during the handshake
// transcript. If the handshake completes successfully, then two instances of a
// cipherState are emitted: one to encrypt messages from initiator to
// responder, and the other for the opposite direction.
type handshakeState struct {
	symmetricState

	initiator bool

	localStatic    keychain.SingleKeyECDH
	localEphemeral keychain.SingleKeyECDH // nolint (false positive)

	remoteStatic    *btcec.PublicKey
	remoteEphemeral *btcec.PublicKey
}

// newHandshakeState returns a new instance of the handshake state initialized
// with the prologue and protocol name. If this is the responder's handshake
// state, then the remotePub can be nil.
func newHandshakeState(initiator bool, prologue []byte,
	localKey keychain.SingleKeyECDH,
	remotePub *btcec.PublicKey) handshakeState {
	h := handshakeState{
		initiator:    initiator,
		localStatic:  localKey,
		remoteStatic: remotePub,
	}

	// Set the current chaining key and handshake digest to the hash of the
	// protocol name, and additionally mix in the prologue. If either sides
	// disagree about the prologue or protocol name, then the handshake
	// will fail.
	h.InitializeSymmetric([]byte(protocolName))
	h.mixHash(prologue)

	// In Noise_XK, the initiator should know the responder's static
	// public key, therefore we include the responder's static key in the
	// handshake digest. If the initiator gets this value wrong, then the
	// handshake will fail.
	if initiator {
		h.mixHash(remotePub.SerializeCompressed())
	} else {
		h.mixHash(localKey.PubKey().SerializeCompressed())
	}

	return h
}

// EphemeralGenerator is a functional option that allows callers to substitute
// a custom function for use when generating ephemeral keys for ActOne or
// ActTwo. The function closure returned by this function can be passed into
// NewBrontideMachine as a function option parameter.
func EphemeralGenerator(gen func() (*btcec.PrivateKey, er.R)) func(*Machine) {
	return func(m *Machine) {
		m.ephemeralGen = gen
	}
}

// Machine is a state-machine which implements Brontide: an
// Authenticated-key Exchange in Three Acts. Brontide is derived from the Noise
// framework, specifically implementing the Noise_XK handshake. Once the
// initial 3-act handshake has completed all messages are encrypted with a
// chacha20 AEAD cipher. On the wire, all messages are prefixed with an
// authenticated+encrypted length field. Additionally, the encrypted+auth'd
// length prefix is used as the AD when encrypting+decryption messages. This
// construction provides confidentiality of packet length, avoids introducing
// a padding-oracle, and binds the encrypted packet length to the packet
// itself.
//
// The acts proceeds the following order (initiator on the left):
//  GenActOne()   ->
//                    RecvActOne()
//                <-  GenActTwo()
//  RecvActTwo()
//  GenActThree() ->
//                    RecvActThree()
//
// This exchange corresponds to the following Noise handshake:
//   <- s
//   ...
//   -> e, es
//   <- e, ee
//   -> s, se
type Machine struct {
	sendCipher cipherState
	recvCipher cipherState

	ephemeralGen func() (*btcec.PrivateKey, er.R)

	handshakeState

	// nextCipherHeader is a static buffer that we'll use to read in the
	// next ciphertext header from the wire. The header is a 2 byte length
	// (of the next ciphertext), followed by a 16 byte MAC.
	nextCipherHeader [encHeaderSize]byte

	// nextHeaderSend holds a reference to the remaining header bytes to
	// write out for a pending message. This allows us to tolerate timeout
	// errors that cause partial writes.
	nextHeaderSend []byte

	// nextHeaderBody holds a reference to the remaining body bytes to write
	// out for a pending message. This allows us to tolerate timeout errors
	// that cause partial writes.
	nextBodySend []byte
}

// NewBrontideMachine creates a new instance of the brontide state-machine. If
// the responder (listener) is creating the object, then the remotePub should
// be nil. The handshake state within brontide is initialized using the ascii
// string "lightning" as the prologue. The last parameter is a set of variadic
// arguments for adding additional options to the brontide Machine
// initialization.
func NewBrontideMachine(initiator bool, localKey keychain.SingleKeyECDH,
	remotePub *btcec.PublicKey, options ...func(*Machine)) *Machine {
	handshake := newHandshakeState(
		initiator, lightningPrologue, localKey, remotePub,
	)

	m := &Machine{
		handshakeState: handshake,
		ephemeralGen:   ephemeralGen,
	}

	// With the default options established, we'll now process all the
	// options passed in as parameters.
	for _, option := range options {
		option(m)
	}

	return m
}

const (
	// HandshakeVersion is the expected version of the brontide handshake.
	// Any messages that carry a different version will cause the handshake
	// to abort immediately.
	HandshakeVersion = byte(0)

	// ActOneSize is the size of the packet sent from initiator to
	// responder in ActOne. The packet consists of a handshake version, an
	// ephemeral key in compressed format, and a 16-byte poly1305 tag.
	//
	// 1 + 33 + 16
	ActOneSize = 50

	// ActTwoSize is the size the packet sent from responder to initiator
	// in ActTwo. The packet consists of a handshake version, an ephemeral
	// key in compressed format and a 16-byte poly1305 tag.
	//
	// 1 + 33 + 16
	ActTwoSize = 50

	// ActThreeSize is the size of the packet sent from initiator to
	// responder in ActThree. The packet consists of a handshake version,
	// the initiators static key encrypted with strong forward secrecy and
	// a 16-byte poly1035 tag.
	//
	// 1 + 33 + 16 + 16
	ActThreeSize = 66
)

// GenActOne generates the initial packet (act one) to be sent from initiator
// to responder. During act one the initiator generates a fresh ephemeral key,
// hashes it into the handshake digest, and performs an ECDH between this key
// and the responder's static key. Future payloads are encrypted with a key
// derived from this result.
//
//    -> e, es
func (b *Machine) GenActOne() ([ActOneSize]byte, er.R) {
	var actOne [ActOneSize]byte

	// e
	localEphemeral, err := b.ephemeralGen()
	if err != nil {
		return actOne, err
	}
	b.localEphemeral = &keychain.PrivKeyECDH{
		PrivKey: localEphemeral,
	}

	ephemeral := localEphemeral.PubKey().SerializeCompressed()
	b.mixHash(ephemeral)

	// es
	s, err := ecdh(b.remoteStatic, b.localEphemeral)
	if err != nil {
		return actOne, err
	}
	b.mixKey(s[:])

	authPayload := b.EncryptAndHash([]byte{})

	actOne[0] = HandshakeVersion
	copy(actOne[1:34], ephemeral)
	copy(actOne[34:], authPayload)

	return actOne, nil
}

// RecvActOne processes the act one packet sent by the initiator. The responder
// executes the mirrored actions to that of the initiator extending the
// handshake digest and deriving a new shared secret based on an ECDH with the
// initiator's ephemeral key and responder's static key.
func (b *Machine) RecvActOne(actOne [ActOneSize]byte) er.R {
	var (
		err er.R
		e   [33]byte
		p   [16]byte
	)

	// If the handshake version is unknown, then the handshake fails
	// immediately.
	if actOne[0] != HandshakeVersion {
		return er.Errorf("act one: invalid handshake version: %v, "+
			"only %v is valid, msg=%x", actOne[0], HandshakeVersion,
			actOne[:])
	}

	copy(e[:], actOne[1:34])
	copy(p[:], actOne[34:])

	// e
	b.remoteEphemeral, err = btcec.ParsePubKey(e[:], btcec.S256())
	if err != nil {
		return err
	}
	b.mixHash(b.remoteEphemeral.SerializeCompressed())

	// es
	s, err := ecdh(b.remoteEphemeral, b.localStatic)
	if err != nil {
		return err
	}
	b.mixKey(s)

	// If the initiator doesn't know our static key, then this operation
	// will fail.
	_, err = b.DecryptAndHash(p[:])
	return err
}

// GenActTwo generates the second packet (act two) to be sent from the
// responder to the initiator. The packet for act two is identical to that of
// act one, but then results in a different ECDH operation between the
// initiator's and responder's ephemeral keys.
//
//    <- e, ee
func (b *Machine) GenActTwo() ([ActTwoSize]byte, er.R) {
	var actTwo [ActTwoSize]byte

	// e
	localEphemeral, err := b.ephemeralGen()
	if err != nil {
		return actTwo, err
	}
	b.localEphemeral = &keychain.PrivKeyECDH{
		PrivKey: localEphemeral,
	}

	ephemeral := localEphemeral.PubKey().SerializeCompressed()
	b.mixHash(localEphemeral.PubKey().SerializeCompressed())

	// ee
	s, err := ecdh(b.remoteEphemeral, b.localEphemeral)
	if err != nil {
		return actTwo, err
	}
	b.mixKey(s)

	authPayload := b.EncryptAndHash([]byte{})

	actTwo[0] = HandshakeVersion
	copy(actTwo[1:34], ephemeral)
	copy(actTwo[34:], authPayload)

	return actTwo, nil
}

// RecvActTwo processes the second packet (act two) sent from the responder to
// the initiator. A successful processing of this packet authenticates the
// initiator to the responder.
func (b *Machine) RecvActTwo(actTwo [ActTwoSize]byte) er.R {
	var (
		err er.R
		e   [33]byte
		p   [16]byte
	)

	// If the handshake version is unknown, then the handshake fails
	// immediately.
	if actTwo[0] != HandshakeVersion {
		return er.Errorf("act two: invalid handshake version: %v, "+
			"only %v is valid, msg=%x", actTwo[0], HandshakeVersion,
			actTwo[:])
	}

	copy(e[:], actTwo[1:34])
	copy(p[:], actTwo[34:])

	// e
	b.remoteEphemeral, err = btcec.ParsePubKey(e[:], btcec.S256())
	if err != nil {
		return err
	}
	b.mixHash(b.remoteEphemeral.SerializeCompressed())

	// ee
	s, err := ecdh(b.remoteEphemeral, b.localEphemeral)
	if err != nil {
		return err
	}
	b.mixKey(s)

	_, err = b.DecryptAndHash(p[:])
	return err
}

// GenActThree creates the final (act three) packet of the handshake. Act three
// is to be sent from the initiator to the responder. The purpose of act three
// is to transmit the initiator's public key under strong forward secrecy to
// the responder. This act also includes the final ECDH operation which yields
// the final session.
//
//    -> s, se
func (b *Machine) GenActThree() ([ActThreeSize]byte, er.R) {
	var actThree [ActThreeSize]byte

	ourPubkey := b.localStatic.PubKey().SerializeCompressed()
	ciphertext := b.EncryptAndHash(ourPubkey)

	s, err := ecdh(b.remoteEphemeral, b.localStatic)
	if err != nil {
		return actThree, err
	}
	b.mixKey(s)

	authPayload := b.EncryptAndHash([]byte{})

	actThree[0] = HandshakeVersion
	copy(actThree[1:50], ciphertext)
	copy(actThree[50:], authPayload)

	// With the final ECDH operation complete, derive the session sending
	// and receiving keys.
	b.split()

	return actThree, nil
}

// RecvActThree processes the final act (act three) sent from the initiator to
// the responder. After processing this act, the responder learns of the
// initiator's static public key. Decryption of the static key serves to
// authenticate the initiator to the responder.
func (b *Machine) RecvActThree(actThree [ActThreeSize]byte) er.R {
	var (
		err er.R
		s   [33 + 16]byte
		p   [16]byte
	)

	// If the handshake version is unknown, then the handshake fails
	// immediately.
	if actThree[0] != HandshakeVersion {
		return er.Errorf("act three: invalid handshake version: %v, "+
			"only %v is valid, msg=%x", actThree[0], HandshakeVersion,
			actThree[:])
	}

	copy(s[:], actThree[1:33+16+1])
	copy(p[:], actThree[33+16+1:])

	// s
	remotePub, err := b.DecryptAndHash(s[:])
	if err != nil {
		return err
	}
	b.remoteStatic, err = btcec.ParsePubKey(remotePub, btcec.S256())
	if err != nil {
		return err
	}

	// se
	se, err := ecdh(b.remoteStatic, b.localEphemeral)
	if err != nil {
		return err
	}
	b.mixKey(se)

	if _, err := b.DecryptAndHash(p[:]); err != nil {
		return err
	}

	// With the final ECDH operation complete, derive the session sending
	// and receiving keys.
	b.split()

	return nil
}

// split is the final wrap-up act to be executed at the end of a successful
// three act handshake. This function creates two internal cipherState
// instances: one which is used to encrypt messages from the initiator to the
// responder, and another which is used to encrypt message for the opposite
// direction.
func (b *Machine) split() {
	var (
		empty   []byte
		sendKey [32]byte
		recvKey [32]byte
	)

	h := hkdf.New(sha256.New, empty, b.chainingKey[:], empty)

	// If we're the initiator the first 32 bytes are used to encrypt our
	// messages and the second 32-bytes to decrypt their messages. For the
	// responder the opposite is true.
	if b.initiator {
		h.Read(sendKey[:])
		b.sendCipher = cipherState{}
		b.sendCipher.InitializeKeyWithSalt(b.chainingKey, sendKey)

		h.Read(recvKey[:])
		b.recvCipher = cipherState{}
		b.recvCipher.InitializeKeyWithSalt(b.chainingKey, recvKey)
	} else {
		h.Read(recvKey[:])
		b.recvCipher = cipherState{}
		b.recvCipher.InitializeKeyWithSalt(b.chainingKey, recvKey)

		h.Read(sendKey[:])
		b.sendCipher = cipherState{}
		b.sendCipher.InitializeKeyWithSalt(b.chainingKey, sendKey)
	}
}

// WriteMessage encrypts and buffers the next message p. The ciphertext of the
// message is prepended with an encrypt+auth'd length which must be used as the
// AD to the AEAD construction when being decrypted by the other side.
//
// NOTE: This DOES NOT write the message to the wire, it should be followed by a
// call to Flush to ensure the message is written.
func (b *Machine) WriteMessage(p []byte) er.R {
	// The total length of each message payload including the MAC size
	// payload exceed the largest number encodable within a 16-bit unsigned
	// integer.
	if len(p) > math.MaxUint16 {
		return ErrMaxMessageLengthExceeded.Default()
	}

	// If a prior message was written but it hasn't been fully flushed,
	// return an error as we only support buffering of one message at a
	// time.
	if len(b.nextHeaderSend) > 0 || len(b.nextBodySend) > 0 {
		return ErrMessageNotFlushed.Default()
	}

	// The full length of the packet is only the packet length, and does
	// NOT include the MAC.
	fullLength := uint16(len(p))

	var pktLen [2]byte
	binary.BigEndian.PutUint16(pktLen[:], fullLength)

	// First, generate the encrypted+MAC'd length prefix for the packet.
	b.nextHeaderSend = b.sendCipher.Encrypt(nil, nil, pktLen[:])

	// Finally, generate the encrypted packet itself.
	b.nextBodySend = b.sendCipher.Encrypt(nil, nil, p)

	return nil
}

// Flush attempts to write a message buffered using WriteMessage to the provided
// io.Writer. If no buffered message exists, this will result in a NOP.
// Otherwise, it will continue to write the remaining bytes, picking up where
// the byte stream left off in the event of a partial write. The number of bytes
// returned reflects the number of plaintext bytes in the payload, and does not
// account for the overhead of the header or MACs.
//
// NOTE: It is safe to call this method again iff a timeout error is returned.
func (b *Machine) Flush(w io.Writer) (int, er.R) {
	// First, write out the pending header bytes, if any exist. Any header
	// bytes written will not count towards the total amount flushed.
	if len(b.nextHeaderSend) > 0 {
		// Write any remaining header bytes and shift the slice to point
		// to the next segment of unwritten bytes. If an error is
		// encountered, we can continue to write the header from where
		// we left off on a subsequent call to Flush.
		n, err := util.Write(w, b.nextHeaderSend)
		b.nextHeaderSend = b.nextHeaderSend[n:]
		if err != nil {
			return 0, err
		}
	}

	// Next, write the pending body bytes, if any exist. Only the number of
	// bytes written that correspond to the ciphertext will be included in
	// the total bytes written, bytes written as part of the MAC will not be
	// counted.
	var nn int
	if len(b.nextBodySend) > 0 {
		// Write out all bytes excluding the mac and shift the body
		// slice depending on the number of actual bytes written.
		n, err := util.Write(w, b.nextBodySend)
		b.nextBodySend = b.nextBodySend[n:]

		// If we partially or fully wrote any of the body's MAC, we'll
		// subtract that contribution from the total amount flushed to
		// preserve the abstraction of returning the number of plaintext
		// bytes written by the connection.
		//
		// There are three possible scenarios we must handle to ensure
		// the returned value is correct. In the first case, the write
		// straddles both payload and MAC bytes, and we must subtract
		// the number of MAC bytes written from n. In the second, only
		// payload bytes are written, thus we can return n unmodified.
		// The final scenario pertains to the case where only MAC bytes
		// are written, none of which count towards the total.
		//
		//                 |-----------Payload------------|----MAC----|
		// Straddle:       S---------------------------------E--------0
		// Payload-only:   S------------------------E-----------------0
		// MAC-only:                                        S-------E-0
		start, end := n+len(b.nextBodySend), len(b.nextBodySend)
		switch {

		// Straddles payload and MAC bytes, subtract number of MAC bytes
		// written from the actual number written.
		case start > macSize && end <= macSize:
			nn = n - (macSize - end)

		// Only payload bytes are written, return n directly.
		case start > macSize && end > macSize:
			nn = n

		// Only MAC bytes are written, return 0 bytes written.
		default:
		}

		if err != nil {
			return nn, err
		}
	}

	return nn, nil
}

// ReadMessage attempts to read the next message from the passed io.Reader. In
// the case of an authentication error, a non-nil error is returned.
func (b *Machine) ReadMessage(r io.Reader) ([]byte, er.R) {
	pktLen, err := b.ReadHeader(r)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, pktLen)
	return b.ReadBody(r, buf)
}

// ReadHeader attempts to read the next message header from the passed
// io.Reader. The header contains the length of the next body including
// additional overhead of the MAC. In the case of an authentication error, a
// non-nil error is returned.
//
// NOTE: This method SHOULD NOT be used in the case that the io.Reader may be
// adversarial and induce long delays. If the caller needs to set read deadlines
// appropriately, it is preferred that they use the split ReadHeader and
// ReadBody methods so that the deadlines can be set appropriately on each.
func (b *Machine) ReadHeader(r io.Reader) (uint32, er.R) {
	_, err := util.ReadFull(r, b.nextCipherHeader[:])
	if err != nil {
		return 0, err
	}

	// Attempt to decrypt+auth the packet length present in the stream.
	pktLenBytes, err := b.recvCipher.Decrypt(
		nil, nil, b.nextCipherHeader[:],
	)
	if err != nil {
		return 0, err
	}

	// Compute the packet length that we will need to read off the wire.
	pktLen := uint32(binary.BigEndian.Uint16(pktLenBytes)) + macSize

	return pktLen, nil
}

// ReadBody attempts to ready the next message body from the passed io.Reader.
// The provided buffer MUST be the length indicated by the packet length
// returned by the preceding call to ReadHeader. In the case of an
// authentication eerror, a non-nil error is returned.
func (b *Machine) ReadBody(r io.Reader, buf []byte) ([]byte, er.R) {
	// Next, using the length read from the packet header, read the
	// encrypted packet itself into the buffer allocated by the read
	// pool.
	_, err := util.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	// Finally, decrypt the message held in the buffer, and return a
	// new byte slice containing the plaintext.
	// TODO(roasbeef): modify to let pass in slice
	return b.recvCipher.Decrypt(nil, nil, buf)
}

// SetCurveToNil sets the 'Curve' parameter to nil on the handshakeState keys.
// This allows us to log the Machine object without spammy log messages.
func (b *Machine) SetCurveToNil() {
	if b.localStatic != nil {
		b.localStatic.PubKey().Curve = nil
	}

	if b.localEphemeral != nil {
		b.localEphemeral.PubKey().Curve = nil
	}

	if b.remoteStatic != nil {
		b.remoteStatic.Curve = nil
	}

	if b.remoteEphemeral != nil {
		b.remoteEphemeral.Curve = nil
	}
}

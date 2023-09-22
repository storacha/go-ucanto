package signer

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"

	"github.com/alanshaw/go-ucanto/did"
	"github.com/alanshaw/go-ucanto/principal/ed25519/verifier"
	"github.com/alanshaw/go-ucanto/ucan/crypto"
	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-varint"
)

const Code = 0x1300
const Name = verifier.Name

var privateTagSize = varint.UvarintSize(Code)
var publicTagSize = varint.UvarintSize(verifier.Code)

const keySize = 32

var size = privateTagSize + keySize + publicTagSize + keySize
var pubKeyOffset = privateTagSize + keySize

func Generate() (crypto.Signer, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating Ed25519 key: %s", err)
	}
	s := make(Ed25519Signer, size)
	varint.PutUvarint(s, Code)
	copy(s[privateTagSize:], priv)
	varint.PutUvarint(s[pubKeyOffset:], verifier.Code)
	copy(s[pubKeyOffset+publicTagSize:], pub)
	return s, nil
}

func Parse(str string) (crypto.Signer, error) {
	_, bytes, err := multibase.Decode(str)
	if err != nil {
		return nil, fmt.Errorf("decoding multibase string: %s", err)
	}
	return Decode(bytes)
}

func Decode(b []byte) (crypto.Signer, error) {
	if len(b) != size {
		return nil, fmt.Errorf("invalid length: %d wanted: %d", len(b), size)
	}

	prc, err := varint.ReadUvarint(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("reading private key codec: %s", err)
	}
	if prc != Code {
		return nil, fmt.Errorf("invalid private key codec: %d", prc)
	}

	puc, err := varint.ReadUvarint(bytes.NewReader(b[pubKeyOffset:]))
	if err != nil {
		return nil, fmt.Errorf("reading public key codec: %s", err)
	}
	if puc != verifier.Code {
		return nil, fmt.Errorf("invalid public key codec: %d", prc)
	}

	_, err = verifier.Decode(b[pubKeyOffset:])
	if err != nil {
		return nil, fmt.Errorf("decoding public key: %s", err)
	}

	s := make(Ed25519Signer, size)
	copy(s, b)

	return s, nil
}

type Ed25519Signer []byte

func (s Ed25519Signer) Code() uint64 {
	return Code
}

func (s Ed25519Signer) Verifier() crypto.Verifier {
	return verifier.Ed25519Verifier(s[pubKeyOffset:])
}

func (s Ed25519Signer) DID() did.DID {
	id, _ := did.Decode(s[pubKeyOffset:])
	return id
}

func (s Ed25519Signer) Encode() []byte {
	return s
}

func (s Ed25519Signer) Sign(msg []byte) crypto.Signature {
	pk := make(ed25519.PrivateKey, ed25519.PrivateKeySize)
	copy(pk, s[privateTagSize:pubKeyOffset])
	copy(pk[ed25519.PrivateKeySize-ed25519.PublicKeySize:], s[pubKeyOffset+publicTagSize:])
	return crypto.NewSignature(crypto.EdDSA, ed25519.Sign(pk, msg))
}

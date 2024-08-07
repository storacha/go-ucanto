package signer

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"

	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-varint"
	"github.com/storacha-network/go-ucanto/did"
	"github.com/storacha-network/go-ucanto/principal"
	"github.com/storacha-network/go-ucanto/principal/ed25519/verifier"
	"github.com/storacha-network/go-ucanto/ucan/crypto/signature"
)

const Code = 0x1300
const Name = verifier.Name

const SignatureCode = verifier.SignatureCode
const SignatureAlgorithm = verifier.SignatureAlgorithm

var privateTagSize = varint.UvarintSize(Code)
var publicTagSize = varint.UvarintSize(verifier.Code)

const keySize = 32

var size = privateTagSize + keySize + publicTagSize + keySize
var pubKeyOffset = privateTagSize + keySize

func Generate() (principal.Signer, error) {
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

func Parse(str string) (principal.Signer, error) {
	_, bytes, err := multibase.Decode(str)
	if err != nil {
		return nil, fmt.Errorf("decoding multibase string: %s", err)
	}
	return Decode(bytes)
}

func Decode(b []byte) (principal.Signer, error) {
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

func (s Ed25519Signer) SignatureCode() uint64 {
	return SignatureCode
}

func (s Ed25519Signer) SignatureAlgorithm() string {
	return SignatureAlgorithm
}

func (s Ed25519Signer) Verifier() principal.Verifier {
	return verifier.Ed25519Verifier(s[pubKeyOffset:])
}

func (s Ed25519Signer) DID() did.DID {
	id, _ := did.Decode(s[pubKeyOffset:])
	return id
}

func (s Ed25519Signer) Encode() []byte {
	return s
}

func (s Ed25519Signer) Sign(msg []byte) signature.SignatureView {
	pk := make(ed25519.PrivateKey, ed25519.PrivateKeySize)
	copy(pk[0:ed25519.PublicKeySize], s[privateTagSize:pubKeyOffset])
	copy(pk[ed25519.PrivateKeySize-ed25519.PublicKeySize:ed25519.PrivateKeySize], s[pubKeyOffset+publicTagSize:pubKeyOffset+publicTagSize+keySize])
	return signature.NewSignatureView(signature.NewSignature(signature.EdDSA, ed25519.Sign(pk, msg)))
}

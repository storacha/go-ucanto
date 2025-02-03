package verifier

import (
	"bytes"
	"crypto/ed25519"
	"fmt"

	"github.com/multiformats/go-varint"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-ucanto/principal/multiformat"
	"github.com/storacha/go-ucanto/ucan/crypto/signature"
)

const Code = 0xed
const Name = "Ed25519"

const SignatureCode = signature.EdDSA
const SignatureAlgorithm = "EdDSA"

var publicTagSize = varint.UvarintSize(Code)

const keySize = 32

var size = publicTagSize + keySize

func Parse(str string) (principal.Verifier, error) {
	did, err := did.Parse(str)
	if err != nil {
		return nil, fmt.Errorf("parsing DID: %s", err)
	}
	return Decode(did.Bytes())
}

func Decode(b []byte) (principal.Verifier, error) {
	if len(b) != size {
		return nil, fmt.Errorf("invalid length: %d wanted: %d", len(b), size)
	}

	prc, err := varint.ReadUvarint(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("reading public key codec: %s", err)
	}
	if prc != Code {
		return nil, fmt.Errorf("invalid public key codec: %d", prc)
	}

	puc, err := varint.ReadUvarint(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("reading public key codec: %s", err)
	}
	if puc != Code {
		return nil, fmt.Errorf("invalid public key codec: %d", prc)
	}

	v := make(Ed25519Verifier, size)
	copy(v, b)

	return v, nil
}

// FromRaw takes raw ed25519 public key bytes and tags with the ed25519 verifier
// multiformat code, returning an ed25519 verifier.
func FromRaw(b []byte) (principal.Verifier, error) {
	return Ed25519Verifier(multiformat.TagWith(Code, b)), nil
}

type Ed25519Verifier []byte

func (v Ed25519Verifier) Code() uint64 {
	return Code
}

func (v Ed25519Verifier) Verify(msg []byte, sig signature.Signature) bool {
	if sig.Code() != signature.EdDSA {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(v[publicTagSize:]), msg, sig.Raw())
}

func (v Ed25519Verifier) DID() did.DID {
	id, _ := did.Decode(v)
	return id
}

func (v Ed25519Verifier) Encode() []byte {
	return v
}

func (s Ed25519Verifier) Raw() []byte {
	k := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(k, s[publicTagSize:])
	return k
}

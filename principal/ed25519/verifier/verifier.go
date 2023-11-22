package verifier

import (
	"bytes"
	"crypto/ed25519"
	"fmt"

	"github.com/alanshaw/go-ucanto/did"
	"github.com/alanshaw/go-ucanto/ucan/crypto/signature"
	"github.com/multiformats/go-varint"
)

const Code = 0xed
const Name = "Ed25519"

var publicTagSize = varint.UvarintSize(Code)

const keySize = 32

var size = publicTagSize + keySize

func Parse(str string) (signature.Verifier, error) {
	did, err := did.Parse(str)
	if err != nil {
		return nil, fmt.Errorf("parsing DID: %s", err)
	}
	return Decode(did.Bytes())
}

func Decode(b []byte) (signature.Verifier, error) {
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

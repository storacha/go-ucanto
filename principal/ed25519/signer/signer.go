package ed25519

import (
	"bytes"
	"fmt"

	"github.com/alanshaw/go-ucanto/did"
	"github.com/alanshaw/go-ucanto/principal"
	"github.com/alanshaw/go-ucanto/principal/ed25519/verifier"
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

	vfr, err := verifier.Decode(b[pubKeyOffset:])
	if err != nil {
		return nil, fmt.Errorf("decoding public key: %s", err)
	}

	return &edSigner{key: b, vfr: vfr}, nil
}

type edSigner struct {
	key []byte
	vfr principal.Verifier
}

func (s *edSigner) Code() uint64 {
	return Code
}

func (s *edSigner) Verifier() principal.Verifier {
	return s.vfr
}

func (s *edSigner) DID() did.DID {
	return s.vfr.DID()
}

func (s *edSigner) Encode() []byte {
	return s.key
}

func (s *edSigner) Sign(b []byte) Signature {
	return []byte{}
}

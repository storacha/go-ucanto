package verifier

import (
	"bytes"
	"fmt"

	"github.com/alanshaw/go-ucanto/principal"
	"github.com/multiformats/go-varint"
)

const Code = 0xed
const Name = "Ed25519"

var publicTagSize = varint.UvarintSize(Code)

const keySize = 32

var size = publicTagSize + keySize

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
}

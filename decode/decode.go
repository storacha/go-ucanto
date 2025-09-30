package decode

import (
	"bytes"
	"fmt"

	"github.com/multiformats/go-varint"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-ucanto/principal/ed25519/signer"
	"github.com/storacha/go-ucanto/principal/ed25519/verifier"
	rsasigner "github.com/storacha/go-ucanto/principal/rsa/signer"
	rsaverifier "github.com/storacha/go-ucanto/principal/rsa/verifier"
	"github.com/storacha/go-ucanto/ucan"
)

// Signer decodes a multiformat encoded signer back to the appropriate
// implementation (ed25519 or RSA) based on the codec prefix.
func Signer(encoded []byte) (principal.Signer, error) {
	code, err := varint.ReadUvarint(bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("reading signer codec: %w", err)
	}

	switch code {
	case signer.Code:
		return signer.Decode(encoded)
	case rsasigner.Code:
		return rsasigner.Decode(encoded)
	default:
		return nil, fmt.Errorf("unsupported signer codec: %d", code)
	}
}

// Verifier decodes a multiformat encoded verifier back to the appropriate
// implementation (ed25519 or RSA) based on the codec prefix.
func Verifier(encoded []byte) (ucan.Verifier, error) {
	code, err := varint.ReadUvarint(bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("reading verifier codec: %w", err)
	}

	switch code {
	case verifier.Code:
		return verifier.Decode(encoded)
	case rsaverifier.Code:
		return rsaverifier.Decode(encoded)
	default:
		return nil, fmt.Errorf("unsupported verifier codec: %d", code)
	}
}

// Principal attempts to decode as both signer and verifier, returning
// whichever succeeds. This is useful when you're not sure which type you have.
func Principal(encoded []byte) (interface{}, error) {
	// Try as signer first
	if s, err := Signer(encoded); err == nil {
		return s, nil
	}

	// Try as verifier
	if v, err := Verifier(encoded); err == nil {
		return v, nil
	}

	return nil, fmt.Errorf("unable to decode as either signer or verifier")
}
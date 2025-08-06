package principal

import (
	"bytes"
	"fmt"

	"github.com/multiformats/go-varint"
	"github.com/storacha/go-ucanto/principal/ed25519/signer"
	"github.com/storacha/go-ucanto/principal/ed25519/verifier"
	rsasigner "github.com/storacha/go-ucanto/principal/rsa/signer"
	rsaverifier "github.com/storacha/go-ucanto/principal/rsa/verifier"
	"github.com/storacha/go-ucanto/ucan"
)

// DecodeSigner decodes a multiformat encoded signer back to the appropriate
// implementation (ed25519 or RSA) based on the codec prefix.
func DecodeSigner(encoded []byte) (Signer, error) {
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

// DecodeVerifier decodes a multiformat encoded verifier back to the appropriate
// implementation (ed25519 or RSA) based on the codec prefix.
func DecodeVerifier(encoded []byte) (ucan.Verifier, error) {
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

// DecodePrincipal attempts to decode as both signer and verifier, returning
// whichever succeeds.
func DecodePrincipal(encoded []byte) (interface{}, error) {
	// Try as signer first
	if s, err := DecodeSigner(encoded); err == nil {
		return s, nil
	}

	// Try as verifier
	if v, err := DecodeVerifier(encoded); err == nil {
		return v, nil
	}

	return nil, fmt.Errorf("unable to decode as either signer or verifier")
}

// ComposedParser implements a parser that tries multiple principal parsers
type ComposedParser struct {
	parsers []Parser
}

// Parser interface for parsing DIDs into verifiers
type Parser interface {
	Parse(did string) (ucan.Verifier, error)
}

// NewComposedParser creates a new composed parser with the given parsers
func NewComposedParser(parsers ...Parser) *ComposedParser {
	return &ComposedParser{parsers: parsers}
}

// Parse attempts to parse the DID using each parser in sequence
func (cp *ComposedParser) Parse(did string) (ucan.Verifier, error) {
	if len(did) < 4 || did[:4] != "did:" {
		return nil, fmt.Errorf("expected DID but got %s", did)
	}

	var lastErr error
	for _, parser := range cp.parsers {
		if v, err := parser.Parse(did); err == nil {
			return v, nil
		} else {
			lastErr = err
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("unsupported DID %s: %w", did, lastErr)
	}
	return nil, fmt.Errorf("unsupported DID %s", did)
}

// Or adds another parser to the composed parser
func (cp *ComposedParser) Or(parser Parser) *ComposedParser {
	return &ComposedParser{parsers: append(cp.parsers, parser)}
}

// Ed25519Parser implements Parser for Ed25519 DIDs
type Ed25519Parser struct{}

func (p Ed25519Parser) Parse(did string) (ucan.Verifier, error) {
	return verifier.Parse(did)
}

// RSAParser implements Parser for RSA DIDs
type RSAParser struct{}

func (p RSAParser) Parse(did string) (ucan.Verifier, error) {
	return rsaverifier.Parse(did)
}

// DefaultParser returns a composed parser with all supported principal types
func DefaultParser() *ComposedParser {
	return NewComposedParser(
		Ed25519Parser{},
		RSAParser{},
	)
}

// ParseDID parses a DID string using the default composed parser
func ParseDID(did string) (ucan.Verifier, error) {
	return DefaultParser().Parse(did)
}

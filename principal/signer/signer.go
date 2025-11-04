package signer

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/multiformats/go-varint"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/principal"
	ed25519signer "github.com/storacha/go-ucanto/principal/ed25519/signer"
	rsasigner "github.com/storacha/go-ucanto/principal/rsa/signer"
	"github.com/storacha/go-ucanto/principal/verifier"
	"github.com/storacha/go-ucanto/ucan/crypto/signature"
)

type Unwrapper interface {
	// Unwrap returns the unwrapped did:key of this signer.
	Unwrap() principal.Signer
}

type WrappedSigner interface {
	principal.Signer
	Unwrapper
}

type wrapsgn struct {
	key      principal.Signer
	verifier principal.Verifier
}

func (w wrapsgn) Code() uint64 {
	return w.key.Code()
}

func (w wrapsgn) DID() did.DID {
	return w.verifier.DID()
}

func (w wrapsgn) Encode() []byte {
	return w.key.Encode()
}

func (w wrapsgn) Raw() []byte {
	return w.key.Raw()
}

func (w wrapsgn) Sign(msg []byte) signature.SignatureView {
	return w.key.Sign(msg)
}

func (w wrapsgn) SignatureAlgorithm() string {
	return w.key.SignatureAlgorithm()
}

func (w wrapsgn) SignatureCode() uint64 {
	return w.key.SignatureCode()
}

func (w wrapsgn) Unwrap() principal.Signer {
	return w.key
}

func (w wrapsgn) Verifier() principal.Verifier {
	return w.verifier
}

// Wrap the key of this signer into a signer with a different DID. This is
// primarily used to wrap a did:key signer with a signer that has a DID of
// a different method.
func Wrap(key principal.Signer, id did.DID) (WrappedSigner, error) {
	if !strings.HasPrefix(key.DID().String(), "did:key:") {
		return nil, fmt.Errorf("verifier is not a did:key")
	}
	vrf, err := verifier.Wrap(key.Verifier(), id)
	if err != nil {
		return nil, err
	}
	return wrapsgn{key, vrf}, nil
}

func Decode(encoded []byte) (principal.Signer, error) {
	code, err := varint.ReadUvarint(bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("reading signer codec: %w", err)
	}

	switch code {
	case ed25519signer.Code:
		return ed25519signer.Decode(encoded)
	case rsasigner.Code:
		return rsasigner.Decode(encoded)
	default:
		return nil, fmt.Errorf("unsupported signer codec: 0x%x", code)
	}
}

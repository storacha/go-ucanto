package verifier

import (
	"fmt"
	"strings"

	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-ucanto/ucan/crypto/signature"
)

type Unwrapper interface {
	// Unwrap returns the unwrapped did:key of this signer.
	Unwrap() principal.Verifier
}

type WrappedVerifier interface {
	principal.Verifier
	Unwrapper
}

type wrapvf struct {
	id  did.DID
	key principal.Verifier
}

func (w wrapvf) Code() uint64 {
	return w.key.Code()
}

func (w wrapvf) DID() did.DID {
	return w.id
}

func (w wrapvf) Encode() []byte {
	return w.key.Encode()
}

func (w wrapvf) Raw() []byte {
	return w.key.Raw()
}

func (w wrapvf) Verify(msg []byte, sig signature.Signature) bool {
	return w.key.Verify(msg, sig)
}

func (w wrapvf) Unwrap() principal.Verifier {
	return w.key
}

// Wrap the key of this verifier into a verifier with a different DID. This is
// primarily used to wrap a did:key verifier with a verifier that has a DID of
// a different method.
func Wrap(key principal.Verifier, id did.DID) (WrappedVerifier, error) {
	if !strings.HasPrefix(key.DID().String(), "did:key:") {
		return nil, fmt.Errorf("verifier is not a did:key")
	}
	return wrapvf{id, key}, nil
}

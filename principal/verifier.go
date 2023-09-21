package principal

import "github.com/alanshaw/go-ucanto/did"

type Verifier interface {
	Verify(payload []byte, signature Signature) bool
	DID() did.DID
}

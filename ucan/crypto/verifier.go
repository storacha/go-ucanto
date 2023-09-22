package crypto

import (
	"github.com/alanshaw/go-ucanto/ucan"
)

type Verifier interface {
	ucan.Principal
	Code() uint64
	// Takes byte encoded message and verifies that it is signed by corresponding
	// signer.
	Verify(msg []byte, signature Signature) bool
	Encode() []byte
}

package crypto

import (
	"github.com/alanshaw/go-ucanto/ucan"
)

type Signer interface {
	ucan.Principal
	// Takes byte encoded message and produces a verifiable signature.
	Sign(msg []byte) Signature
	Code() uint64
	Verifier() Verifier
	Encode() []byte
}

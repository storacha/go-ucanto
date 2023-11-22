package crypto

import (
	"github.com/alanshaw/go-ucanto/ucan"
	"github.com/alanshaw/go-ucanto/ucan/crypto/signature"
)

type Signer interface {
	ucan.Principal
	// Takes byte encoded message and produces a verifiable signature.
	Sign(msg []byte) signature.Signature
	Code() uint64
	Verifier() signature.Verifier
	Encode() []byte
}

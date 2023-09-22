package principal

import (
	"github.com/alanshaw/go-ucanto/ucan"
)

type Verifier interface {
	ucan.Principal
	Verify(payload []byte, signature Signature) bool
}

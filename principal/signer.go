package principal

import "github.com/alanshaw/go-ucanto/did"

type Signature interface {
	Bytes() []byte
	Code() uint64
	Size() int
}

type Signer interface {
	Sign(payload []byte) Signature
	DID() did.DID
	Code() uint64
	Verifier() Verifier
	Encode() []byte
}

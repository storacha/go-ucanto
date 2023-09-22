package principal

import "github.com/alanshaw/go-ucanto/ucan"

type Signature interface {
	Bytes() []byte
	Code() uint64
	Size() int
}

type Signer interface {
	ucan.Principal
	Sign(payload []byte) Signature
	Code() uint64
	Verifier() Verifier
	Encode() []byte
}

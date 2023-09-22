package crypto

import (
	"bytes"

	"github.com/multiformats/go-varint"
)

const EdDSA = 0xd0ed

type Signature interface {
	Code() uint64
	Size() uint64
	Bytes() []byte
	// Raw signature (without a signature algorithm info).
	Raw() []byte
}

func NewSignature(code uint64, raw []byte) Signature {
	cl := varint.UvarintSize(code)
	rl := varint.UvarintSize(uint64(len(raw)))
	sig := make(signature, cl+rl+len(raw))
	varint.PutUvarint(sig, code)
	varint.PutUvarint(sig[cl:], uint64(len(raw)))
	copy(sig[cl+rl:], raw)
	return sig
}

type signature []byte

func (s signature) Code() uint64 {
	c, _ := varint.ReadUvarint(bytes.NewReader(s))
	return c
}

func (s signature) Size() uint64 {
	n, _ := varint.ReadUvarint(bytes.NewReader(s[varint.UvarintSize(s.Code()):]))
	return n
}

func (s signature) Raw() []byte {
	cl := varint.UvarintSize(s.Code())
	rl := varint.UvarintSize(s.Size())
	return s[cl+rl:]
}

func (s signature) Bytes() []byte {
	return s
}

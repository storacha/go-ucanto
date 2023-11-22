package signature

import (
	"bytes"

	"github.com/multiformats/go-varint"
)

const EdDSA = 0xd0ed

type Signature interface {
	Code() uint64
	Size() uint64
	Bytes() []byte
	// Raw signature (without signature algorithm info).
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

func Encode(s Signature) []byte {
	return s.Bytes()
}

func Decode(b []byte) Signature {
	return signature(b)
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

type SignatureView interface {
	Signature
	// Verify that the signature was produced by the given message.
	Verify(msg []byte, signer Verifier) bool
}

func NewSignatureView(s Signature) SignatureView {
	return signatureView(signature(s.Bytes()))
}

type signatureView signature

func (v signatureView) Bytes() []byte {
	return signature(v).Bytes()
}

func (v signatureView) Code() uint64 {
	return signature(v).Code()
}

func (v signatureView) Raw() []byte {
	return signature(v).Raw()
}

func (v signatureView) Size() uint64 {
	return signature(v).Size()
}

func (v signatureView) Verify(msg []byte, signer Verifier) bool {
	return signer.Verify(msg, v)
}

package signature

import (
	"bytes"
	"fmt"

	"github.com/multiformats/go-varint"
)

const NON_STANDARD = 0xd000
const ES256K = 0xd0e7
const BLS12381G1 = 0xd0ea
const BLS12381G2 = 0xd0eb
const EdDSA = 0xd0ed
const ES256 = 0xd01200
const ES384 = 0xd01201
const ES512 = 0xd01202
const RS256 = 0xd01205
const EIP191 = 0xd191

func CodeName(code uint64) (string, error) {
	switch code {
	case ES256K:
		return "ES256K", nil
	case BLS12381G1:
		return "BLS12381G1", nil
	case BLS12381G2:
		return "BLS12381G2", nil
	case EdDSA:
		return "EdDSA", nil
	case ES256:
		return "ES256", nil
	case ES384:
		return "ES384", nil
	case ES512:
		return "ES512", nil
	case RS256:
		return "RS256", nil
	case EIP191:
		return "EIP191", nil
	default:
		return "", fmt.Errorf("unknown signature algorithm code 0x%x", code)
	}
}

func NameCode(name string) (uint64, error) {
	switch name {
	case "ES256K":
		return ES256K, nil
	case "BLS12381G1":
		return BLS12381G1, nil
	case "BLS12381G2":
		return BLS12381G2, nil
	case "EdDSA":
		return EdDSA, nil
	case "ES256":
		return ES256, nil
	case "ES384":
		return ES384, nil
	case "ES512":
		return ES512, nil
	case "RS256":
		return RS256, nil
	case "EIP191":
		return EIP191, nil
	default:
		return NON_STANDARD, nil
	}
}

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

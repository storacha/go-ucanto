package verifier

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"fmt"

	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-ucanto/principal/multiformat"
	"github.com/storacha/go-ucanto/ucan/crypto/signature"
)

const Code = 0x1205
const Name = "RSA"

const SignatureCode = signature.RS256
const SignatureAlgorithm = "RS256"

func Parse(str string) (principal.Verifier, error) {
	did, err := did.Parse(str)
	if err != nil {
		return nil, fmt.Errorf("parsing DID: %s", err)
	}
	return Decode(did.Bytes())
}

func Decode(b []byte) (principal.Verifier, error) {
	utb, err := multiformat.UntagWith(Code, b, 0)
	if err != nil {
		return nil, err
	}

	pub, err := x509.ParsePKCS1PublicKey(utb)
	if err != nil {
		return nil, fmt.Errorf("parsing public key: %s", err)
	}

	return rsaverifier{bytes: b, pubKey: pub}, nil
}

type rsaverifier struct {
	bytes  []byte
	pubKey *rsa.PublicKey
}

func (v rsaverifier) Code() uint64 {
	return Code
}

func (v rsaverifier) Verify(msg []byte, sig signature.Signature) bool {
	if sig.Code() != signature.RS256 {
		return false
	}

	hash := sha256.New()
	hash.Write(msg)
	digest := hash.Sum(nil)

	err := rsa.VerifyPKCS1v15(v.pubKey, crypto.SHA256, digest, sig.Raw())
	return err == nil
}

func (v rsaverifier) DID() did.DID {
	id, _ := did.Decode(v.bytes)
	return id
}

func (v rsaverifier) Encode() []byte {
	return v.bytes
}

func (v rsaverifier) Raw() []byte {
	b, _ := multiformat.UntagWith(Code, v.bytes, 0)
	return b
}

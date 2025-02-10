package signer

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"fmt"

	"github.com/multiformats/go-multibase"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-ucanto/principal/multiformat"
	"github.com/storacha/go-ucanto/principal/rsa/verifier"
	"github.com/storacha/go-ucanto/ucan/crypto/signature"
)

const Code = 0x1305
const Name = verifier.Name

const SignatureCode = verifier.SignatureCode
const SignatureAlgorithm = verifier.SignatureAlgorithm

const keySize = 2048

func Generate() (principal.Signer, error) {
	priv, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, fmt.Errorf("generating RSA key: %s", err)
	}

	// Next we need to encode public key, because `RSAVerifier` uses it to
	// for implementing the `DID()` method.
	pubbytes := multiformat.TagWith(verifier.Code, x509.MarshalPKCS1PublicKey(&priv.PublicKey))

	verif, err := verifier.Decode(pubbytes)
	if err != nil {
		return nil, fmt.Errorf("decoding public bytes: %s", err)
	}

	// Export key in Private Key Cryptography Standards (PKCS) format and extract
	// the bytes corresponding to the private key, which we tag with RSA private
	// key multiformat code. With both binary and actual key representation we
	// create a RSASigner view.
	prvbytes := multiformat.TagWith(Code, x509.MarshalPKCS1PrivateKey(priv))

	return RSASigner{bytes: prvbytes, privKey: priv, verifier: verif}, nil
}

func Parse(str string) (principal.Signer, error) {
	_, bytes, err := multibase.Decode(str)
	if err != nil {
		return nil, fmt.Errorf("decoding multibase string: %s", err)
	}
	return Decode(bytes)
}

func Format(signer principal.Signer) (string, error) {
	return multibase.Encode(multibase.Base64pad, signer.Encode())
}

func Decode(b []byte) (principal.Signer, error) {
	utb, err := multiformat.UntagWith(Code, b, 0)
	if err != nil {
		return nil, err
	}

	priv, err := x509.ParsePKCS1PrivateKey(utb)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %s", err)
	}

	pubbytes := multiformat.TagWith(verifier.Code, x509.MarshalPKCS1PublicKey(&priv.PublicKey))

	verif, err := verifier.Decode(pubbytes)
	if err != nil {
		return nil, fmt.Errorf("decoding public bytes: %s", err)
	}

	return RSASigner{bytes: b, privKey: priv, verifier: verif}, nil
}

// FromRaw takes raw RSA private key in PKCS #1, ASN.1 DER form and tags with
// the RSA signer and verifier multiformat codes, returning an RSA signer.
func FromRaw(b []byte) (principal.Signer, error) {
	tb := multiformat.TagWith(Code, b)
	priv, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %s", err)
	}
	verif, err := verifier.FromRaw(x509.MarshalPKCS1PublicKey(&priv.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("decoding public bytes: %s", err)
	}
	return RSASigner{bytes: tb, privKey: priv, verifier: verif}, nil
}

type RSASigner struct {
	bytes    []byte
	privKey  *rsa.PrivateKey
	verifier principal.Verifier
}

func (s RSASigner) Code() uint64 {
	return Code
}

func (s RSASigner) SignatureCode() uint64 {
	return SignatureCode
}

func (s RSASigner) SignatureAlgorithm() string {
	return SignatureAlgorithm
}

func (s RSASigner) Verifier() principal.Verifier {
	return s.verifier
}

func (s RSASigner) DID() did.DID {
	return s.verifier.DID()
}

func (s RSASigner) Encode() []byte {
	return s.bytes
}

func (s RSASigner) Raw() []byte {
	b, _ := multiformat.UntagWith(Code, s.bytes, 0)
	return b
}

func (s RSASigner) Sign(msg []byte) signature.SignatureView {
	hash := sha256.New()
	hash.Write(msg)
	digest := hash.Sum(nil)
	sig, _ := rsa.SignPKCS1v15(nil, s.privKey, crypto.SHA256, digest)
	return signature.NewSignatureView(signature.NewSignature(SignatureCode, sig))
}

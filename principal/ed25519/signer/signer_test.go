package signer

import (
	"crypto/ed25519"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateEncodeDecode(t *testing.T) {
	s0, err := Generate()
	if err != nil {
		t.Fatalf("generating Ed25519 key: %v", err)
	}

	fmt.Println(s0.DID().String())

	s1, err := Decode(s0.Encode())
	if err != nil {
		t.Fatalf("decoding Ed25519 key: %v", err)
	}

	fmt.Println(s1.DID().String())

	if s0.DID().String() != s1.DID().String() {
		t.Fatalf("public key mismatch: %s != %s", s0.DID().String(), s1.DID().String())
	}
}

func TestGenerateFormatParse(t *testing.T) {
	s0, err := Generate()
	if err != nil {
		t.Fatalf("generating Ed25519 key: %v", err)
	}

	fmt.Println(s0.DID().String())

	str, err := Format(s0)
	if err != nil {
		t.Fatalf("formatting Ed25519 key: %v", err)
	}

	fmt.Println(str)

	s1, err := Parse(str)
	if err != nil {
		t.Fatalf("parsing Ed25519 key: %v", err)
	}

	fmt.Println(s1.DID().String())

	if s0.DID().String() != s1.DID().String() {
		t.Fatalf("public key mismatch: %s != %s", s0.DID().String(), s1.DID().String())
	}
}

func TestVerify(t *testing.T) {
	s0, err := Generate()
	if err != nil {
		t.Fatalf("generating Ed25519 key: %v", err)
	}

	msg := []byte("testy")
	sig := s0.Sign(msg)

	res := s0.Verifier().Verify(msg, sig)
	if res != true {
		t.Fatalf("verify failed")
	}
}

func TestSignerRaw(t *testing.T) {
	s, err := Generate()
	require.NoError(t, err)

	msg := []byte{1, 2, 3}
	raw := s.Raw()
	sig := ed25519.Sign(raw, msg)

	require.Equal(t, s.Sign(msg).Raw(), sig)
}

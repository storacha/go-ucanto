package signer

import (
	"fmt"
	"testing"
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

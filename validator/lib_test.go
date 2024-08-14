package validator

import (
	"testing"

	"github.com/storacha-network/go-ucanto/principal/ed25519/signer"
	"github.com/storacha-network/go-ucanto/ucan"
)

func TestIsSelfIssued(t *testing.T) {
	alice, err := signer.Generate()
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}
	bob, err := signer.Generate()
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	cap := ucan.NewCapability("upload/add", alice.DID().String(), struct{}{})

	canIssue := IsSelfIssued(cap, alice.DID())
	if canIssue == false {
		t.Fatal("capability self issued by alice")
	}

	canIssue = IsSelfIssued(cap, bob.DID())
	if canIssue == true {
		t.Fatal("capability not self issued by bob")
	}
}

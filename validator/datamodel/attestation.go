package datamodel

import (
	_ "embed"
	"fmt"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed errors.ipldsch
var attestationsch []byte
var attestationTypeSystem *schema.TypeSystem

func init() {
	ts, err := ipld.LoadSchemaBytes(attestationsch)
	if err != nil {
		panic(fmt.Errorf("failed to load IPLD schema: %s", err))
	}
	attestationTypeSystem = ts
}

func AttestationType() schema.Type {
	return attestationTypeSystem.TypeByName("Attestation")
}

type AttestationModel struct {
	Proof ipld.Link
}
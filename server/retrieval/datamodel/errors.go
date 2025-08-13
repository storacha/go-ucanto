package datamodel

import (
	// for go:embed
	_ "embed"
	"fmt"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed errors.ipldsch
var errorsch []byte

var (
	errorTypeSystem *schema.TypeSystem
)

func init() {
	ts, err := ipld.LoadSchemaBytes(errorsch)
	if err != nil {
		panic(fmt.Errorf("failed to load IPLD schema: %w", err))
	}
	errorTypeSystem = ts
}

func Schema() []byte {
	return errorsch
}

func MissingProofsType() schema.Type {
	return errorTypeSystem.TypeByName("MissingProofs")
}

type MissingProofsModel struct {
	Name    string
	Message string
	Proofs  []ipld.Link
}

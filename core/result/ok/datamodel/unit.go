package datamodel

import (
	// to use go:embed
	_ "embed"
	"fmt"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed unit.ipldsch
var unitSchema []byte

var unitType schema.Type

func init() {
	ts, err := ipld.LoadSchemaBytes(unitSchema)
	if err != nil {
		panic(fmt.Errorf("loading unit schema: %w", err))
	}
	unitType = ts.TypeByName("Unit")
}

func UnitType() schema.Type {
	return unitType
}

func Schema() []byte {
	return unitSchema
}

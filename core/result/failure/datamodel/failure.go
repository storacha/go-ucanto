package datamodel

import (
	// to use go:embed
	_ "embed"
	"fmt"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
	ucanipld "github.com/storacha/go-ucanto/core/ipld"
)

//go:embed failure.ipldsch
var failureSchema []byte

// FailureModel is a generic failure
type FailureModel struct {
	Name    *string
	Message string
	Stack   *string
}

func (f *FailureModel) ToIPLD() (ipld.Node, error) {
	return ucanipld.WrapWithRecovery(f, typ)
}

var typ schema.Type

func init() {
	ts, err := ipld.LoadSchemaBytes(failureSchema)
	if err != nil {
		panic(fmt.Errorf("loading failure schema: %w", err))
	}
	typ = ts.TypeByName("Failure")
}

func FailureType() schema.Type {
	return typ
}

func Schema() []byte {
	return failureSchema
}

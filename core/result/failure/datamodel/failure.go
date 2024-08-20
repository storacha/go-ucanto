package datamodel

import (
	// to use go:embed
	_ "embed"
	"fmt"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
	ucanipld "github.com/storacha-network/go-ucanto/core/ipld"
)

//go:embed failure.ipldsch
var failureSchema []byte

// Failure is a generic failure
type Failure struct {
	Name    *string
	Message string
	Stack   *string
}

func (f *Failure) Build() (ipld.Node, error) {
	return ucanipld.WrapWithRecovery(f, typ)
}

var (
	typ schema.Type
)

func init() {
	ts, err := ipld.LoadSchemaBytes(failureSchema)
	if err != nil {
		panic(fmt.Errorf("loading failure schema: %w", err))
	}
	typ = ts.TypeByName("Failure")
}
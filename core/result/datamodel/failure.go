package datamodel

import (
	// to use go:embed
	_ "embed"
	"fmt"
	"sync"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed failure.ipldsch
var failureSchema []byte

// Failure is a generic failure
type Failure struct {
	Name    *string
	Message string
	Stack   *string
}

var (
	once sync.Once
	ts   *schema.TypeSystem
	err  error
)

func mustLoadSchema() *schema.TypeSystem {
	once.Do(func() {
		ts, err = ipld.LoadSchemaBytes(failureSchema)
	})
	if err != nil {
		panic(fmt.Errorf("failed to load IPLD schema: %s", err))
	}
	return ts
}

// returns the failure schematype
func Type() schema.Type {
	return mustLoadSchema().TypeByName("Failure")
}

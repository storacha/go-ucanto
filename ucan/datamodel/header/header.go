package header

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed header.ipldsch
var headersch []byte

var (
	once sync.Once
	ts   *schema.TypeSystem
	err  error
)

func mustLoadSchema() *schema.TypeSystem {
	once.Do(func() {
		ts, err = ipld.LoadSchemaBytes(headersch)
	})
	if err != nil {
		panic(fmt.Errorf("failed to load IPLD schema: %w", err))
	}
	return ts
}

func Type() schema.Type {
	return mustLoadSchema().TypeByName("Header")
}

type HeaderModel struct {
	Alg string
	Ucv string
	Typ string
}

package ucan

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed ucan.ipldsch
var ucansch []byte

var (
	once sync.Once
	ts   *schema.TypeSystem
	err  error
)

func mustLoadSchema() *schema.TypeSystem {
	once.Do(func() {
		ts, err = ipld.LoadSchemaBytes(ucansch)
	})
	if err != nil {
		panic(fmt.Errorf("failed to load IPLD schema: %s", err))
	}
	return ts
}

func Type() schema.Type {
	return mustLoadSchema().TypeByName("UCAN")
}

type UCANModel struct {
	V   string
	Iss []byte
	Aud []byte
	S   []byte
	Att []CapabilityModel
	Prf []ipld.Link
	Exp uint64
	Fct []FactModel
	Nnc *string
	Nbf *uint64
}

type CapabilityModel struct {
	With string
	Can  string
	Nb   datamodel.Node
}

type FactModel struct {
	Keys   []string
	Values map[string]datamodel.Node
}

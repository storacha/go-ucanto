package payload

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed payload.ipldsch
var payloadsch []byte

var (
	once sync.Once
	ts   *schema.TypeSystem
	err  error
)

func mustLoadSchema() *schema.TypeSystem {
	once.Do(func() {
		ts, err = ipld.LoadSchemaBytes(payloadsch)
	})
	if err != nil {
		panic(fmt.Errorf("failed to load IPLD schema: %s", err))
	}
	return ts
}

func Type() schema.Type {
	return mustLoadSchema().TypeByName("Payload")
}

type PayloadModel struct {
	Iss []string
	Aud []string
	Att []CapabilityModel
	Prf []string
	Exp uint64
	Fct []FactModel
	Nnc string
	Nbf uint64
}

type CapabilityModel struct {
	With string
	Can  string
	Nb   NbModel
}

type NbModel struct {
	Keys   []string
	Values map[string]datamodel.Node
}

type FactModel struct {
	Keys   []string
	Values map[string]datamodel.Node
}

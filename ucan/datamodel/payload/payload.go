package payload

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
	udm "github.com/storacha/go-ucanto/ucan/datamodel/ucan"
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
		panic(fmt.Errorf("failed to load IPLD schema: %w", err))
	}
	return ts
}

func Type() schema.Type {
	return mustLoadSchema().TypeByName("Payload")
}

type PayloadModel struct {
	Iss string
	Aud string
	Att []udm.CapabilityModel
	Prf []string
	Exp *int
	Fct []udm.FactModel
	Nnc *string
	Nbf *int
}

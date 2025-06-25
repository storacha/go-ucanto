package datamodel

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/schema"
	schemadmt "github.com/ipld/go-ipld-prime/schema/dmt"
	schemadsl "github.com/ipld/go-ipld-prime/schema/dsl"
)

//go:embed receipt.ipldsch
var receipt []byte

//go:embed anyresult.ipldsch
var anyResultSchema []byte

var anyReceiptTs *schema.TypeSystem

func init() {
	ts, err := NewReceiptModelType(anyResultSchema)
	if err != nil {
		panic(fmt.Errorf("failed to load IPLD schema: %w", err))
	}
	anyReceiptTs = ts.TypeSystem()
}

func TypeSystem() *schema.TypeSystem {
	return anyReceiptTs
}

type ReceiptModel[O any, X any] struct {
	Ocm OutcomeModel[O, X]
	Sig []byte
}

type OutcomeModel[O any, X any] struct {
	Ran  ipld.Link
	Out  ResultModel[O, X]
	Fx   EffectsModel
	Meta MetaModel
	Iss  *string
	Prf  []ipld.Link
}

type EffectsModel struct {
	Fork []ipld.Link
	Join ipld.Link
}

type MetaModel struct {
	Keys   []string
	Values map[string]datamodel.Node
}

type ResultModel[O any, X any] struct {
	Ok    *O
	Error *X
}

// NewReceiptModelType creates a new schema.Type for a Receipt. You must
// provide the schema containing a Result type, which is a keyed union. e.g.
//
//	type Result union {
//	  | Ok "ok"
//	  | Err "error"
//	} representation keyed
//
//	type Ok struct {
//	  status String (rename "Status")
//	}
//
//	type Err struct {
//	  message String (rename "Message")
//	}
func NewReceiptModelType(resultschema []byte) (schema.Type, error) {
	sch := bytes.Join([][]byte{resultschema, receipt}, []byte("\n"))
	ts, err := ipld.LoadSchemaBytes(sch)
	if err != nil {
		return nil, err
	}
	return ts.TypeByName("Receipt"), nil
}

func NewReceiptModelFromTypes(successType schema.Type, errType schema.Type) (schema.Type, error) {
	ts := new(schema.TypeSystem)
	ts.Init()
	schema.SpawnDefaultBasicTypes(ts)
	schema.MergeTypeSystem(ts, successType.TypeSystem(), true)
	schema.MergeTypeSystem(ts, errType.TypeSystem(), true)
	ts.Accumulate(schema.SpawnUnion("Result", []schema.TypeName{successType.Name(), errType.Name()}, schema.SpawnUnionRepresentationKeyed(map[string]schema.TypeName{"ok": successType.Name(), "error": errType.Name()})))
	sch, err := schemadsl.Parse("", bytes.NewReader(receipt))
	if err != nil {
		return nil, err
	}
	schemadmt.SpawnSchemaTypes(ts, sch)
	if errs := ts.ValidateGraph(); errs != nil {
		return nil, errors.Join(errs...)
	}
	return ts.TypeByName("Receipt"), nil
}

package datamodel

import (
	"bytes"
	_ "embed"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed receipt.ipldsch
var receipt []byte

type ReceiptModel[O any, X any] struct {
	Ocm OutcomeModel[O, X]
	Sig []byte
}

type OutcomeModel[O any, X any] struct {
	Ran  ipld.Link
	Out  *ResultModel[O, X]
	Fx   EffectsModel
	Meta MetaModel
	Iss  []byte
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
	Ok  O
	Err X
}

// NewReceiptModelType creates a new schema.Type for a Receipt. You must provide the
// schema containing a Result type, which is a keyed union. e.g.
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

package receipt

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/bindnode"
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

// NewReceiptType creates a new schema.Type for a Receipt. You must provide the
// schema containing a Result type, which is a a keyed union. e.g.
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
func NewReceiptType(resultschema []byte) (schema.Type, error) {
	sch := bytes.Join([][]byte{resultschema, receipt}, []byte("\n"))
	ts, err := ipld.LoadSchemaBytes(sch)
	if err != nil {
		return nil, err
	}
	return ts.TypeByName("Receipt"), nil
}

func Encode[O any, X any](r *ReceiptModel[O, X], typ schema.Type) ([]byte, error) {
	node := bindnode.Wrap(r, typ)
	var buf bytes.Buffer
	err := dagcbor.Encode(node.Representation(), &buf)
	if err != nil {
		return nil, fmt.Errorf("encoding dag-cbor: %s", err)
	}
	return buf.Bytes(), nil
}

func Decode[O any, X any](b []byte, typ schema.Type) (*ReceiptModel[O, X], error) {
	r := ReceiptModel[O, X]{}
	_, err := ipld.Unmarshal(b, dagcbor.Decode, &r, typ)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

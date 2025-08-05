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

func (f FailureModel) Error() string {
	return f.Message
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

// Bind binds the IPLD node to a [datamodel.FailureModel]. This works around
// IPLD requiring data to match the schema _exactly_.
//
// Note: the IPLD node is expected to be a map kind, with a "message" key and
// optionally a "name" and "stack" (all values strings). If none of these are
// true then you get back a nil value [datamodel.FailureModel].
func Bind(n ipld.Node) FailureModel {
	f := FailureModel{}
	nn, err := n.LookupByString("name")
	if err == nil {
		name, err := nn.AsString()
		if err == nil {
			f.Name = &name
		}
	}
	mn, err := n.LookupByString("message")
	if err == nil {
		msg, err := mn.AsString()
		if err == nil {
			f.Message = msg
		}
	}
	sn, err := n.LookupByString("stack")
	if err == nil {
		stack, err := sn.AsString()
		if err == nil {
			f.Stack = &stack
		}
	}
	return f
}

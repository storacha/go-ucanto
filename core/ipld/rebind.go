package ipld

import (
	"errors"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
)

// Rebind takes a Node and binds it to the Go type according to the passed schema.
func Rebind[T any](nd datamodel.Node, typ schema.Type) (ptrVal T, err error) {
	defer func() {
		if r := recover(); r != nil {
			if asStr, ok := r.(string); ok {
				err = errors.New(asStr)
			} else if asErr, ok := r.(error); ok {
				err = asErr
			} else {
				err = errors.New("unknown panic rebinding node")
			}
		}
	}()

	if typedNode, ok := nd.(schema.TypedNode); ok {
		nd = typedNode.Representation()
	}

	var nilbind T
	np := bindnode.Prototype(&nilbind, typ)
	nb := np.Representation().NewBuilder()
	err = nb.AssignNode(nd)
	if err != nil {
		return
	}
	rnd := nb.Build()
	ptrVal = *bindnode.Unwrap(rnd).(*T)
	return
}

package ipld

import (
	"errors"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/storacha/go-ucanto/core/ipld/block"
)

type Link = ipld.Link
type Block = block.Block
type Node = ipld.Node

// Builder can be modeled as an IPLD data and provides a `ToIPLD` method to
// build itself into a `datamodel.Node`.
type Builder interface {
	ToIPLD() (Node, error)
}

// WrapWithRecovery behaves like bindnode.Wrap but converts panics into errors
func WrapWithRecovery(ptrVal interface{}, typ schema.Type, opts ...bindnode.Option) (nd Node, err error) {
	defer func() {
		if r := recover(); r != nil {
			if asStr, ok := r.(string); ok {
				err = errors.New(asStr)
			} else if asErr, ok := r.(error); ok {
				err = asErr
			} else {
				err = errors.New("unknown panic building node")
			}
		}
	}()
	nd = bindnode.Wrap(ptrVal, typ, opts...)
	return
}

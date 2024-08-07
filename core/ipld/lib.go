package ipld

import (
	"github.com/ipld/go-ipld-prime"
	"github.com/storacha-network/go-ucanto/core/ipld/block"
)

type Link = ipld.Link
type Block = block.Block
type Node = ipld.Node

// Builder can be modeled as an IPLD data and provides a `Build“ method to
// build itself into a `datamodel.Node`.
type Builder interface {
	Build() (Node, error)
}

package ipld

import (
	"github.com/ipld/go-ipld-prime"
	"github.com/web3-storage/go-ucanto/core/ipld/block"
)

type Link = ipld.Link
type Block = block.Block
type Node = ipld.Node

// Datamodeler describes an object that can be modeled as IPLD data.
type Datamodeler interface {
	ToIPLD() Node
}

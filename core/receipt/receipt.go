package receipt

import (
	"github.com/alanshaw/go-ucanto/core/dag/blockstore"
	"github.com/alanshaw/go-ucanto/core/ipld"
)

type Receipt interface{}

func NewReceipt(root ipld.Link, blocks blockstore.BlockReader) (Receipt, error) {
	return nil, nil
}

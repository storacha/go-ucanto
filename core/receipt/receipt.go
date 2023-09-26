package receipt

import (
	"github.com/alanshaw/go-ucanto/core/dag"
	"github.com/alanshaw/go-ucanto/core/ipld"
)

type Receipt interface{}

func NewReceipt(root ipld.Link, bs dag.BlockStore) (Receipt, error) {
	return nil, nil
}

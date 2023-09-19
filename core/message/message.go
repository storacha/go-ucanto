package message

import (
	"fmt"

	"github.com/alanshaw/go-ucanto/core"
	"github.com/alanshaw/go-ucanto/core/dag"
	"github.com/alanshaw/go-ucanto/core/dag/cbor"
	"github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/alanshaw/go-ucanto/core/iterable"
)

type AgentMessage interface {
	ipld.IPLDView
	Invocations() []core.Invocation
	Receipts() []core.Receipt
	Get(link ipld.Link) (core.Receipt, error)
}

type message struct {
	blks dag.BlockStore
}

func (m *message) Blocks() iterable.Iterator[ipld.Block] {
	return m.blks.Iterator()
}

func Build(invocation core.Invocation, receipt core.Receipt) (AgentMessage, error) {
	return nil, fmt.Errorf("not implemented")
}

func NewMessage(roots []ipld.Link, blocks iterable.Iterator[ipld.Block]) (AgentMessage, error) {
	bstore, err := dag.NewBlockStore(blocks)
	if err != nil {
		return nil, fmt.Errorf("creating blockstore: %s", err)
	}

	if len(roots) == 0 {
		return nil, fmt.Errorf("missing roots")
	}

	rblock, ok := bstore.Get(roots[0])
	if !ok {
		return nil, fmt.Errorf("missing root block: %s", roots[0])
	}

	data, err := cbor.Decode(rblock.Bytes())

	return &message{blks: bstore}, nil
}

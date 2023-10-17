package message

import (
	"fmt"

	"github.com/alanshaw/go-ucanto/core/dag/blockstore"
	"github.com/alanshaw/go-ucanto/core/dag/cbor"
	"github.com/alanshaw/go-ucanto/core/invocation"
	"github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/alanshaw/go-ucanto/core/iterable"
	"github.com/alanshaw/go-ucanto/core/message/datamodel/agentmessage"
	"github.com/alanshaw/go-ucanto/core/receipt"
)

type AgentMessage interface {
	ipld.IPLDView
	Invocations() ([]invocation.Invocation, error)
	Receipts() ([]receipt.Receipt, error)
	Get(link ipld.Link) (receipt.Receipt, bool, error)
}

type message struct {
	root ipld.Block
	data *agentmessage.Data
	blks blockstore.BlockReader
}

func (m *message) Root() ipld.Block {
	return m.root
}

func (m *message) Blocks() iterable.Iterator[ipld.Block] {
	return m.blks.Iterator()
}

func (m *message) Invocations() ([]invocation.Invocation, error) {
	var invs []invocation.Invocation
	for _, l := range m.data.Execute {
		inv, err := invocation.NewInvocation(l, m.blks)
		if err != nil {
			return nil, err
		}
		invs = append(invs, inv)
	}
	return invs, nil
}

func (m *message) Receipts() ([]receipt.Receipt, error) {
	var rcpts []receipt.Receipt
	for _, k := range m.data.Report.Keys {
		l, _ := m.data.Report.Values[k]
		rcpt, err := receipt.NewReceipt(l, m.blks)
		if err != nil {
			return nil, err
		}
		rcpts = append(rcpts, rcpt)
	}
	return rcpts, nil
}

func (m *message) Get(link ipld.Link) (receipt.Receipt, bool, error) {
	r, ok := m.data.Report.Values[link.String()]
	if !ok {
		return nil, false, nil
	}
	rcpt, err := receipt.NewReceipt(r, m.blks)
	if err != nil {
		return nil, false, err
	}
	return rcpt, true, nil
}

func Build(invocation invocation.Invocation) (AgentMessage, error) {
	iblks, err := iterable.Collect(invocation.Blocks())
	if err != nil {
		return nil, err
	}
	bs, err := blockstore.NewBlockStore(blockstore.WithBlocks(iblks))

	ex := []ipld.Link{}
	for _, ib := range iblks {
		ex = append(ex, ib.Link())
	}

	data := agentmessage.Data{Execute: ex}
	m, err := agentmessage.Encode(&data)
	if err != nil {
		return nil, err
	}

	rt, err := cbor.NewBlock(m)
	if err != nil {
		return nil, err
	}
	err = bs.Put(rt)
	if err != nil {
		return nil, err
	}

	return &message{root: rt, data: &data, blks: bs}, nil
}

func NewMessage(roots []ipld.Link, blks blockstore.BlockReader) (AgentMessage, error) {
	if len(roots) == 0 {
		return nil, fmt.Errorf("missing roots")
	}

	rblock, ok, err := blks.Get(roots[0])
	if err != nil {
		return nil, fmt.Errorf("getting root block: %s", err)
	}
	if !ok {
		return nil, fmt.Errorf("missing root block: %s", roots[0])
	}

	data, err := agentmessage.Decode(rblock.Bytes())
	if err != nil {
		return nil, fmt.Errorf("decoding message: %s", err)
	}

	return &message{root: rblock, data: data, blks: blks}, nil
}

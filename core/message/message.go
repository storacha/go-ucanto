package message

import (
	"bytes"
	"fmt"
	"io"

	"github.com/alanshaw/go-ucanto/core/dag"
	"github.com/alanshaw/go-ucanto/core/invocation"
	"github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/alanshaw/go-ucanto/core/iterable"
	"github.com/alanshaw/go-ucanto/core/message/schema/agentmessage"
	"github.com/alanshaw/go-ucanto/core/receipt"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/node/bindnode"
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
	bs   dag.BlockStore
}

func (m *message) Root() ipld.Block {
	return m.root
}

func (m *message) Blocks() iterable.Iterator[ipld.Block] {
	return m.bs.Iterator()
}

func (m *message) Invocations() ([]invocation.Invocation, error) {
	var invs []invocation.Invocation
	for _, l := range m.data.Execute {
		inv, err := invocation.NewInvocation(l, m.bs)
		if err != nil {
			return nil, err
		}
		invs = append(invs, inv)
	}
	return invs, nil
}

func (m *message) Receipts() ([]receipt.Receipt, error) {
	var rcpts []receipt.Receipt
	for _, l := range m.data.Report {
		rcpt, err := receipt.NewReceipt(l, m.bs)
		if err != nil {
			return nil, err
		}
		rcpts = append(rcpts, rcpt)
	}
	return rcpts, nil
}

func (m *message) Get(link ipld.Link) (receipt.Receipt, bool, error) {
	r, ok := m.data.Report[link.String()]
	if !ok {
		return nil, false, nil
	}
	rcpt, err := receipt.NewReceipt(r, m.bs)
	if err != nil {
		return nil, false, err
	}
	return rcpt, true, nil
}

func Build(invocation invocation.Invocation) (AgentMessage, error) {
	var rt ipld.Block
	it := invocation.Blocks()
	data := agentmessage.Data{}
	bs := dag.NewBlockStore(iterable.NewIterator[ipld.Block](func() (ipld.Block, error) {
		b, err := it.Next()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
		}

	}))
	return &message{root: rt, data: &data, bs: bs}, nil
}

func NewMessage(roots []ipld.Link, bs dag.BlockStore) (AgentMessage, error) {
	if len(roots) == 0 {
		return nil, fmt.Errorf("missing roots")
	}

	rblock, ok := bs.Get(roots[0])
	if !ok {
		return nil, fmt.Errorf("missing root block: %s", roots[0])
	}

	data, err := decodeAgentMessageModel(rblock.Bytes())
	if err != nil {
		return nil, fmt.Errorf("decoding message: %s", err)
	}

	return &message{root: rblock, data: data, bs: bs}, nil
}

func decodeAgentMessageModel(b []byte) (*agentmessage.Data, error) {
	ts, err := agentmessage.LoadSchema()
	if err != nil {
		return nil, err
	}
	npt := bindnode.Prototype((*agentmessage.Data)(nil), ts.TypeByName("AgentMessageModel"))
	nb := npt.Representation().NewBuilder()
	err = dagcbor.Decode(nb, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	data := bindnode.Unwrap(nb.Build()).(*agentmessage.Data)
	return data, nil
}

package message

import (
	"fmt"
	"iter"

	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/ipld/block"
	"github.com/storacha/go-ucanto/core/ipld/codec/cbor"
	"github.com/storacha/go-ucanto/core/ipld/hash/sha256"
	mdm "github.com/storacha/go-ucanto/core/message/datamodel"
	"github.com/storacha/go-ucanto/core/receipt"
)

type AgentMessage interface {
	ipld.View
	// Invocations is a list of links to the root block of invocations than can
	// be found in the message.
	Invocations() []ipld.Link
	// Get a given invocation from the message by root CID.
	Invocation(root ipld.Link) (invocation.Invocation, bool, error)
	// Receipts is a list of links to the root block of receipts that can be
	// found in the message.
	Receipts() []ipld.Link
	// Get a given receipt from the message by root CID.
	Receipt(root ipld.Link) (receipt.AnyReceipt, bool, error)
	// Get returns a receipt link from the message, given an invocation link.
	Get(link ipld.Link) (ipld.Link, bool)
}

type message struct {
	root ipld.Block
	data *mdm.DataModel
	blks blockstore.BlockReader
	// invs is cache of invocations decoded from the message
	invs map[string]invocation.Invocation
	// rcpts is a cache of receipts decoded from the message
	rcpts map[string]receipt.AnyReceipt
}

var _ AgentMessage = (*message)(nil)

func (m *message) Root() ipld.Block {
	return m.root
}

func (m *message) Blocks() iter.Seq2[ipld.Block, error] {
	return m.blks.Iterator()
}

func (m *message) Invocations() []ipld.Link {
	return m.data.Execute
}

func (m *message) Invocation(root ipld.Link) (invocation.Invocation, bool, error) {
	if inv, ok := m.invs[root.String()]; ok {
		return inv, true, nil
	}
	rtBlk, ok, err := m.blks.Get(root)
	if !ok || err != nil {
		return nil, ok, err
	}
	inv, err := invocation.NewInvocation(rtBlk, m.blks)
	if err != nil {
		return nil, false, err
	}
	m.invs[root.String()] = inv
	return inv, true, nil
}

func (m *message) Receipts() []ipld.Link {
	var rcpts []ipld.Link
	if m.data.Report == nil {
		return rcpts
	}
	for _, k := range m.data.Report.Keys {
		l, ok := m.data.Report.Values[k]
		if ok {
			rcpts = append(rcpts, l)
		}
	}
	return rcpts
}

func (m *message) Receipt(root ipld.Link) (receipt.AnyReceipt, bool, error) {
	if rcpt, ok := m.rcpts[root.String()]; ok {
		return rcpt, true, nil
	}
	_, ok, err := m.blks.Get(root)
	if !ok || err != nil {
		return nil, ok, err
	}
	rcpt, err := receipt.NewAnyReceipt(root, m.blks)
	if err != nil {
		return nil, false, err
	}
	m.rcpts[root.String()] = rcpt
	return rcpt, true, nil
}

func (m *message) Get(link ipld.Link) (ipld.Link, bool) {
	var rcpt ipld.Link
	found := false
	for _, k := range m.data.Report.Keys {
		if k == link.String() {
			rcpt = m.data.Report.Values[k]
			found = true
			break
		}
	}
	if !found {
		return nil, false
	}
	return rcpt, true
}

func Build(invocations []invocation.Invocation, receipts []receipt.AnyReceipt) (AgentMessage, error) {
	bs, err := blockstore.NewBlockStore()
	if err != nil {
		return nil, err
	}

	ex := []ipld.Link{}
	invCache := map[string]invocation.Invocation{}
	for _, inv := range invocations {
		ex = append(ex, inv.Link())
		invCache[inv.Link().String()] = inv

		err := blockstore.WriteInto(inv, bs)
		if err != nil {
			return nil, err
		}
	}

	var report *mdm.ReportModel
	rcptCache := map[string]receipt.AnyReceipt{}
	if len(receipts) > 0 {
		report = &mdm.ReportModel{
			Keys:   make([]string, 0, len(receipts)),
			Values: make(map[string]ipld.Link, len(receipts)),
		}
		for _, receipt := range receipts {
			err := blockstore.WriteInto(receipt, bs)
			if err != nil {
				return nil, err
			}
			rcptCache[receipt.Root().Link().String()] = receipt

			key := receipt.Ran().Link().String()
			report.Keys = append(report.Keys, key)
			if _, ok := report.Values[key]; !ok {
				report.Values[key] = receipt.Root().Link()
			}
		}
	}

	msg := mdm.AgentMessageModel{
		UcantoMessage7: &mdm.DataModel{
			Execute: ex,
			Report:  report,
		},
	}

	rt, err := block.Encode(
		&msg,
		mdm.Type(),
		cbor.Codec,
		sha256.Hasher,
	)
	if err != nil {
		return nil, err
	}
	err = bs.Put(rt)
	if err != nil {
		return nil, err
	}

	return &message{
		root:  rt,
		data:  msg.UcantoMessage7,
		blks:  bs,
		invs:  invCache,
		rcpts: rcptCache,
	}, nil
}

func NewMessage(root ipld.Link, blks blockstore.BlockReader) (AgentMessage, error) {
	rblock, ok, err := blks.Get(root)
	if err != nil {
		return nil, fmt.Errorf("getting root block: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("missing root block: %s", root)
	}

	msg := mdm.AgentMessageModel{}
	err = block.Decode(
		rblock,
		&msg,
		mdm.Type(),
		cbor.Codec,
		sha256.Hasher,
	)
	if err != nil {
		return nil, fmt.Errorf("decoding message: %w", err)
	}

	return &message{
		root:  rblock,
		data:  msg.UcantoMessage7,
		blks:  blks,
		invs:  map[string]invocation.Invocation{},
		rcpts: map[string]receipt.AnyReceipt{},
	}, nil
}

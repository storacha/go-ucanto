package message

import (
	"fmt"

	"github.com/alanshaw/go-ucanto/core/dag/blockstore"
	"github.com/alanshaw/go-ucanto/core/invocation"
	"github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/alanshaw/go-ucanto/core/ipld/block"
	"github.com/alanshaw/go-ucanto/core/ipld/codec/cbor"
	"github.com/alanshaw/go-ucanto/core/ipld/hash/sha256"
	"github.com/alanshaw/go-ucanto/core/iterable"
	mdm "github.com/alanshaw/go-ucanto/core/message/datamodel"
)

type AgentMessage interface {
	ipld.IPLDView
	// Invocations is a list of links to the root block of invocations than can
	// be found in the message.
	Invocations() []ipld.Link
	// Receipts is a list of links to the root block of receipts that can be
	// found in the message.
	Receipts() []ipld.Link
	// Get returns a receipt link from the message, given an invocation link.
	Get(link ipld.Link) (ipld.Link, bool)
}

type message struct {
	root ipld.Block
	data *mdm.DataModel
	blks blockstore.BlockReader
}

func (m *message) Root() ipld.Block {
	return m.root
}

func (m *message) Blocks() iterable.Iterator[ipld.Block] {
	return m.blks.Iterator()
}

func (m *message) Invocations() []ipld.Link {
	return m.data.Execute
}

func (m *message) Receipts() []ipld.Link {
	var rcpts []ipld.Link
	for _, k := range m.data.Report.Keys {
		l, _ := m.data.Report.Values[k]
		rcpts = append(rcpts, l)
	}
	return rcpts
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

	msg := mdm.AgentMessageModel{
		UcantoMessage7: &mdm.DataModel{
			Execute: ex,
		},
	}

	rt, err := block.Encode(
		msg,
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

	return &message{root: rt, data: msg.UcantoMessage7, blks: bs}, nil
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

	msg := mdm.AgentMessageModel{}
	err = block.Decode(
		rblock,
		&msg,
		mdm.Type(),
		cbor.Codec,
		sha256.Hasher,
	)
	if err != nil {
		return nil, fmt.Errorf("decoding message: %s", err)
	}

	return &message{root: rblock, data: msg.UcantoMessage7, blks: blks}, nil
}

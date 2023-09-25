package message

import (
	"bytes"
	"fmt"

	"github.com/alanshaw/go-ucanto/core"
	"github.com/alanshaw/go-ucanto/core/dag"
	"github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/alanshaw/go-ucanto/core/iterable"
	prime "github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/node/bindnode"
)

type AgentMessage interface {
	ipld.IPLDView
	Invocations() []core.Invocation
	Receipts() []core.Receipt
	Get(link ipld.Link) (core.Receipt, error)
}

type message struct {
	root ipld.Block
	data *agentMessageData
	bs   dag.BlockStore
}

func (m *message) Blocks() iterable.Iterator[ipld.Block] {
	return m.bs.Iterator()
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

	data, err := decodeAgentMessageModel(rblock.Bytes())
	if err != nil {
		fmt.Errorf("decoding message: %s", err)
	}

	return &message{root: rblock, data: data, bs: bstore}, nil
}

// Describes ucanto@7 message data format send between (client/server) agents.
type agentMessageData struct {
	// Set of (invocation) delegation links to be executed by the agent.
	execute []ipld.Link
	// Map of receipts keyed by the (invocation) delegation.
	report map[string]ipld.Link
}

func decodeAgentMessageModel(b []byte) (*agentMessageData, error) {
	ts, err := prime.LoadSchemaBytes([]byte(`
		type AgentMessageModel union {
			| AgentMessageData "ucanto/message@7.0.0"
		} representation keyed
		
		type AgentMessageData struct {
			execute optional [Link]
			report optional {String:Link}
		}
	`))
	if err != nil {
		return nil, err
	}
	npt := bindnode.Prototype((*agentMessageData)(nil), ts.TypeByName("AgentMessageModel"))
	nb := npt.Representation().NewBuilder()
	err = dagcbor.Decode(nb, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	data := bindnode.Unwrap(nb.Build()).(*agentMessageData)
	return data, nil
}

// TODO: use IPLD schema
// func decodeAgentMessageModel(rnd datamodel.Node) (*agentMessageData, error) {
// 	if rnd.Kind() != datamodel.Kind_Map {
// 		return nil, fmt.Errorf("unexpected datamodel kind: %s", rnd.Kind())
// 	}
// 	vnd, err := rnd.LookupByString("ucanto/message@7.0.0")
// 	if err != nil {
// 		return nil, fmt.Errorf("looking up variant key: %s", err)
// 	}
// 	if vnd.Kind() != datamodel.Kind_Map {
// 		return nil, fmt.Errorf("unexpected agent message data kind: %s", vnd.Kind())
// 	}
// 	exnd, err := vnd.LookupByString("execute")
// 	if err != nil {
// 		return nil, fmt.Errorf("looking up execute child: %s", err)
// 	}
// 	execute := []ipld.Link{}
// 	if !exnd.IsNull() {
// 		if exnd.Kind() != datamodel.Kind_List {
// 			return nil, fmt.Errorf("unexpected agent message execute kind: %s", exnd.Kind())
// 		}
// 		it := exnd.ListIterator()
// 		for it.Done() == false {
// 			_, item, err := it.Next()
// 			if err != nil {
// 				return nil, fmt.Errorf("iterating execute list: %s", err)
// 			}
// 			link, err := item.AsLink()
// 			if err != nil {
// 				return nil, fmt.Errorf("parsing execute list item as link: %s", err)
// 			}
// 			execute = append(execute, link)
// 		}
// 	}
// 	repnd, err := vnd.LookupByString("report")
// 	if err != nil {
// 		return nil, fmt.Errorf("looking up report child: %s", err)
// 	}
// 	report := map[string]ipld.Link{}
// 	if !repnd.IsNull() {
// 		if repnd.Kind() != datamodel.Kind_Map {
// 			return nil, fmt.Errorf("unexpected agent message execute kind: %s", exnd.Kind())
// 		}
// 		it := repnd.MapIterator()
// 		for it.Done() == false {
// 			k, v, err := it.Next()
// 			if err != nil {
// 				return nil, fmt.Errorf("iterating report map: %s", err)
// 			}
// 			key, err := k.AsString()
// 			if err != nil {
// 				return nil, fmt.Errorf("parsing report map key as string: %s", err)
// 			}
// 			link, err := v.AsLink()
// 			if err != nil {
// 				return nil, fmt.Errorf("parsing report map value as link: %s", err)
// 			}
// 			report[key] = link
// 		}
// 	}

// 	return &agentMessageData{execute, report}, nil
// }

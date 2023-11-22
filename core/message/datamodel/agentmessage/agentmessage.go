package agentmessage

import (
	"bytes"
	_ "embed"
	"fmt"
	"sync"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed agentmessage.ipldsch
var message []byte

var (
	once sync.Once
	ts   *schema.TypeSystem
	err  error
)

func loadSchema() (*schema.TypeSystem, error) {
	once.Do(func() {
		ts, err = ipld.LoadSchemaBytes(message)
		if err != nil {
			return
		}
	})
	return ts, err
}

type Model struct {
	UcantoMessage7 *Data
}

// Describes ucanto@7 message data format send between (client/server) agents.
type Data struct {
	// Set of (invocation) delegation links to be executed by the agent.
	Execute []ipld.Link
	// Map of receipts keyed by the (invocation) delegation.
	Report *ReportModel
}

type ReportModel struct {
	Keys   []string
	Values map[string]ipld.Link
}

func Encode(d *Data) ([]byte, error) {
	ts, err := loadSchema()
	if err != nil {
		return nil, err
	}
	schemaType := ts.TypeByName("AgentMessageModel")
	model := Model{d}
	node := bindnode.Wrap(&model, schemaType)
	var buf bytes.Buffer
	err = dagcbor.Encode(node.Representation(), &buf)
	if err != nil {
		return nil, fmt.Errorf("encoding dag-cbor: %s", err)
	}
	return buf.Bytes(), nil
}

func Decode(b []byte) (*Data, error) {
	ts, err := loadSchema()
	if err != nil {
		return nil, err
	}
	npt := bindnode.Prototype((*Model)(nil), ts.TypeByName("AgentMessageModel"))
	nb := npt.Representation().NewBuilder()
	err = dagcbor.Decode(nb, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	model := bindnode.Unwrap(nb.Build()).(*Model)
	return model.UcantoMessage7, nil
}

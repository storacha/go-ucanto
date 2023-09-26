package agentmessage

import (
	_ "embed"
	"sync"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed agentmessage.ipldsch
var message []byte

var (
	once sync.Once
	ts   *schema.TypeSystem
	err  error
)

func LoadSchema() (*schema.TypeSystem, error) {
	once.Do(func() {
		ts, err = ipld.LoadSchemaBytes(message)
		if err != nil {
			return
		}
	})
	return ts, err
}

// Describes ucanto@7 message data format send between (client/server) agents.
type Data struct {
	// Set of (invocation) delegation links to be executed by the agent.
	Execute []ipld.Link
	// Map of receipts keyed by the (invocation) delegation.
	Report map[string]ipld.Link
}

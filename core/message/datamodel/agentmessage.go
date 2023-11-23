package datamodel

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed agentmessage.ipldsch
var agentmessage []byte

var (
	once sync.Once
	ts   *schema.TypeSystem
	err  error
)

func mustLoadSchema() *schema.TypeSystem {
	once.Do(func() {
		ts, err = ipld.LoadSchemaBytes(agentmessage)
	})
	if err != nil {
		panic(fmt.Errorf("failed to load IPLD schema: %s", err))
	}
	return ts
}

func Type() schema.Type {
	return mustLoadSchema().TypeByName("AgentMessage")
}

type AgentMessageModel struct {
	UcantoMessage7 *DataModel
}

// Describes ucanto@7 message data format send between (client/server) agents.
type DataModel struct {
	// Set of (invocation) delegation links to be executed by the agent.
	Execute []ipld.Link
	// Map of receipts keyed by the (invocation) delegation.
	Report *ReportModel
}

type ReportModel struct {
	Keys   []string
	Values map[string]ipld.Link
}

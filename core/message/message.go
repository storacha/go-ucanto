package message

import (
	"fmt"

	"github.com/alanshaw/go-ucanto/core"
	coreipld "github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/ipld/go-ipld-prime"
)

type AgentMessage interface {
	coreipld.View
	Invocations() []core.Invocation
	Receipts() []core.Receipt
	Get(link ipld.Link) (core.Receipt, error)
}

type message struct {
}

func Build(invocation core.Invocation, receipt core.Receipt) (AgentMessage, error) {
	return nil, fmt.Errorf("not implemented")
}

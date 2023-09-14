package message

import (
	"fmt"

	"github.com/alanshaw/go-ucanto/core"
	"github.com/alanshaw/go-ucanto/core/ipld"
)

type AgentMessage interface {
	ipld.IPLDView
	Invocations() []core.Invocation
	Receipts() []core.Receipt
	Get(link ipld.Link) (core.Receipt, error)
}

type message struct {
}

func Build(invocation core.Invocation, receipt core.Receipt) (AgentMessage, error) {
	return nil, fmt.Errorf("not implemented")
}

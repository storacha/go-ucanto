package ran

import (
	"github.com/storacha-network/go-ucanto/core/dag/blockstore"
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/ucan"
)

type Ran struct {
	invocation invocation.Invocation
	link       ucan.Link
}

func (r Ran) Invocation() (invocation.Invocation, bool) {
	return r.invocation, r.invocation != nil
}

func (r Ran) Link() ucan.Link {
	if r.invocation != nil {
		return r.invocation.Link()
	}
	return r.link
}

func FromInvocation(invocation invocation.Invocation) Ran {
	return Ran{invocation, nil}
}

func FromLink(link ucan.Link) Ran {
	return Ran{nil, link}
}

func (r Ran) WriteInto(bs blockstore.BlockWriter) (ipld.Link, error) {
	if invocation, ok := r.Invocation(); ok {
		return r.Link(), blockstore.WriteInto(invocation, bs)
	}
	return r.Link(), nil
}

package server

import (
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/ucan"
)

type HandlerFunc[N any, C ucan.Capability[N], I invocation.Invocation, O, X ipld.Builder] func(capability C, invocation I, context InvocationContext) (Transaction[O, X], error)

// Provide is used to define given capability provider. It decorates the passed
// handler and takes care of UCAN validation. It only calls the handler
// when validation succeeds.
func Provide[N any, C ucan.Capability[N], I invocation.Invocation, O, X ipld.Builder](capability C, handler HandlerFunc[N, C, I, O, X]) ServiceMethod[I, O, X] {
	return func(invocation I, context InvocationContext) (Transaction[O, X], error) {
		// TODO: validation
		return handler(capability, invocation, context)
	}
}

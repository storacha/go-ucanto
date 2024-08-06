package server

import (
	"github.com/web3-storage/go-ucanto/core/invocation"
	"github.com/web3-storage/go-ucanto/core/ipld"
	"github.com/web3-storage/go-ucanto/ucan"
)

type HandlerFunc[N any, C ucan.Capability[N], I invocation.Invocation, O, X ipld.Datamodeler] func(capability C, invocation I, context InvocationContext) (Transaction[O, X], error)

// Provide is used to define given capability provider. It decorates the passed
// handler and takes care of UCAN validation. It only calls the handler
// when validation succeeds.
func Provide[N any, C ucan.Capability[N], I invocation.Invocation, O, X ipld.Datamodeler](capability C, handler HandlerFunc[N, C, I, O, X]) ServiceMethod[I, O, X] {
	return func(invocation I, context InvocationContext) (Transaction[O, X], error) {
		// TODO: validation
		return handler(capability, invocation, context)
	}
}

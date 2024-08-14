package server

import (
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/server/transaction"
	"github.com/storacha-network/go-ucanto/ucan"
)

type HandlerFunc[N any, C ucan.Capability[N], O, X ipld.Builder] func(capability C, invocation invocation.Invocation, context InvocationContext) (transaction.Transaction[O, X], error)

// Provide is used to define given capability provider. It decorates the passed
// handler and takes care of UCAN validation. It only calls the handler
// when validation succeeds.
func Provide[N any, C ucan.Capability[N], O, X ipld.Builder](capability C, handler HandlerFunc[N, C, O, X]) ServiceMethod[O, X] {
	return func(invocation invocation.Invocation, context InvocationContext) (transaction.Transaction[O, X], error) {
		// TODO: validation
		return handler(capability, invocation, context)
	}
}

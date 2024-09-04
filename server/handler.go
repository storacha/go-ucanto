package server

import (
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/server/transaction"
	"github.com/storacha-network/go-ucanto/ucan"
	"github.com/storacha-network/go-ucanto/validator"
)

type HandlerFunc[C any, O any] func(capability ucan.Capability[C], invocation invocation.Invocation, context InvocationContext) (out O, fork []ipld.Link, join ipld.Link, err error)

// Provide is used to define given capability provider. It decorates the passed
// handler and takes care of UCAN validation. It only calls the handler
// when validation succeeds.
func Provide[C any, O any](capability validator.CapabilityParser[C], handler HandlerFunc[C, O]) ServiceMethod[O, any] {
	return func(invocation invocation.Invocation, context InvocationContext) (transaction.Transaction[O, any], error) {
		vctx := validator.NewValidationContext(
			context.ID().Verifier(),
			capability,
			context.CanIssue,
			context.ValidateAuthorization,
			context.ResolveProof,
			context.ParsePrincipal,
			context.ResolveDIDKey,
		)

		authorization, err := validator.Access(invocation, vctx)
		if err != nil {
			return transaction.NewTransaction(result.Error[O](any(err))), err
		}

		o, fk, jn, herr := handler(authorization.Capability(), invocation, context)
		if herr != nil {

		}

		return transaction.NewTransaction(result.Ok[O, any](o))
	}
}

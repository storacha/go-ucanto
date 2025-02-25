package server

import (
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/receipt/fx"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/core/result/failure"
	"github.com/storacha/go-ucanto/core/schema"
	"github.com/storacha/go-ucanto/server/transaction"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/storacha/go-ucanto/validator"
)

type HandlerFunc[C any, O ipld.Builder] func(capability ucan.Capability[C], invocation invocation.Invocation, context InvocationContext) (out O, fx fx.Effects, err error)

// Provide is used to define given capability provider. It decorates the passed
// handler and takes care of UCAN validation. It only calls the handler
// when validation succeeds.
func Provide[C any, O ipld.Builder](capability validator.CapabilityParser[C], handler HandlerFunc[C, O]) ServiceMethod[O] {
	return func(invocation invocation.Invocation, context InvocationContext) (transaction.Transaction[O, ipld.Builder], error) {
		vctx := validator.NewValidationContext(
			context.ID().Verifier(),
			capability,
			context.CanIssue,
			context.ValidateAuthorization,
			context.ResolveProof,
			context.ParsePrincipal,
			context.ResolveDIDKey,
			context.AuthorityProofs()...,
		)

		// confirm the audience of the invocation is this service
		thisService := schema.Literal(context.ID().DID().String())
		if _, err := thisService.Read(invocation.Audience().DID().String()); err != nil {
			audErr := NewInvalidAudienceError(context.ID().DID().String(), invocation.Audience().DID().String())
			return transaction.NewTransaction(result.Error[O, ipld.Builder](audErr)), nil
		}

		auth, aerr := validator.Access(invocation, vctx)
		if aerr != nil {
			return transaction.NewTransaction(result.Error[O, ipld.Builder](failure.FromError(aerr))), nil
		}

		o, fx, herr := handler(auth.Capability(), invocation, context)
		if herr != nil {
			return nil, herr
		}

		return transaction.NewTransaction(result.Ok[O, ipld.Builder](o), transaction.WithEffects(fx)), nil
	}
}

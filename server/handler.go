package server

import (
	"context"

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

type HandlerFunc[C any, O ipld.Builder] func(ctx context.Context, capability ucan.Capability[C], invocation invocation.Invocation, context InvocationContext) (out O, fx fx.Effects, err error)

// Provide is used to define given capability provider. It decorates the passed
// handler and takes care of UCAN validation. It only calls the handler
// when validation succeeds.
func Provide[C any, O ipld.Builder](capability validator.CapabilityParser[C], handler HandlerFunc[C, O]) ServiceMethod[O] {
	return func(ctx context.Context, invocation invocation.Invocation, ictx InvocationContext) (transaction.Transaction[O, ipld.Builder], error) {
		vctx := validator.NewValidationContext(
			ictx.ID().Verifier(),
			capability,
			ictx.CanIssue,
			ictx.ValidateAuthorization,
			ictx.ResolveProof,
			ictx.ParsePrincipal,
			ictx.ResolveDIDKey,
			ictx.AuthorityProofs()...,
		)

		// confirm the audience of the invocation is this service or any of the configured alternative audiences
		acceptedAudiences := schema.Literal(ictx.ID().DID().String())
		if len(ictx.AlternativeAudiences()) > 0 {
			altAudiences := make([]schema.Reader[string, string], 0, len(ictx.AlternativeAudiences()))
			for _, a := range ictx.AlternativeAudiences() {
				altAudiences = append(altAudiences, schema.Literal(a.DID().String()))
			}

			acceptedAudiences = schema.Or(append(altAudiences, acceptedAudiences)...)
		}

		if _, err := acceptedAudiences.Read(invocation.Audience().DID().String()); err != nil {
			expectedAudiences := append([]ucan.Principal{ictx.ID()}, ictx.AlternativeAudiences()...)
			audErr := NewInvalidAudienceError(invocation.Audience(), expectedAudiences...)
			return transaction.NewTransaction(result.Error[O, ipld.Builder](audErr)), nil
		}

		auth, aerr := validator.Access(ctx, invocation, vctx)
		if aerr != nil {
			return transaction.NewTransaction(result.Error[O, ipld.Builder](failure.FromError(aerr))), nil
		}

		o, fx, herr := handler(ctx, auth.Capability(), invocation, ictx)
		if herr != nil {
			return nil, herr
		}

		return transaction.NewTransaction(result.Ok[O, ipld.Builder](o), transaction.WithEffects(fx)), nil
	}
}

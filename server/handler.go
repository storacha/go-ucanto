package server

import (
	"fmt"

	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/server/transaction"
	"github.com/storacha-network/go-ucanto/ucan"
	"github.com/storacha-network/go-ucanto/validator"
)

type HandlerFunc[C any, O, X ipld.Builder] func(capability ucan.Capability[C], invocation invocation.Invocation, context InvocationContext) (transaction.Transaction[O, X], error)

// Provide is used to define given capability provider. It decorates the passed
// handler and takes care of UCAN validation. It only calls the handler
// when validation succeeds.
func Provide[C any, O, X ipld.Builder](capability validator.CapabilityParser[C], handler HandlerFunc[C, O, X]) ServiceMethod[O, X] {
	return func(invocation invocation.Invocation, context InvocationContext) (transaction.Transaction[O, X], error) {
		vctx := validator.NewValidationContext(
			capability,
			context.CanIssue,
			context.ValidateAuthorization,
			context.ResolveProof,
			context.ParsePrincipal,
			context.ResolveDIDKey,
		)

		authorization, err := validator.Access(invocation, vctx)
		if err != nil {
			return nil, err
		}

		return result.MatchResultR2(authorization, func(ok validator.Authorization[C]) (transaction.Transaction[O, X], error) {
			return handler(ok.Capability(), invocation, context)
		}, func(err validator.UnauthorizedError[C]) (transaction.Transaction[O, X], error) {
			if failure, ok := any(err).(X); ok {
				return transaction.NewTransaction(result.Error[O](failure)), nil
			}
			return nil, fmt.Errorf("error was not an IPLD builder")
		})
	}
}

package retrieval2

import (
	"context"

	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/receipt/fx"
	"github.com/storacha/go-ucanto/server"
	"github.com/storacha/go-ucanto/server/transaction"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/storacha/go-ucanto/validator"
)

// HandlerFunc is an invocation handler function. It is different to
// [server.HandlerFunc] in that it allows an [RetrievalResponse] to be returned,
// which for a retrieval server will determine the HTTP headers and body content
// of the HTTP response. The usual handler response (out and effects) are added
// to the X-Agent-Message HTTP header.
type HandlerFunc[C any, O, X ipld.Builder] func(ctx context.Context, capability ucan.Capability[C], invocation invocation.Invocation, context server.InvocationContext) (Transaction[O, X], error)

// Provide is used to define given capability provider. It decorates the passed
// handler and takes care of UCAN validation. It only calls the handler
// when validation succeeds.
func Provide[C any, O ipld.Builder](capability validator.CapabilityParser[C], handler HandlerFunc[C, O]) ServiceMethod[O] {
	return func(ctx context.Context, inv invocation.Invocation, ictx server.InvocationContext) (transaction.Transaction[O, ipld.Builder], *RetrievalResponse, error) {
		var response *RetrievalResponse
		method := server.Provide(capability, func(ctx context.Context, capability ucan.Capability[C], inv invocation.Invocation, ictx server.InvocationContext) (O, fx.Effects, error) {
			out, fx, res, err := handler(ctx, capability, inv, ictx)
			response = res
			return out, fx, err
		})
		tx, err := method(ctx, inv, ictx)
		return tx, response, err
	}
}

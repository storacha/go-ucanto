package server

import (
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/server/transaction"
	"github.com/storacha-network/go-ucanto/transport"
)

// Option is an option configuring a ucanto server.
type Option func(cfg *srvConfig) error

type srvConfig struct {
	codec                 transport.InboundCodec
	service               map[string]ServiceMethod[ipld.Builder, ipld.Builder]
	validateAuthorization AuthorizationValidatorFunc
	canIssue              CanIssueFunc
	catch                 ErrorHandlerFunc
}

func WithServiceMethod[O, X ipld.Builder](can string, handleFunc ServiceMethod[O, X]) Option {
	return func(cfg *srvConfig) error {
		cfg.service[can] = func(input invocation.Invocation, context InvocationContext) (transaction.Transaction[ipld.Builder, ipld.Builder], error) {
			tx, err := handleFunc(input, context)
			if err != nil {
				return nil, err
			}
			out := result.MapResultR0(
				tx.Out(),
				func(o O) ipld.Builder { return o },
				func(x X) ipld.Builder { return x },
			)
			var opts []transaction.Option
			if tx.Fx() != nil {
				opts = append(opts, transaction.WithForks(tx.Fx().Fork()), transaction.WithJoin(tx.Fx().Join()))
			}
			return transaction.NewTransaction(out, opts...), nil
		}
		return nil
	}
}

// WithInboundCodec configures the codec used to decode requests and encode
// responses.
func WithInboundCodec(codec transport.InboundCodec) Option {
	return func(cfg *srvConfig) error {
		cfg.codec = codec
		return nil
	}
}

// WithAuthValidator configures the authorization validator function. The
// primary purpose of the validator is to allow checking UCANs for revocation.
func WithAuthValidator(fn AuthorizationValidatorFunc) Option {
	return func(cfg *srvConfig) error {
		cfg.validateAuthorization = fn
		return nil
	}
}

// WithErrorHandler configures a function to be called when errors occur during
// execution of a handler.
func WithErrorHandler(fn ErrorHandlerFunc) Option {
	return func(cfg *srvConfig) error {
		cfg.catch = fn
		return nil
	}
}

// WithCanIssue configures a function that determines whether a given capability
// can be issued by a given DID or whether it needs to be delegated to the
// issuer.
func WithCanIssue(fn CanIssueFunc) Option {
	return func(cfg *srvConfig) error {
		cfg.canIssue = fn
		return nil
	}
}

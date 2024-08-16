package server

import (
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/server/transaction"
	"github.com/storacha-network/go-ucanto/transport"
	"github.com/storacha-network/go-ucanto/validator"
)

// Option is an option configuring a ucanto server.
type Option func(cfg *srvConfig) error

type srvConfig struct {
	codec                 transport.InboundCodec
	service               map[string]ServiceMethod[ipld.Builder, ipld.Builder]
	validateAuthorization validator.RevocationCheckerFunc[any]
	canIssue              validator.CanIssueFunc[any]
	resolveProof          validator.ProofResolverFunc
	parsePrincipal        validator.PrincipalParserFunc
	resolveDIDKey         validator.PrincipalResolverFunc
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

// WithRevocationChecker configures the function used to check UCANs for
// revocation.
func WithRevocationChecker(fn validator.RevocationCheckerFunc[any]) Option {
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
func WithCanIssue(fn validator.CanIssueFunc[any]) Option {
	return func(cfg *srvConfig) error {
		cfg.canIssue = fn
		return nil
	}
}

// WithProofResolver configures a function that finds delegations corresponding
// to a given link. If a resolver is not provided the validator may not be able
// to explore corresponding path within a proof chain.
func WithProofResolver(fn validator.ProofResolverFunc) Option {
	return func(cfg *srvConfig) error {
		cfg.resolveProof = fn
		return nil
	}
}

// WithPrincipalParser configures a function that provides verifier instances
// that can validate UCANs issued by a given principal.
func WithPrincipalParser(fn validator.PrincipalParserFunc) Option {
	return func(cfg *srvConfig) error {
		cfg.parsePrincipal = fn
		return nil
	}
}

// WithPrincipalResolver configures a function that resolves the key of a
// principal that is identified by DID different from did:key method.
func WithPrincipalResolver(fn validator.PrincipalResolverFunc) Option {
	return func(cfg *srvConfig) error {
		cfg.resolveDIDKey = fn
		return nil
	}
}

package server

import (
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/server/transaction"
	"github.com/storacha/go-ucanto/transport"
	"github.com/storacha/go-ucanto/validator"
)

// Option is an option configuring a ucanto server.
type Option func(cfg *srvConfig) error

type srvConfig struct {
	codec                 transport.InboundCodec
	service               Service
	validateAuthorization validator.RevocationCheckerFunc[any]
	canIssue              validator.CanIssueFunc[any]
	resolveProof          validator.ProofResolverFunc
	parsePrincipal        validator.PrincipalParserFunc
	resolveDIDKey         validator.PrincipalResolverFunc
	catch                 ErrorHandlerFunc
}

func WithServiceMethod[O ipld.Builder](can string, handleFunc ServiceMethod[O]) Option {
	return func(cfg *srvConfig) error {
		cfg.service[can] = func(input invocation.Invocation, context InvocationContext) (transaction.Transaction[ipld.Builder, ipld.Builder], error) {
			tx, err := handleFunc(input, context)
			if err != nil {
				return nil, err
			}
			out := result.MapOk(tx.Out(), func(o O) ipld.Builder { return o })
			return transaction.NewTransaction(out, transaction.WithEffects(tx.Fx())), nil
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

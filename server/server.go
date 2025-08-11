package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/invocation/ran"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/core/receipt"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/core/result/failure"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-ucanto/principal/ed25519/verifier"
	"github.com/storacha/go-ucanto/server/transaction"
	"github.com/storacha/go-ucanto/transport"
	"github.com/storacha/go-ucanto/transport/car"
	thttp "github.com/storacha/go-ucanto/transport/http"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/storacha/go-ucanto/validator"
)

// InvocationContext is the context provided to service methods.
type InvocationContext interface {
	validator.RevocationChecker[any]
	validator.CanIssuer[any]
	validator.ProofResolver
	validator.PrincipalParser
	validator.PrincipalResolver
	validator.AuthorityProver
	// ID is the DID of the service the invocation was sent to.
	ID() principal.Signer

	// AlternativeAudiences are other audiences the service will accept for invocations.
	AlternativeAudiences() []ucan.Principal
}

// ServiceMethod is an invocation handler.
type ServiceMethod[O ipld.Builder, X failure.IPLDBuilderFailure] func(
	context.Context,
	invocation.Invocation,
	InvocationContext,
) (transaction.Transaction[O, X], error)

// Service is a mapping of service names to handlers, used to define a
// service implementation.
type Service = map[ucan.Ability]ServiceMethod[ipld.Builder, failure.IPLDBuilderFailure]

type ServiceInvocation = invocation.IssuedInvocation

type Server interface {
	// ID is the DID which will be used to verify that received invocation
	// audience matches it.
	ID() principal.Signer
	Codec() transport.InboundCodec
	Context() InvocationContext
	// Service is the actual service providing capability handlers.
	Service() Service
	Catch(err HandlerExecutionError[any])
}

// Server is a materialized service that is configured to use a specific
// transport channel. It has a invocation context which contains the DID of the
// service itself, among other things.
type ServerView interface {
	Server
	transport.Channel
	// Run executes a single invocation and returns a receipt.
	Run(ctx context.Context, invocation ServiceInvocation) (receipt.AnyReceipt, error)
}

// ErrorHandlerFunc allows non-result errors generated during handler execution
// to be logged.
type ErrorHandlerFunc func(err HandlerExecutionError[any])

func NewServer(id principal.Signer, options ...Option) (ServerView, error) {
	cfg := srvConfig{service: Service{}}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	codec := cfg.codec
	if codec == nil {
		codec = car.NewCARInboundCodec()
	}

	canIssue := cfg.canIssue
	if canIssue == nil {
		canIssue = validator.IsSelfIssued
	}

	catch := cfg.catch
	if catch == nil {
		catch = func(err HandlerExecutionError[any]) {
			fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		}
	}

	validateAuthorization := cfg.validateAuthorization
	if validateAuthorization == nil {
		validateAuthorization = func(context.Context, validator.Authorization[any]) validator.Revoked {
			return nil
		}
	}

	resolveProof := cfg.resolveProof
	if resolveProof == nil {
		resolveProof = validator.ProofUnavailable
	}

	parsePrincipal := cfg.parsePrincipal
	if parsePrincipal == nil {
		parsePrincipal = ParsePrincipal
	}

	resolveDIDKey := cfg.resolveDIDKey
	if resolveDIDKey == nil {
		resolveDIDKey = validator.FailDIDKeyResolution
	}

	ctx := serverContext{id, canIssue, validateAuthorization, resolveProof, parsePrincipal, resolveDIDKey, cfg.authorityProofs, cfg.altAudiences}
	svr := &server{id, cfg.service, ctx, codec, catch}
	return svr, nil
}

func ParsePrincipal(str string) (principal.Verifier, error) {
	// TODO: Ed or RSA
	return verifier.Parse(str)
}

type serverContext struct {
	id                    principal.Signer
	canIssue              validator.CanIssueFunc[any]
	validateAuthorization validator.RevocationCheckerFunc[any]
	resolveProof          validator.ProofResolverFunc
	parsePrincipal        validator.PrincipalParserFunc
	resolveDIDKey         validator.PrincipalResolverFunc
	authorityProofs       []delegation.Delegation
	altAudiences          []ucan.Principal
}

func (ctx serverContext) ID() principal.Signer {
	return ctx.id
}

func (sctx serverContext) CanIssue(capability ucan.Capability[any], issuer did.DID) bool {
	return sctx.canIssue(capability, issuer)
}

func (sctx serverContext) ValidateAuthorization(ctx context.Context, auth validator.Authorization[any]) validator.Revoked {
	return sctx.validateAuthorization(ctx, auth)
}

func (sctx serverContext) ResolveProof(ctx context.Context, proof ucan.Link) (delegation.Delegation, validator.UnavailableProof) {
	return sctx.resolveProof(ctx, proof)
}

func (sctx serverContext) ParsePrincipal(str string) (principal.Verifier, error) {
	return sctx.parsePrincipal(str)
}

func (sctx serverContext) ResolveDIDKey(ctx context.Context, did did.DID) (did.DID, validator.UnresolvedDID) {
	return sctx.resolveDIDKey(ctx, did)
}

func (sctx serverContext) AuthorityProofs() []delegation.Delegation {
	return sctx.authorityProofs
}

func (sctx serverContext) AlternativeAudiences() []ucan.Principal {
	return sctx.altAudiences
}

type server struct {
	id      principal.Signer
	service Service
	context InvocationContext
	codec   transport.InboundCodec
	catch   ErrorHandlerFunc
}

func (srv *server) ID() principal.Signer {
	return srv.id
}

func (srv *server) Service() Service {
	return srv.service
}

func (srv *server) Context() InvocationContext {
	return srv.context
}

func (srv *server) Codec() transport.InboundCodec {
	return srv.codec
}

func (srv *server) Request(ctx context.Context, request transport.HTTPRequest) (transport.HTTPResponse, error) {
	return Handle(ctx, srv, request)
}

func (srv *server) Run(ctx context.Context, invocation ServiceInvocation) (receipt.AnyReceipt, error) {
	return Run(ctx, srv, invocation)
}

func (srv *server) Catch(err HandlerExecutionError[any]) {
	srv.catch(err)
}

var _ transport.Channel = (*server)(nil)
var _ ServerView = (*server)(nil)

func Handle(ctx context.Context, server Server, request transport.HTTPRequest) (transport.HTTPResponse, error) {
	selection, aerr := server.Codec().Accept(request)
	if aerr != nil {
		return thttp.NewHTTPResponse(aerr.Status(), strings.NewReader(aerr.Error()), aerr.Headers()), nil
	}

	msg, err := selection.Decoder().Decode(request)
	if err != nil {
		return thttp.NewHTTPResponse(http.StatusBadRequest, strings.NewReader("The server failed to decode the request payload. Please format the payload according to the specified media type."), nil), nil
	}

	result, err := Execute(ctx, server, msg)
	if err != nil {
		return nil, err
	}

	return selection.Encoder().Encode(result)
}

func Execute(ctx context.Context, server Server, msg message.AgentMessage) (message.AgentMessage, error) {
	br, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(msg.Blocks()))
	if err != nil {
		return nil, err
	}

	var invs []invocation.Invocation
	for _, invlnk := range msg.Invocations() {
		inv, err := invocation.NewInvocationView(invlnk, br)
		if err != nil {
			return nil, err
		}
		invs = append(invs, inv)
	}

	var rcpts []receipt.AnyReceipt
	var rerr error
	var wg sync.WaitGroup
	var lock sync.RWMutex
	for _, inv := range invs {
		wg.Add(1)
		go func(inv invocation.Invocation) {
			defer wg.Done()
			rcpt, err := Run(ctx, server, inv)
			if err != nil {
				rerr = err
				return
			}

			lock.Lock()
			rcpts = append(rcpts, rcpt)
			lock.Unlock()
		}(inv)
	}
	wg.Wait()

	if rerr != nil {
		return nil, rerr
	}

	return message.Build(nil, rcpts)
}

func Run(ctx context.Context, server Server, invocation ServiceInvocation) (receipt.AnyReceipt, error) {
	caps := invocation.Capabilities()
	// Invocation needs to have one single capability
	if len(caps) != 1 {
		err := NewInvocationCapabilityError(invocation.Capabilities())
		return receipt.Issue(server.ID(), result.NewFailure(err), ran.FromInvocation(invocation))
	}

	cap := caps[0]
	handle, ok := server.Service()[cap.Can()]
	if !ok {
		err := NewHandlerNotFoundError(cap)
		return receipt.Issue(server.ID(), result.NewFailure(err), ran.FromInvocation(invocation))
	}

	tx, err := handle(ctx, invocation, server.Context())
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		herr := NewHandlerExecutionError(err, cap)
		server.Catch(herr)
		return receipt.Issue(server.ID(), result.NewFailure(herr), ran.FromInvocation(invocation))
	}

	fx := tx.Fx()
	var opts []receipt.Option
	if fx != nil {
		opts = append(opts, receipt.WithJoin(fx.Join()), receipt.WithFork(fx.Fork()...))
	}

	rcpt, err := receipt.Issue(server.ID(), tx.Out(), ran.FromInvocation(invocation), opts...)
	if err != nil {
		herr := NewHandlerExecutionError(err, cap)
		server.Catch(herr)
		return receipt.Issue(server.ID(), result.NewFailure(herr), ran.FromInvocation(invocation))
	}

	return rcpt, nil
}

package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/web3-storage/go-ucanto/core/dag/blockstore"
	"github.com/web3-storage/go-ucanto/core/invocation"
	"github.com/web3-storage/go-ucanto/core/invocation/ran"
	"github.com/web3-storage/go-ucanto/core/ipld"
	"github.com/web3-storage/go-ucanto/core/message"
	"github.com/web3-storage/go-ucanto/core/receipt"
	"github.com/web3-storage/go-ucanto/core/result"
	"github.com/web3-storage/go-ucanto/did"
	"github.com/web3-storage/go-ucanto/principal"
	"github.com/web3-storage/go-ucanto/principal/ed25519/verifier"
	"github.com/web3-storage/go-ucanto/transport"
	"github.com/web3-storage/go-ucanto/transport/car"
	thttp "github.com/web3-storage/go-ucanto/transport/http"
	"github.com/web3-storage/go-ucanto/ucan"
	"github.com/web3-storage/go-ucanto/validator"
)

// PrincipalParser provides verifier instances that can validate UCANs issued
// by a given principal.
type PrincipalParser interface {
	Parse(str string) (principal.Verifier, error)
}

// CanIssue informs validator whether given capability can be issued by a
// given DID or whether it needs to be delegated to the issuer.
type CanIssueFunc func(capability ucan.Capability[any], issuer did.DID) bool

// InvocationContext is the context provided to service methods.
type InvocationContext interface {
	// ID is the DID of the service the invocation was sent to.
	ID() principal.Signer
	Principal() PrincipalParser
	// CanIssue informs validator whether given capability can be issued by a
	// given DID or whether it needs to be delegated to the issuer.
	CanIssue(capability ucan.Capability[any], issuer did.DID) bool
	// ValidateAuthorization validates the passed authorization and returns
	// a result indicating validity. The primary purpose is to check for
	// revocation.
	ValidateAuthorization(auth validator.Authorization) result.Failure
}

// Transaction defines a result & effect pair, used by provider that wishes to
// return results that have effects.
type Transaction[O, X ipld.Datamodeler] interface {
	Out() result.Result[O, X]
	Fx() receipt.Effects
}

// ServiceMethod is an invocation handler.
type ServiceMethod[I invocation.Invocation, O, X ipld.Datamodeler] func(input I, context InvocationContext) (Transaction[O, X], error)

// ServiceDefinition is a mapping of service names to handlers, used to define a
// service implementation.
type ServiceDefinition = map[string]ServiceMethod[invocation.Invocation, ipld.Datamodeler, ipld.Datamodeler]

type ServiceInvocation = invocation.IssuedInvocation

type Server interface {
	// ID is the DID which will be used to verify that received invocation
	// audience matches it.
	ID() principal.Signer
	Codec() transport.InboundCodec
	Context() InvocationContext
	// Service is the actual service providing capability handlers.
	Service() ServiceDefinition
	Catch(err HandlerExecutionError[any])
}

// Server is a materialized service that is configured to use a specific
// transport channel. It has a invocation context which contains the DID of the
// service itself, among other things.
type ServerView interface {
	Server
	transport.Channel
	// Run executes a single invocation and returns a receipt.
	Run(invocation ServiceInvocation) (receipt.AnyReceipt, error)
}

// AuthorizationValidatorFunc validates the passed authorization and returns
// a result indicating validity. The primary purpose is to check for revocation.
type AuthorizationValidatorFunc func(auth validator.Authorization) result.Failure

// ErrorHandlerFunc allows non-result errors generated during handler execution
// to be logged.
type ErrorHandlerFunc func(err HandlerExecutionError[any])

// Option is an option configuring a ucanto server.
type Option func(cfg *srvConfig) error

type srvConfig struct {
	codec                 transport.InboundCodec
	validateAuthorization AuthorizationValidatorFunc
	canIssue              CanIssueFunc
	catch                 ErrorHandlerFunc
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

func NewServer(id principal.Signer, service ServiceDefinition, options ...Option) (ServerView, error) {
	cfg := srvConfig{}
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
			fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		}
	}

	validateAuthorization := cfg.validateAuthorization
	if validateAuthorization == nil {
		validateAuthorization = func(auth validator.Authorization) result.Failure {
			return nil
		}
	}

	ctx := &context{id: id, canIssue: canIssue, principal: &principalParser{}}
	svr := &server{id: id, service: service, context: ctx, codec: codec}
	return svr, nil
}

type principalParser struct{}

func (p *principalParser) Parse(str string) (principal.Verifier, error) {
	return verifier.Parse(str)
}

var _ PrincipalParser = (*principalParser)(nil)

type context struct {
	id                    principal.Signer
	canIssue              CanIssueFunc
	principal             PrincipalParser
	validateAuthorization AuthorizationValidatorFunc
}

func (ctx *context) ID() principal.Signer {
	return ctx.id
}

func (ctx *context) CanIssue(capability ucan.Capability[any], issuer did.DID) bool {
	return ctx.canIssue(capability, issuer)
}

func (ctx *context) Principal() PrincipalParser {
	return ctx.principal
}

func (ctx *context) ValidateAuthorization(auth validator.Authorization) result.Failure {
	return ctx.validateAuthorization(auth)
}

var _ InvocationContext = (*context)(nil)

type server struct {
	id      principal.Signer
	service ServiceDefinition
	context InvocationContext
	codec   transport.InboundCodec
	catch   ErrorHandlerFunc
}

func (srv *server) ID() principal.Signer {
	return srv.id
}

func (srv *server) Service() ServiceDefinition {
	return srv.service
}

func (srv *server) Context() InvocationContext {
	return srv.context
}

func (srv *server) Codec() transport.InboundCodec {
	return srv.codec
}

func (srv *server) Request(request transport.HTTPRequest) (transport.HTTPResponse, error) {
	return Handle(srv, request)
}

func (srv *server) Run(invocation ServiceInvocation) (receipt.AnyReceipt, error) {
	return Run(srv, invocation)
}

func (srv *server) Catch(err HandlerExecutionError[any]) {
	srv.catch(err)
}

var _ transport.Channel = (*server)(nil)
var _ ServerView = (*server)(nil)

func Handle(server Server, request transport.HTTPRequest) (transport.HTTPResponse, error) {
	selection, aerr := server.Codec().Accept(request)
	if aerr != nil {
		return thttp.NewHTTPResponse(aerr.Status(), strings.NewReader(aerr.Error()), aerr.Headers()), nil
	}

	msg, err := selection.Decoder().Decode(request)
	if err != nil {
		return thttp.NewHTTPResponse(http.StatusBadRequest, strings.NewReader("The server failed to decode the request payload. Please format the payload according to the specified media type."), nil), nil
	}

	result, err := Execute(server, msg)
	if err != nil {
		return nil, err
	}

	return selection.Encoder().Encode(result)
}

func Execute(server Server, msg message.AgentMessage) (message.AgentMessage, error) {
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
			rcpt, err := Run(server, inv)
			if err != nil {
				rerr = err
				return
			}

			lock.Lock()
			defer lock.Unlock()
			rcpts = append(rcpts, rcpt)
		}(inv)
	}
	wg.Wait()

	if rerr != nil {
		return nil, rerr
	}

	return message.Build(nil, rcpts)
}

func Run(server Server, invocation ServiceInvocation) (receipt.AnyReceipt, error) {
	caps := invocation.Capabilities()
	// Invocation needs to have one single capability
	if len(caps) != 1 {
		res := result.Error(NewInvocationCapabilityError(invocation.Capabilities()).ToIPLD())
		return receipt.Issue(server.ID(), res, ran.FromInvocation(invocation))
	}

	cap := caps[0]
	handle, ok := server.Service()[cap.Can()]
	if !ok {
		res := result.Error(NewHandlerNotFoundError(cap).ToIPLD())
		return receipt.Issue(server.ID(), res, ran.FromInvocation(invocation))
	}

	outcome, err := handle(invocation, server.Context())
	if err != nil {
		herr := NewHandlerExecutionError(err, cap)
		server.Catch(herr)

		res := result.Error(herr.ToIPLD())
		return receipt.Issue(server.ID(), res, ran.FromInvocation(invocation))
	}

	out := outcome.Out()
	var res result.AnyResult
	if value, ok := out.Ok(); ok {
		res = result.Ok(value.ToIPLD())
	}
	if value, ok := out.Error(); ok {
		res = result.Error(value.ToIPLD())
	}

	fx := outcome.Fx()
	var opts []receipt.Option
	if fx != nil {
		opts = append(opts, receipt.WithJoin(fx.Join()), receipt.WithForks(fx.Fork()))
	}

	rcpt, err := receipt.Issue(server.ID(), res, ran.FromInvocation(invocation), opts...)
	if err != nil {
		herr := NewHandlerExecutionError(err, cap)
		server.Catch(herr)

		res := result.Error(herr.ToIPLD())
		return receipt.Issue(server.ID(), res, ran.FromInvocation(invocation))
	}

	return rcpt, nil
}

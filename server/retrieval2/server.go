package retrieval2

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/core/receipt"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-ucanto/server"
	"github.com/storacha/go-ucanto/server/transaction"
	"github.com/storacha/go-ucanto/transport"
	"github.com/storacha/go-ucanto/transport/headercar"
)

type RetrievalRequest struct {
	// Relative URL requested.
	URL url.URL
	// Headers are the HTTP headers sent in the HTTP request to the server.
	Headers http.Header
}

type RetrievalResponse struct {
	// Status is the HTTP status that should be returned. e.g. 206 when returning
	// a range request.
	Status int
	// Headers are additional HTTP headers to return in the response. At minimum
	// they should include the Content-Length header, but should also include
	// Content-Range for byte range responses.
	Headers http.Header
	// Body is the data to return in the response body.
	Body io.Reader
}

type InvocationContext interface {
	server.InvocationContext
	Request() *RetrievalRequest
}

type Transaction[O any, X any] interface {
	transaction.Transaction[O, X]
	Response() *RetrievalResponse
}

// ServiceMethod is an invocation handler. It is different to
// [server.ServiceMethod] in that it allows an [RetrievalResponse] to be
// returned, which for a retrieval server will determine the HTTP headers and
// body content of the HTTP response. The usual handler response (out and
// effects) are added to the X-Agent-Message HTTP header.
type ServiceMethod[O, X ipld.Builder] func(context.Context, invocation.Invocation, InvocationContext) (Transaction[O, X], error)

type ServiceInvocation = invocation.IssuedInvocation

type Server[O, X ipld.Builder] interface {
	// ID is the DID which will be used to verify that received invocation
	// audience matches it.
	ID() principal.Signer
	Codec() transport.InboundCodec
	Context() InvocationContext
	// Handler is the capability handler for retrievals.
	Handler() ServiceMethod[O, X]
	Catch(err server.HandlerExecutionError[any])
}

// Server is a materialized service that is configured to use a specific
// transport channel. It has a invocation context which contains the DID of the
// service itself, among other things.
type ServerView[O, X ipld.Builder] interface {
	Server[O, X]
	transport.Channel
}

// NewServer creates a retrieval server, which is a UCAN server that comes
// pre-loaded with a [headercar] codec.
//
// Handlers have an additional return parameter - the data to return in the body
// of the response.
//
// They require a delegation cache, which allows delegations that are too big
// for the header to be sent in multiple rounds. By default an in-memory cache
// is provided if none is passed in options.
//
// The carheader codec will accept agent messages where the invocation is a CID
// that can be looked up in the delegations cache.
//
// The delegations cache should be a size bounded LRU to prevent DoS attacks.
func NewServer[O, X ipld.Builder](id principal.Signer, handler ServiceMethod[O, X], options ...Option) (ServerView[O, X], error) {
	cfg := srvConfig{service: Service{}}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	dlgCache := cfg.delegationCache
	if dlgCache == nil {
		dc, err := NewMemoryDelegationCache(-1)
		if err != nil {
			return nil, err
		}
		dlgCache = dc
	}

	bodyProvider := &bodyProvider{responses: map[string]*RetrievalResponse{}}
	codec := headercar.NewInboundCodec(headercar.WithResponseBodyProvider(bodyProvider))

	srvOpts := []server.Option{server.WithInboundCodec(codec)}
	for ability, method := range cfg.service {
		srvOpts = append(srvOpts, server.WithServiceMethod(ability, func(ctx context.Context, inv invocation.Invocation, ictx server.InvocationContext) (transaction.Transaction[ipld.Builder, ipld.Builder], error) {
			txn, err := method(ctx, inv, ictx)
			bodyProvider.setResponse(inv.Link(), txn.Response())
			return txn, err
		}))
	}
	if cfg.canIssue != nil {
		srvOpts = append(srvOpts, server.WithCanIssue(cfg.canIssue))
	}
	if cfg.catch != nil {
		srvOpts = append(srvOpts, server.WithErrorHandler(cfg.catch))
	}
	if cfg.validateAuthorization != nil {
		srvOpts = append(srvOpts, server.WithRevocationChecker(cfg.validateAuthorization))
	}
	if cfg.resolveProof != nil {
		srvOpts = append(srvOpts, server.WithProofResolver(cfg.resolveProof))
	}
	if cfg.parsePrincipal != nil {
		srvOpts = append(srvOpts, server.WithPrincipalParser(cfg.parsePrincipal))
	}
	if cfg.resolveDIDKey != nil {
		srvOpts = append(srvOpts, server.WithPrincipalResolver(cfg.resolveDIDKey))
	}

	return server.NewServer(id, srvOpts...)
}

type retrievalServer struct {
	server server.ServerView
}

func (rs *retrievalServer) ID() principal.Signer {
	return rs.server.ID()
}

func (rs *retrievalServer) Service() server.Service {
	return rs.server.Service()
}

func (rs *retrievalServer) Context() server.InvocationContext {
	return rs.server.Context()
}

func (rs *retrievalServer) Codec() transport.InboundCodec {
	return rs.server.Codec()
}

func (rs *retrievalServer) Request(ctx context.Context, request transport.HTTPRequest) (transport.HTTPResponse, error) {
	return rs.server.Request(ctx, request)
}

func (rs *retrievalServer) Run(ctx context.Context, invocation server.ServiceInvocation) (receipt.AnyReceipt, error) {
	return rs.server.Run(ctx, rs, invocation)
}

func (rs *retrievalServer) Catch(err HandlerExecutionError[any]) {
	srv.catch(err)
}

var _ transport.Channel = (*server)(nil)
var _ ServerView = (*server)(nil)

type bodyProvider struct {
	responses map[string]*RetrievalResponse
	mutex     sync.Mutex
}

func (bp *bodyProvider) setResponse(ran ipld.Link, retrieval *RetrievalResponse) {
	bp.mutex.Lock()
	bp.responses[ran.String()] = retrieval
	bp.mutex.Unlock()
}

func (bp *bodyProvider) Stream(msg message.AgentMessage) (io.Reader, int, http.Header, error) {
	rcpt, _, err := msg.Receipt(msg.Receipts()[0])
	if err != nil {
		return nil, 0, nil, err
	}
	inv := rcpt.Ran()
	key := inv.Link().String()
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	res, ok := bp.responses[key]
	if !ok {
		return nil, 0, nil, nil
	}
	delete(bp.responses, key)
	return res.Body, res.Status, res.Headers, nil
}

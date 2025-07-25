package retrieval

import (
	"context"
	"io"
	"net/http"
	"sync"

	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-ucanto/server"
	"github.com/storacha/go-ucanto/server/transaction"
	"github.com/storacha/go-ucanto/transport/headercar"
	"github.com/storacha/go-ucanto/ucan"
)

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

// ServiceMethod is an invocation handler. It is different to
// [server.ServiceMethod] in that it allows an [RetrievalResponse] to be
// returned, which for a retrieval server will determine the HTTP headers and
// body content of the HTTP response. The usual handler response (out and
// effects) are added to the X-Agent-Message HTTP header.
type ServiceMethod[O ipld.Builder] func(context.Context, invocation.Invocation, server.InvocationContext) (transaction.Transaction[O, ipld.Builder], *RetrievalResponse, error)

// Service is a mapping of service names to handlers, used to define a
// retrieval server service implementation.
type Service = map[ucan.Ability]ServiceMethod[ipld.Builder]

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
func NewServer(id principal.Signer, options ...Option) (server.ServerView, error) {
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
	codec := headercar.NewInboundCodec(dlgCache, headercar.WithResponseBodyProvider(bodyProvider))

	srvOpts := []server.Option{server.WithInboundCodec(codec)}
	for ability, method := range cfg.service {
		srvOpts = append(srvOpts, server.WithServiceMethod(ability, func(ctx context.Context, inv invocation.Invocation, ictx server.InvocationContext) (transaction.Transaction[ipld.Builder, ipld.Builder], error) {
			txn, res, err := method(ctx, inv, ictx)
			bodyProvider.setResponse(inv.Link(), res)
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

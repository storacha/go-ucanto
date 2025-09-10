package retrieval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	ipldprime "github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagjson"
	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/core/receipt"
	"github.com/storacha/go-ucanto/core/receipt/ran"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/core/result/failure"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-ucanto/server"
	"github.com/storacha/go-ucanto/server/transaction"
	"github.com/storacha/go-ucanto/transport"
	"github.com/storacha/go-ucanto/transport/headercar"
	hcmsg "github.com/storacha/go-ucanto/transport/headercar/message"
	thttp "github.com/storacha/go-ucanto/transport/http"
	"github.com/storacha/go-ucanto/ucan"
)

type Request struct {
	// Relative URL requested.
	URL *url.URL
	// Headers are the HTTP headers sent in the HTTP request to the server.
	Headers http.Header
}

type Response struct {
	// Status is the HTTP status that should be returned. e.g. 206 when returning
	// a range request.
	Status int
	// Headers are additional HTTP headers to return in the response. At minimum
	// they should include the Content-Length header, but should also include
	// Content-Range for byte range responses.
	Headers http.Header
	// Body is the data to return in the response body.
	Body io.ReadCloser
}

func NewResponse(status int, headers http.Header, body io.ReadCloser) Response {
	return Response{Status: status, Headers: headers, Body: body}
}

// ServiceMethod is an invocation handler. It is different to
// [server.ServiceMethod] in that it allows an [Response] to be
// returned as part of the [transation.Transation], which for a retrieval server
// will determine the HTTP headers and body content of the HTTP response. The
// usual handler response (out and effects) are added to the X-Agent-Message
// HTTP header.
type ServiceMethod[O ipld.Builder, X failure.IPLDBuilderFailure] func(
	context.Context,
	invocation.Invocation,
	server.InvocationContext,
	Request,
) (transaction.Transaction[O, X], Response, error)

// Service is a mapping of service names to handlers, used to define a
// service implementation.
type Service = map[ucan.Ability]ServiceMethod[ipld.Builder, failure.IPLDBuilderFailure]

// CachingServer is a retrieval server that also caches invocations/delegations
// to allow invocations with delegations chains bigger than HTTP header size
// limits to be executed as multiple requests.
type CachingServer interface {
	server.Server[Service]
	Cache() delegation.Store
}

// NewServer creates a retrieval server, which is a UCAN server that comes
// pre-loaded with a [headercar] codec.
//
// Handlers have an additional return value - the data to return in the body
// of the response as well as HTTP headers and status code. They also have an
// additional parameter, which are the details of the request - the URL that was
// requested and the HTTP headers.
//
// They require a delegation cache, which allows delegations that are too big
// for the header to be sent in multiple rounds. By default an in-memory cache
// is provided if none is passed in options.
//
// The carheader codec will accept agent messages where the invocation is a CID
// that can be looked up in the delegations cache.
//
// The delegations cache should be a size bounded LRU to prevent DoS attacks.
func NewServer(id principal.Signer, options ...Option) (*Server, error) {
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

	codec := headercar.NewInboundCodec()
	srvOpts := []server.Option{server.WithInboundCodec(codec)}
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

	srv, err := server.NewServer(id, srvOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating server: %w", err)
	}

	return &Server{
		server:          srv,
		service:         cfg.service,
		delegationCache: dlgCache,
	}, nil
}

type Server struct {
	server          server.Server[server.Service]
	service         Service
	delegationCache delegation.Store
}

func (srv *Server) ID() principal.Signer {
	return srv.server.ID()
}

func (srv *Server) Service() Service {
	return srv.service
}

func (srv *Server) Context() server.InvocationContext {
	return srv.server.Context()
}

func (srv *Server) Codec() transport.InboundCodec {
	return srv.server.Codec()
}

// Request handles an inbound HTTP request to the retrieval server. The request
// URL will only be non-empty if this method is called with a request that is a
// [transport.InboundHTTPRequest].
func (srv *Server) Request(ctx context.Context, request transport.HTTPRequest) (transport.HTTPResponse, error) {
	return Handle(ctx, srv, request)
}

func (srv *Server) Run(ctx context.Context, invocation server.ServiceInvocation) (receipt.AnyReceipt, error) {
	rcpt, _, err := Run(ctx, srv, invocation, Request{})
	return rcpt, err
}

func (srv *Server) Cache() delegation.Store {
	return srv.delegationCache
}

func (srv *Server) Catch(err server.HandlerExecutionError[any]) {
	srv.server.Catch(err)
}

var _ CachingServer = (*Server)(nil)

func Handle(ctx context.Context, srv CachingServer, request transport.HTTPRequest) (transport.HTTPResponse, error) {
	resp, err := handle(ctx, srv, request)
	if err != nil {
		return nil, err
	}

	// ensure headers are not nil
	headers := resp.Headers()
	if headers == nil {
		headers = http.Header{}
	} else {
		headers = headers.Clone()
	}
	// ensure the Vary header is set for ALL responses
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Vary
	headers.Add("Vary", hcmsg.HeaderName)

	if resp.Body() == nil {
		return thttp.NewResponse(resp.Status(), http.NoBody, headers), nil
	}
	return thttp.NewResponse(resp.Status(), resp.Body(), headers), nil
}

func handle(ctx context.Context, srv CachingServer, request transport.HTTPRequest) (transport.HTTPResponse, error) {
	selection, aerr := srv.Codec().Accept(request)
	if aerr != nil {
		return thttp.NewResponse(aerr.Status(), io.NopCloser(strings.NewReader(aerr.Error())), aerr.Headers()), nil
	}

	msg, err := selection.Decoder().Decode(request)
	if err != nil {
		return thttp.NewResponse(http.StatusBadRequest, io.NopCloser(strings.NewReader("The server failed to decode the request payload. Please format the payload according to the specified media type.")), nil), nil
	}

	// retrieval server supports only 1 invocation in the agent message, since
	// only a single handler can use the body.
	invs := msg.Invocations()
	if len(invs) != 1 {
		var rcpts []receipt.AnyReceipt
		res := result.NewFailure(NewAgentMessageInvocationError())
		for _, l := range invs {
			rcpt, err := receipt.Issue(srv.ID(), res, ran.FromLink(l))
			if err != nil {
				return nil, fmt.Errorf("issuing invocation error receipt: %w", err)
			}
			rcpts = append(rcpts, rcpt)
		}
		out, err := message.Build(nil, rcpts)
		if err != nil {
			return nil, fmt.Errorf("building invocation error message: %w", err)
		}
		resp, err := selection.Encoder().Encode(out)
		if err != nil {
			return nil, fmt.Errorf("encoding invocation error message: %w", err)
		}
		return resp, nil
	}

	retreq := Request{Headers: request.Headers()}
	if inreq, ok := request.(transport.InboundHTTPRequest); ok {
		retreq.URL = inreq.URL()
	}

	out, execResp, err := Execute(ctx, srv, msg, retreq)
	if err != nil {
		return nil, fmt.Errorf("executing invocations: %w", err)
	}

	// if there is no agent message to respond with, we simply respond with the
	// response from execution (i.e. missing proofs response)
	if out == nil {
		return thttp.NewResponse(execResp.Status, execResp.Body, execResp.Headers), nil
	}

	encResp, err := selection.Encoder().Encode(out)
	if err != nil {
		return nil, fmt.Errorf("encoding response message: %w", err)
	}

	// Use status from execution response if non-zero and encode response status
	// is zero or 200.
	status := encResp.Status()
	if execResp.Status != 0 && (status == 0 || status == http.StatusOK) {
		status = execResp.Status
	}

	// Merge headers
	headers := encResp.Headers()
	if execResp.Headers != nil {
		if headers == nil {
			headers = http.Header{}
		}
		for name, values := range execResp.Headers {
			for _, v := range values {
				headers.Add(name, v)
			}
		}
	}

	return thttp.NewResponse(status, execResp.Body, headers), nil
}

func Execute(ctx context.Context, srv CachingServer, msg message.AgentMessage, req Request) (message.AgentMessage, Response, error) {
	// retrieval server supports only 1 invocation in the agent message, since
	// only a single handler can use the body.
	invs := msg.Invocations()
	if len(invs) != 1 {
		var rcpts []receipt.AnyReceipt
		res := result.NewFailure(NewAgentMessageInvocationError())
		for _, l := range invs {
			rcpt, err := receipt.Issue(srv.ID(), res, ran.FromLink(l))
			if err != nil {
				return nil, Response{}, err
			}
			rcpts = append(rcpts, rcpt)
		}
		out, err := message.Build(nil, rcpts)
		if err != nil {
			return nil, Response{}, err
		}
		return out, Response{Status: http.StatusBadRequest}, nil
	}

	inv, err := ExtractInvocation(ctx, invs[0], msg, srv.Cache())
	if err != nil {
		mpe := MissingProofs{}
		if errors.As(err, &mpe) {
			n, err := mpe.ToIPLD()
			if err != nil {
				return nil, Response{}, fmt.Errorf("building missing proofs IPLD view: %w", err)
			}
			body, err := ipldprime.Encode(n, dagjson.Encode)
			if err != nil {
				return nil, Response{}, fmt.Errorf("encoding missing proofs repsonse: %w", err)
			}
			headers := http.Header{}
			expiry := time.Now().Add(10 * time.Minute).Unix() // TODO: honour this?
			headers.Set("X-UCAN-Cache-Expiry", fmt.Sprintf("%d", expiry))
			headers.Set("Content-Type", "application/json")
			return nil, Response{
				Status:  http.StatusNotExtended,
				Body:    io.NopCloser(bytes.NewReader(body)),
				Headers: headers,
			}, nil
		}
		return nil, Response{}, err
	}

	rcpt, resp, err := Run(ctx, srv, inv, req)
	if err != nil {
		return nil, Response{}, fmt.Errorf("running invocation: %w", err)
	}
	out, err := message.Build(nil, []receipt.AnyReceipt{rcpt})
	if err != nil {
		return nil, Response{}, fmt.Errorf("building agent message: %w", err)
	}
	return out, resp, nil
}

func ExtractInvocation(ctx context.Context, root ipld.Link, msg message.AgentMessage, cache delegation.Store) (invocation.Invocation, error) {
	bs, err := blockstore.NewBlockStore(blockstore.WithBlocksIterator(msg.Blocks()))
	if err != nil {
		return nil, fmt.Errorf("creating blockstore from agent message: %w", err)
	}

	var dlgs []delegation.Delegation
	var newdlgs []delegation.Delegation
	chkpfs := []ipld.Link{root}
	var missingpfs []ipld.Link
	for len(chkpfs) > 0 {
		prf := chkpfs[0]
		chkpfs = chkpfs[1:]

		blk, ok, err := bs.Get(prf)
		if err != nil {
			return nil, fmt.Errorf("getting block %s: %w", prf.String(), err)
		}

		var dlg delegation.Delegation
		if ok {
			dlg, err = delegation.NewDelegation(blk, bs)
			if err != nil {
				return nil, fmt.Errorf("creating delegation %s: %w", prf.String(), err)
			}
			newdlgs = append(newdlgs, dlg)
		} else {
			dlg, ok, err = cache.Get(ctx, prf)
			if err != nil {
				return nil, fmt.Errorf("getting delegation %s from cache: %w", prf.String(), err)
			}
			if !ok {
				missingpfs = append(missingpfs, prf)
				continue
			}
		}

		dlgs = append(dlgs, dlg)
		chkpfs = append(chkpfs, dlg.Proofs()...)
	}

	if len(missingpfs) > 0 {
		for _, dlg := range newdlgs {
			err := cache.Put(ctx, dlg) // cache new delegations for subsequent request
			if err != nil {
				return nil, fmt.Errorf("caching delegation %s: %w", dlg.Link().String(), err)
			}
		}
		return nil, NewMissingProofsError(missingpfs)
	}

	// Add the blocks from the delegations to the blockstore and create a new
	// invocation which has everything.
	for _, dlg := range dlgs {
		for b, err := range dlg.Export() {
			if err != nil {
				return nil, fmt.Errorf("exporting blocks from delegation %s: %w", dlg.Link().String(), err)
			}
			err = bs.Put(b)
			if err != nil {
				return nil, fmt.Errorf("putting block %s: %w", b.Link().String(), err)
			}
		}
	}
	return invocation.NewInvocationView(root, bs)
}

// Run is similar to [server.Run] except the receipts that are issued do not
// include the invocation block(s) in order to save bytes when transmitting the
// receipt in HTTP headers.
func Run(ctx context.Context, srv server.Server[Service], invocation server.ServiceInvocation, req Request) (receipt.AnyReceipt, Response, error) {
	caps := invocation.Capabilities()
	// Invocation needs to have one single capability
	if len(caps) != 1 {
		capErr := server.NewInvocationCapabilityError(invocation.Capabilities())
		rcpt, err := receipt.Issue(srv.ID(), result.NewFailure(capErr), ran.FromLink(invocation.Link()))
		return rcpt, Response{}, err
	}

	cap := caps[0]
	handle, ok := srv.Service()[cap.Can()]
	if !ok {
		notFoundErr := server.NewHandlerNotFoundError(cap)
		rcpt, err := receipt.Issue(srv.ID(), result.NewFailure(notFoundErr), ran.FromLink(invocation.Link()))
		return rcpt, Response{}, err
	}

	tx, resp, err := handle(ctx, invocation, srv.Context(), req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, Response{}, err
		}
		execErr := server.NewHandlerExecutionError(err, cap)
		srv.Catch(execErr)
		rcpt, err := receipt.Issue(srv.ID(), result.NewFailure(execErr), ran.FromLink(invocation.Link()))
		if err != nil {
			return nil, Response{}, err
		}
		return rcpt, resp, nil
	}

	fx := tx.Fx()
	var opts []receipt.Option
	if fx != nil {
		opts = append(opts, receipt.WithJoin(fx.Join()), receipt.WithFork(fx.Fork()...))
	}

	rcpt, err := receipt.Issue(srv.ID(), tx.Out(), ran.FromLink(invocation.Link()), opts...)
	if err != nil {
		return nil, Response{}, err
	}

	return rcpt, resp, nil
}

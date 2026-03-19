package retrieval

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"

	"github.com/storacha/go-ucanto/client"
	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/ipld/block"
	"github.com/storacha/go-ucanto/core/ipld/codec/cbor"
	"github.com/storacha/go-ucanto/core/ipld/codec/json"
	ucansha256 "github.com/storacha/go-ucanto/core/ipld/hash/sha256"
	"github.com/storacha/go-ucanto/core/message"
	mdm "github.com/storacha/go-ucanto/core/message/datamodel"
	rdm "github.com/storacha/go-ucanto/server/retrieval/datamodel"
	"github.com/storacha/go-ucanto/transport"
	"github.com/storacha/go-ucanto/transport/headercar"
	hcmsg "github.com/storacha/go-ucanto/transport/headercar/message"
	thttp "github.com/storacha/go-ucanto/transport/http"
	"github.com/storacha/go-ucanto/ucan"
)

// Option is an option configuring a retrieval connection.
type Option func(cfg *config)

type config struct {
	client  *http.Client
	headers http.Header
}

// WithClient configures the HTTP client the connection should use to make
// requests.
func WithClient(c *http.Client) Option {
	return func(cfg *config) {
		cfg.client = c
	}
}

// WithHeaders configures additional HTTP headers to send with requests.
func WithHeaders(h http.Header) Option {
	return func(cfg *config) {
		cfg.headers = h
	}
}

// NewConnection creates a new connection to a retrieval server that uses the
// headercar transport.
func NewConnection(id ucan.Principal, url *url.URL, opts ...Option) (*Connection, error) {
	cfg := config{}
	for _, o := range opts {
		o(&cfg)
	}

	hasher := sha256.New
	channel := thttp.NewChannel(
		url,
		thttp.WithMethod("GET"),
		thttp.WithSuccessStatusCode(
			http.StatusOK,
			http.StatusPartialContent,
			http.StatusNotExtended,                 // indicates further proof must be supplied
			http.StatusRequestHeaderFieldsTooLarge, // indicates invocation is too large to fit in headers
		),
		thttp.WithClient(cfg.client),
		thttp.WithHeaders(cfg.headers),
	)
	codec := headercar.NewOutboundCodec()
	return &Connection{id, channel, codec, hasher}, nil
}

type Connection struct {
	id      ucan.Principal
	channel transport.Channel
	codec   transport.OutboundCodec
	hasher  func() hash.Hash
}

var _ client.Connection = (*Connection)(nil)

func (c *Connection) ID() ucan.Principal {
	return c.id
}

func (c *Connection) Codec() transport.OutboundCodec {
	return c.codec
}

func (c *Connection) Channel() transport.Channel {
	return c.channel
}

func (c *Connection) Hasher() hash.Hash {
	return c.hasher()
}

// ExecutionOption is an option configuring a retrieval execution.
type ExecutionOption func(cfg *execConfig)

type execConfig struct {
	allowPublicRetrieval bool
}

// WithPublicRetrieval configures the client to allow retrievals from public
// buckets.
//
// This means that responses that do not include an X-Agent-Message header will
// be treated as valid rather than errors. It is up to the caller to inspect the
// response data to determine if it is acceptable.
//
// When this option is set and the response does not contain an X-Agent-Message
// header, the [client.ExecutionResponse] returned by the call to [Execute] will
// be nil.
//
// Note: this does not prevent the client from sending authorized requests, it
// only affects how responses are interpreted.
func WithPublicRetrieval() ExecutionOption {
	return func(cfg *execConfig) {
		cfg.allowPublicRetrieval = true
	}
}

// Execute performs a UCAN invocation using the headercar transport,
// implementing a "probe and retry" pattern to handle HTTP header size
// limitations when the invocation is too large to fit.
//
// The method first attempts to send the complete invocation (including all
// proofs) in HTTP headers. If this fails due to size constraints (typically 4KB
// header limit), it falls back to a multipart negotiation protocol:
//
//  1. Send invocation with ALL proofs omitted
//  2. Server responds with 510 (Not Extended) listing missing proof CID(s)
//  3. Send partial invocations with each missing proof attached one by one
//  4. Repeat until server has all required proofs (200/206 response)
//
// This approach optimizes for the common case (shallow delegation chains that
// fit in headers) while also handling deep proof chains that require
// multiple round trips. The server caches proofs between requests, so each
// proof only needs to be sent once per session.
//
// Note: The current implementation processes missing proofs sequentially rather
// than in batches, which means deep delegation chains will result in multiple
// HTTP round trips. This trade-off prioritizes implementation simplicity over
// network efficiency, which is acceptable given current delegation chain depths
// but may need optimization as authorization hierarchies grow deeper.
//
// Returns the execution response, the final HTTP response, and any error
// encountered.
func Execute(ctx context.Context, inv invocation.Invocation, conn client.Connection, options ...ExecutionOption) (client.ExecutionResponse, transport.HTTPResponse, error) {
	cfg := execConfig{}
	for _, o := range options {
		o(&cfg)
	}

	input, err := message.Build([]invocation.Invocation{inv}, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building message: %w", err)
	}

	var response transport.HTTPResponse
	multi := false

	req, err := conn.Codec().Encode(input)
	if err != nil {
		if errors.Is(err, hcmsg.ErrHeaderTooLarge) {
			multi = true
		} else {
			return nil, nil, fmt.Errorf("encoding message: %w", err)
		}
	} else {
		response, err = conn.Channel().Request(ctx, req)
		if err != nil {
			return nil, nil, fmt.Errorf("sending message: %w", err)
		}

		if response.Status() == http.StatusRequestHeaderFieldsTooLarge {
			multi = true
			err := response.Body().Close() // we don't need this anymore
			if err != nil {
				return nil, nil, fmt.Errorf("closing response body: %w", err)
			}
		}
	}

	// if the header fields are too big, we need to split the delegation into
	// multiple requests...
	if multi {
		response, err = sendPartialInvocations(ctx, inv, conn)
		if err != nil {
			return nil, nil, fmt.Errorf("sending partial invocations: %w", err)
		}
	}

	output, err := conn.Codec().Decode(response)
	if err != nil {
		if cfg.allowPublicRetrieval && errors.Is(err, hcmsg.ErrMissingHeader) {
			return nil, response, nil
		}
		return nil, nil, fmt.Errorf("decoding message: %w", err)
	}

	return client.ExecutionResponse(output), response, nil
}

func sendPartialInvocations(ctx context.Context, inv invocation.Invocation, conn client.Connection) (transport.HTTPResponse, error) {
	br, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(inv.Export()))
	if err != nil {
		return nil, fmt.Errorf("reading invocation blocks: %w", err)
	}
	part, err := omitProofs(inv)
	if err != nil {
		return nil, fmt.Errorf("creating invocation %s with omitted proofs: %w", inv.Link().String(), err)
	}

	parts := map[string]delegation.Delegation{}
	prfs := inv.Proofs()
	for len(prfs) > 0 {
		root := prfs[0]
		prfs = prfs[1:]
		prf, err := delegation.NewDelegationView(root, br)
		if err != nil {
			return nil, fmt.Errorf("creating delegation: %w", err)
		}
		prfs = append(prfs, prf.Proofs()...)
		// now export without proofs
		prf, err = omitProofs(prf)
		if err != nil {
			return nil, fmt.Errorf("creating delegation %s with omitted proofs: %w", prf.Link().String(), err)
		}
		parts[prf.Link().String()] = prf
	}
	// we already tried this
	if len(parts) == 0 {
		return nil, errors.New("invocation is too big to send in HTTP headers")
	}

	// now send the parts
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		input, err := newPartialInvocationMessage(inv.Link(), part)
		if err != nil {
			return nil, fmt.Errorf("building message: %w", err)
		}

		req, err := conn.Codec().Encode(input)
		if err != nil {
			return nil, fmt.Errorf("encoding message: %w", err)
		}

		res, err := conn.Channel().Request(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("sending message: %w", err)
		}

		if res.Status() == http.StatusPartialContent || res.Status() == http.StatusOK {
			return res, nil
		}

		// if still too big, then fail
		if res.Status() == http.StatusRequestHeaderFieldsTooLarge {
			return nil, errors.New("invocation is too big to send in HTTP headers")
		}

		if res.Status() != http.StatusNotExtended {
			return nil, fmt.Errorf("unexpected status code: %d", res.Status())
		}

		bodyReader := res.Body()
		body, err := io.ReadAll(bodyReader)
		if err != nil {
			bodyReader.Close()
			return nil, fmt.Errorf("reading not extended body: %w", err)
		}
		if err = bodyReader.Close(); err != nil {
			return nil, fmt.Errorf("closing response body: %w", err)
		}

		var model rdm.MissingProofsModel
		err = json.Decode(body, &model, rdm.MissingProofsType())
		if err != nil {
			return nil, fmt.Errorf("decoding body: %w", err)
		}
		if len(model.Proofs) == 0 {
			return nil, errors.New("server did not include missing proofs in response")
		}

		p, ok := parts[model.Proofs[0].String()]
		if !ok {
			return nil, fmt.Errorf("missing proof not found or was already sent: %s", model.Proofs[0].String())
		}
		part = p
		delete(parts, p.Link().String())
	}
}

func omitProofs(dlg delegation.Delegation) (delegation.Delegation, error) {
	blocks := dlg.Export(delegation.WithOmitProof(dlg.Proofs()...))
	bs, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(blocks))
	if err != nil {
		return nil, err
	}
	return delegation.NewDelegation(dlg.Root(), bs)
}

func newPartialInvocationMessage(invocation ipld.Link, part delegation.Delegation) (message.AgentMessage, error) {
	bs, err := blockstore.NewBlockStore(blockstore.WithBlocksIterator(part.Export()))
	if err != nil {
		return nil, err
	}
	msg := mdm.AgentMessageModel{
		UcantoMessage7: &mdm.DataModel{
			Execute: []ipld.Link{invocation},
		},
	}
	rt, err := block.Encode(
		&msg,
		mdm.Type(),
		cbor.Codec,
		ucansha256.Hasher,
	)
	if err != nil {
		return nil, err
	}
	err = bs.Put(rt)
	if err != nil {
		return nil, err
	}
	return message.NewMessage(rt.Link(), bs)
}

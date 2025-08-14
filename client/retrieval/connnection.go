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
	thttp "github.com/storacha/go-ucanto/transport/http"
	"github.com/storacha/go-ucanto/ucan"
)

// NewConnection creates a new connection to a retrieval server that uses the
// headercar transport.
func NewConnection(id ucan.Principal, endpoint string) (*Connection, error) {
	hasher := sha256.New
	url, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	channel := thttp.NewChannel(
		url,
		thttp.WithMethod("GET"),
		thttp.WithSuccessStatusCode(
			http.StatusOK,
			http.StatusPartialContent,
			http.StatusNotExtended,                 // indicates further proof must be supplied
			http.StatusRequestHeaderFieldsTooLarge, // indicates invocation is too large to fit in headers
		),
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

func Execute(ctx context.Context, inv invocation.Invocation, conn client.Connection) (client.ExecutionResponse, transport.HTTPResponse, error) {
	input, err := message.Build([]invocation.Invocation{inv}, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building message: %w", err)
	}

	req, err := conn.Codec().Encode(input)
	if err != nil {
		return nil, nil, fmt.Errorf("encoding message: %w", err)
	}

	response, err := conn.Channel().Request(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("sending message: %w", err)
	}

	// if the header fields are too big, we need to split the delegation into
	// multiple requests...
	if response.Status() == http.StatusRequestHeaderFieldsTooLarge {
		br, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(inv.Export()))
		if err != nil {
			return nil, nil, fmt.Errorf("reading invocation blocks: %w", err)
		}
		part, err := omitProofs(inv)
		if err != nil {
			return nil, nil, fmt.Errorf("creating invocation %s with omitted proofs: %w", inv.Link().String(), err)
		}

		parts := map[string]delegation.Delegation{}
		prfs := inv.Proofs()
		for len(prfs) > 0 {
			root := prfs[0]
			prfs = prfs[1:]
			prf, err := delegation.NewDelegationView(root, br)
			if err != nil {
				return nil, nil, fmt.Errorf("creating delegation: %w", err)
			}
			prfs = append(prfs, prf.Proofs()...)
			// now export without proofs
			prf, err = omitProofs(prf)
			if err != nil {
				return nil, nil, fmt.Errorf("creating delegation %s with omitted proofs: %w", prf.Link().String(), err)
			}
			parts[prf.Link().String()] = prf
		}
		// we already tried this
		if len(parts) == 0 {
			return nil, nil, errors.New("invocation is too big to send in HTTP headers")
		}

		// now send the parts
		for {
			input, err := newPartialInvocationMessage(inv.Link(), part)
			if err != nil {
				return nil, nil, fmt.Errorf("building message: %w", err)
			}

			req, err := conn.Codec().Encode(input)
			if err != nil {
				return nil, nil, fmt.Errorf("encoding message: %w", err)
			}

			res, err := conn.Channel().Request(ctx, req)
			if err != nil {
				return nil, nil, fmt.Errorf("sending message: %w", err)
			}

			if res.Status() == http.StatusPartialContent || res.Status() == http.StatusOK {
				response = res
				break
			}

			// if still too big, then fail
			if res.Status() == http.StatusRequestHeaderFieldsTooLarge {
				return nil, nil, errors.New("invocation is too big to send in HTTP headers")
			}

			if res.Status() != http.StatusNotExtended {
				return nil, nil, fmt.Errorf("unexpected status code: %d", res.Status())
			}

			body, err := io.ReadAll(response.Body())
			if err != nil {
				return nil, nil, fmt.Errorf("reading not extended body: %w", err)
			}

			var model rdm.MissingProofsModel
			err = json.Decode(body, &model, rdm.MissingProofsType())
			if err != nil {
				return nil, nil, fmt.Errorf("decoding body: %w", err)
			}
			if len(model.Proofs) == 0 {
				return nil, nil, fmt.Errorf("missing missing proofs: %w", err)
			}

			p, ok := parts[model.Proofs[0].String()]
			if !ok {
				return nil, nil, fmt.Errorf("missing proof not found or was already sent: %s", model.Proofs[0].String())
			}
			part = p
			delete(parts, p.Link().String())
		}
	}

	output, err := conn.Codec().Decode(response)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding message: %w", err)
	}

	return client.ExecutionResponse(output), response, nil
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

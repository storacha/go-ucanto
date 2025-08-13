package retrieval

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"net/http"
	"net/url"

	"github.com/storacha/go-ucanto/client"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/message"
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
			http.StatusNoContent,
			http.StatusNotExtended,
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

	var response transport.HTTPResponse
	for {
		req, err := conn.Codec().Encode(input)
		if err != nil {
			return nil, nil, fmt.Errorf("encoding message: %w", err)
		}

		// TODO: split request if too large

		res, err := conn.Channel().Request(ctx, req)
		if err != nil {
			return nil, nil, fmt.Errorf("sending message: %w", err)
		}

		if res.Status() == http.StatusNotExtended {
			return nil, nil, errors.New("not implemented")
		}

		response = res
		break
	}

	output, err := conn.Codec().Decode(response)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding message: %w", err)
	}

	return client.ExecutionResponse(output), response, nil
}

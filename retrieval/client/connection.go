package client

import (
	"context"
	"crypto/sha256"
	"fmt"
	"hash"

	"github.com/storacha/go-ucanto/client"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/transport"
	"github.com/storacha/go-ucanto/transport/headercar"
	"github.com/storacha/go-ucanto/ucan"
)

func NewConnection(id ucan.Principal, channel transport.Channel) (*Connection, error) {
	hasher := sha256.New
	codec := headercar.NewOutboundCodec()
	return &Connection{id, codec, channel, hasher}, nil
}

type Connection struct {
	id      ucan.Principal
	codec   transport.OutboundCodec
	channel transport.Channel
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

func Execute(ctx context.Context, invocations []invocation.Invocation, conn client.Connection) (client.ExecutionResponse, error) {
	input, err := message.Build(invocations, nil)
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

	output, err := conn.Codec().Decode(res)
	if err != nil {
		return nil, fmt.Errorf("decoding message: %w", err)
	}

	return client.ExecutionResponse(output), nil
}

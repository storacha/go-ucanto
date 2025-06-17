package client

import (
	"context"
	"crypto/sha256"
	"fmt"
	"hash"
	"iter"

	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld/block"
	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/transport"
	"github.com/storacha/go-ucanto/transport/car"
	"github.com/storacha/go-ucanto/ucan"
)

type Connection interface {
	ID() ucan.Principal
	Channel() transport.Channel
	Codec() transport.OutboundCodec
	Hasher() hash.Hash
}

// Option is an option configuring a ucanto connection.
type Option func(cfg *connConfig) error

type connConfig struct {
	hasher func() hash.Hash
	codec  transport.OutboundCodec
}

// WithHasher configures the hasher factory.
func WithHasher(hasher func() hash.Hash) Option {
	return func(cfg *connConfig) error {
		cfg.hasher = hasher
		return nil
	}
}

// WithOutboundCodec configures the codec used to encode requests and decode
// responses.
func WithOutboundCodec(codec transport.OutboundCodec) Option {
	return func(cfg *connConfig) error {
		cfg.codec = codec
		return nil
	}
}

func NewConnection(id ucan.Principal, channel transport.Channel, options ...Option) (Connection, error) {
	cfg := connConfig{hasher: sha256.New}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	hasher := cfg.hasher
	if hasher == nil {
		hasher = sha256.New
	}

	codec := cfg.codec
	if codec == nil {
		codec = car.NewCAROutboundCodec()
	}

	c := conn{id, codec, channel, hasher}
	return &c, nil
}

type conn struct {
	id      ucan.Principal
	codec   transport.OutboundCodec
	channel transport.Channel
	hasher  func() hash.Hash
}

var _ Connection = (*conn)(nil)

func (c *conn) ID() ucan.Principal {
	return c.id
}

func (c *conn) Codec() transport.OutboundCodec {
	return c.codec
}

func (c *conn) Channel() transport.Channel {
	return c.channel
}

func (c *conn) Hasher() hash.Hash {
	return c.hasher()
}

type ExecutionResponse interface {
	// Blocks returns an iterator of all the IPLD blocks that are included in
	// the response.
	Blocks() iter.Seq2[block.Block, error]
	// Get returns a link to a receipt, given an invocation link.
	Get(inv ucan.Link) (ucan.Link, bool)
}

func Execute(ctx context.Context, invocations []invocation.Invocation, conn Connection) (ExecutionResponse, error) {
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

	return ExecutionResponse(output), nil
}

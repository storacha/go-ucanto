package client

import (
	"crypto/sha256"
	"fmt"
	"hash"

	"github.com/alanshaw/go-ucanto/core/invocation"
	"github.com/alanshaw/go-ucanto/core/message"
	"github.com/alanshaw/go-ucanto/core/receipt"
	"github.com/alanshaw/go-ucanto/transport"
	"github.com/alanshaw/go-ucanto/ucan"
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
}

// WithHasher configures the hasher factory.
func WithHasher(hasher func() hash.Hash) Option {
	return func(cfg *connConfig) error {
		cfg.hasher = hasher
		return nil
	}
}

func NewConnection(id ucan.Principal, codec transport.OutboundCodec, channel transport.Channel, options ...Option) (Connection, error) {
	cfg := connConfig{sha256.New}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}
	c := conn{id, codec, channel, cfg.hasher}
	return &c, nil
}

type conn struct {
	id      ucan.Principal
	codec   transport.OutboundCodec
	channel transport.Channel
	hasher  func() hash.Hash
}

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

// TODO: move single execute onto invocation for now...
func Execute[O, X any](invocation invocation.Invocation, rcptreader receipt.ReceiptReader[O, X], conn Connection) (receipt.Receipt[O, X], error) {
	input, err := message.Build(invocation)
	if err != nil {
		return nil, fmt.Errorf("building message: %s", err)
	}

	req, err := conn.Codec().Encode(input)
	if err != nil {
		return nil, fmt.Errorf("encoding message: %s", err)
	}

	res, err := conn.Channel().Request(req)
	if err != nil {
		return nil, fmt.Errorf("sending message: %s", err)
	}

	output, err := conn.Codec().Decode(res)
	if err != nil {
		return nil, fmt.Errorf("decoding message: %s", err)
	}

	rt, ok := output.Get(invocation.Link())
	if !ok {
		return nil, fmt.Errorf("missing receipt for invocation: %s", invocation.Link())
	}

	return rcptreader.Get(output, rt)
}

var _ Connection = (*conn)(nil)

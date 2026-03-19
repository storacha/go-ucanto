package message

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"

	"github.com/multiformats/go-multibase"
	"github.com/storacha/go-ucanto/core/car"
	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/message"
)

const (
	// HeaderName is the default name of the HTTP header.
	HeaderName = "X-Agent-Message"
	// Maximum size in bytes the header value may be.
	MaxHeaderSizeBytes = 4 * 1024
)

var (
	ErrHeaderTooLarge = errors.New("maximum agent message header size exceeded")
	ErrMissingHeader  = fmt.Errorf("missing %s header", HeaderName)
)

type encodeConfig struct {
	maxSize int
}

type EncodeOption func(c *encodeConfig)

// WithMaxSize configures the maximum size allowed for the header value. The
// default is [MaxHeaderSizeBytes]. Set to -1 to disable the size restriction.
func WithMaxSize(size int) EncodeOption {
	return func(c *encodeConfig) {
		c.maxSize = size
	}
}

// EncodeHeader encodes a [message.AgentMessage] as a HTTP header string.
func EncodeHeader(msg message.AgentMessage, opts ...EncodeOption) (string, error) {
	cfg := encodeConfig{}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.maxSize == 0 {
		cfg.maxSize = MaxHeaderSizeBytes
	}

	data := car.Encode([]ipld.Link{msg.Root().Link()}, msg.Blocks())

	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err := io.Copy(gz, data)
	if err != nil {
		gz.Close()
		return "", fmt.Errorf("compressing CAR data: %w", err)
	}
	if err := gz.Close(); err != nil {
		return "", fmt.Errorf("closing gzip writer: %w", err)
	}

	h, err := multibase.Encode(multibase.Base64, b.Bytes())
	if err != nil {
		return "", fmt.Errorf("multibase encoding: %w", err)
	}

	if cfg.maxSize != -1 && len(h) > cfg.maxSize {
		return "", ErrHeaderTooLarge
	}

	return h, nil
}

// DecodeHeader decodes a [message.AgentMessage] from a HTTP header string.
func DecodeHeader(h string) (message.AgentMessage, error) {
	_, data, err := multibase.Decode(h)
	if err != nil {
		return nil, fmt.Errorf("multibase decoding X-Agent-Message header: %w", err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gz.Close()
	roots, blocks, err := car.Decode(gz)
	if err != nil {
		return nil, fmt.Errorf("decoding CAR: %w", err)
	}
	if len(roots) != 1 {
		return nil, fmt.Errorf("unexpected number of roots: %d", len(roots))
	}
	bstore, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(blocks))
	if err != nil {
		return nil, fmt.Errorf("creating blockstore: %w", err)
	}
	return message.NewMessage(roots[0], bstore)
}

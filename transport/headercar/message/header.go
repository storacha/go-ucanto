package message

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/multiformats/go-multibase"
	"github.com/storacha/go-ucanto/core/car"
	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/message"
)

// AgentMessageHeader is the default name of the HTTP header.
var AgentMessageHeader = "X-Agent-Message"

// EncodeHeader encodes a [message.AgentMessage] as a HTTP header string.
func EncodeHeader(msg message.AgentMessage) (string, error) {
	data := car.Encode([]ipld.Link{msg.Root().Link()}, msg.Blocks())

	r, w := io.Pipe()
	go func() {
		gz := gzip.NewWriter(w)
		_, err := io.Copy(gz, data)
		gz.Close()
		w.CloseWithError(err)
	}()

	var b bytes.Buffer
	_, err := b.ReadFrom(r)
	if err != nil {
		return "", fmt.Errorf("reading encoded CAR: %w", err)
	}

	h, err := multibase.Encode(multibase.Base64, b.Bytes())
	if err != nil {
		return "", fmt.Errorf("multibase encoding: %w", err)
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

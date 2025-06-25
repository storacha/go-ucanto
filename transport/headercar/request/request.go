package request

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
	"github.com/storacha/go-ucanto/transport"
	chmessage "github.com/storacha/go-ucanto/transport/headercar/message"
	uhttp "github.com/storacha/go-ucanto/transport/http"
)

func Encode(msg message.AgentMessage, data chmessage.AgentMessageDataStreamer) (transport.HTTPRequest, error) {
	carBody := car.Encode([]ipld.Link{msg.Root().Link()}, msg.Blocks())

	r, w := io.Pipe()
	go func() {
		gz := gzip.NewWriter(w)
		_, err := io.Copy(gz, carBody)
		gz.Close()
		w.CloseWithError(err)
	}()

	var b bytes.Buffer
	_, err := b.ReadFrom(r)
	if err != nil {
		return nil, fmt.Errorf("reading encoded CAR: %w", err)
	}

	xAgentMsg, err := multibase.Encode(multibase.Base64, b.Bytes())
	if err != nil {
		return nil, fmt.Errorf("multibase encoding: %w", err)
	}

	body, headers, err := data.Stream(msg)
	if err != nil {
		return nil, fmt.Errorf("streaming data: %w", err)
	}
	headers.Set("X-Agent-Message", xAgentMsg)

	return uhttp.NewHTTPRequest(body, headers), nil
}

func Decode(req transport.HTTPRequest) (message.AgentMessage, error) {
	msgHdr := req.Headers().Get("X-Agent-Message")
	if msgHdr == "" {
		return nil, errors.New("missing X-Agent-Message header in request")
	}
	_, data, err := multibase.Decode(msgHdr)
	if msgHdr == "" {
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
	bstore, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(blocks))
	if err != nil {
		return nil, fmt.Errorf("creating blockstore: %w", err)
	}
	return message.NewMessage(roots, bstore)
}

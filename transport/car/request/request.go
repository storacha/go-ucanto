package request

import (
	"fmt"
	"net/http"

	"github.com/storacha/go-ucanto/core/car"
	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/transport"
	uhttp "github.com/storacha/go-ucanto/transport/http"
)

const ContentType = car.ContentType

func Encode(message message.AgentMessage) (transport.HTTPRequest, error) {
	headers := http.Header{}
	headers.Add("Content-Type", car.ContentType)
	// signal that we want to receive a CAR file in the response
	headers.Add("Accept", car.ContentType)
	reader := car.Encode([]ipld.Link{message.Root().Link()}, message.Blocks())
	return uhttp.NewHTTPRequest(reader, headers), nil
}

func Decode(req transport.HTTPRequest) (message.AgentMessage, error) {
	roots, blocks, err := car.Decode(req.Body())
	if err != nil {
		return nil, fmt.Errorf("decoding CAR: %w", err)
	}
	bstore, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(blocks))
	if err != nil {
		return nil, fmt.Errorf("creating blockstore: %w", err)
	}
	return message.NewMessage(roots, bstore)
}

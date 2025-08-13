package response

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

func Encode(msg message.AgentMessage) (transport.HTTPResponse, error) {
	headers := http.Header{}
	headers.Add("Content-Type", car.ContentType)
	reader := car.Encode([]ipld.Link{msg.Root().Link()}, msg.Blocks())
	return uhttp.NewResponse(http.StatusOK, reader, headers), nil
}

func Decode(response transport.HTTPResponse) (message.AgentMessage, error) {
	roots, blocks, err := car.Decode(response.Body())
	if err != nil {
		return nil, fmt.Errorf("decoding CAR: %w", err)
	}
	bstore, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(blocks))
	if err != nil {
		return nil, fmt.Errorf("creating blockstore: %w", err)
	}
	return message.NewMessage(roots, bstore)
}

package response

import (
	"fmt"

	"github.com/alanshaw/go-ucanto/core/car"
	"github.com/alanshaw/go-ucanto/core/dag"
	"github.com/alanshaw/go-ucanto/core/message"
	"github.com/alanshaw/go-ucanto/transport"
)

const ContentType = car.ContentType

func Decode(response transport.HTTPResponse) (message.AgentMessage, error) {
	roots, blocks, err := car.Decode(response.Body())
	if err != nil {
		return nil, fmt.Errorf("decoding response: %s", err)
	}
	bstore, err := dag.NewBlockStore(blocks)
	if err != nil {
		return nil, fmt.Errorf("creating blockstore: %s", err)
	}
	return message.NewMessage(roots, bstore)
}

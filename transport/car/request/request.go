package request

import (
	"net/http"

	"github.com/alanshaw/go-ucanto/core/car"
	"github.com/alanshaw/go-ucanto/core/message"
	"github.com/alanshaw/go-ucanto/transport"
	"github.com/ipld/go-ipld-prime"
)

const ContentType = car.ContentType

func Encode(message message.AgentMessage) (transport.HTTPRequest, error) {
	headers := http.Header{}
	headers.Add("Content-Type", car.ContentType)
	// signal that we want to receive a CAR file in the response
	headers.Add("Accept", car.ContentType)
	reader := car.Encode([]ipld.Link{message.Root().Link()}, message.Blocks())
	return transport.NewHTTPRequest(reader, headers), nil
}

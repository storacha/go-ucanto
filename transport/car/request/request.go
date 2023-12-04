package request

import (
	"net/http"

	"github.com/ipld/go-ipld-prime"
	"github.com/web3-storage/go-ucanto/core/car"
	"github.com/web3-storage/go-ucanto/core/message"
	"github.com/web3-storage/go-ucanto/transport"
	uhttp "github.com/web3-storage/go-ucanto/transport/http"
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

package transport

import (
	"github.com/web3-storage/go-ucanto/core/message"
)

type RequestEncoder interface {
	Encode(message message.AgentMessage) (HTTPRequest, error)
}

type RequestDecoder interface {
	Decode(request HTTPRequest) (message.AgentMessage, error)
}

type ResponseEncoder interface {
	Encode(message message.AgentMessage) (HTTPResponse, error)
}

type ResponseDecoder interface {
	Decode(response HTTPResponse) (message.AgentMessage, error)
}

type OutboundCodec interface {
	RequestEncoder
	ResponseDecoder
}

type InboundAcceptCodec interface {
	// Decoder will be used by a server to decode HTTP Request into an invocation
	// `Batch` that will be executed using a `service`.
	Decoder() RequestDecoder
	// Encoder will be used to encode batch of invocation results into an HTTP
	// response that will be sent back to the client that initiated the request.
	Encoder() ResponseEncoder
}

type InboundCodec interface {
	Accept(request HTTPRequest) (InboundAcceptCodec, HTTPError)
}

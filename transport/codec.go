package transport

import (
	"github.com/alanshaw/go-ucanto/core/message"
)

type RequestEncoder interface {
	Encode(message message.AgentMessage) (HTTPRequest, error)
}

type ResponseDecoder interface {
	Decode(response HTTPResponse) (message.AgentMessage, error)
}

type OutboundCodec interface {
	RequestEncoder
	ResponseDecoder
}

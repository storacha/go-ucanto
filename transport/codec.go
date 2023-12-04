package transport

import (
	"github.com/web3-storage/go-ucanto/core/message"
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

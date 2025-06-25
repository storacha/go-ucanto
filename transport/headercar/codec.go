package headercar

import (
	"net/http"

	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/transport"
	chmessage "github.com/storacha/go-ucanto/transport/headercar/message"
	"github.com/storacha/go-ucanto/transport/headercar/request"
	"github.com/storacha/go-ucanto/transport/headercar/response"
	thttp "github.com/storacha/go-ucanto/transport/http"
)

type config struct {
	data chmessage.AgentMessageDataStreamer
}

type Option func(c *config)

func WithDataStreamer(dataStreamer chmessage.AgentMessageDataStreamer) Option {
	return func(c *config) {
		c.data = dataStreamer
	}
}

type OutboundCodec struct {
	data chmessage.AgentMessageDataStreamer
}

func (oc *OutboundCodec) Encode(msg message.AgentMessage) (transport.HTTPRequest, error) {
	return request.Encode(msg, oc.data)
}

func (oc *OutboundCodec) Decode(res transport.HTTPResponse) (message.AgentMessage, error) {
	return response.Decode(res)
}

var _ transport.OutboundCodec = (*OutboundCodec)(nil)

func NewOutboundCodec(opts ...Option) transport.OutboundCodec {
	cfg := config{}
	for _, option := range opts {
		option(&cfg)
	}
	if cfg.data == nil {
		cfg.data = chmessage.EmptyDataStreamer{}
	}
	return &OutboundCodec{data: cfg.data}
}

type InboundAcceptCodec struct {
	data chmessage.AgentMessageDataStreamer
}

func (cic *InboundAcceptCodec) Encoder() transport.ResponseEncoder {
	return cic
}

func (cic *InboundAcceptCodec) Decoder() transport.RequestDecoder {
	return cic
}

func (cic *InboundAcceptCodec) Encode(msg message.AgentMessage) (transport.HTTPResponse, error) {
	return response.Encode(msg, cic.data)
}

func (cic *InboundAcceptCodec) Decode(req transport.HTTPRequest) (message.AgentMessage, error) {
	return request.Decode(req)
}

type InboundCodec struct {
	codec transport.InboundAcceptCodec
}

func (ic *InboundCodec) Accept(req transport.HTTPRequest) (transport.InboundAcceptCodec, transport.HTTPError) {
	msgHdr := req.Headers().Get("X-Agent-Message")
	if msgHdr == "" {
		return nil, thttp.NewHTTPError(
			"The server cannot process the request because the payload format is not supported. Please send the X-Agent-Message header.",
			http.StatusUnsupportedMediaType,
			http.Header{},
		)
	}
	return ic.codec, nil
}

var _ transport.InboundCodec = (*InboundCodec)(nil)

func NewInboundCodec(opts ...Option) transport.InboundCodec {
	cfg := config{}
	for _, option := range opts {
		option(&cfg)
	}
	if cfg.data == nil {
		cfg.data = chmessage.EmptyDataStreamer{}
	}
	return &InboundCodec{codec: &InboundAcceptCodec{data: cfg.data}}
}

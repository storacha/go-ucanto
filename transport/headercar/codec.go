package headercar

import (
	"net/http"

	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/transport"
	hcmsg "github.com/storacha/go-ucanto/transport/headercar/message"
	"github.com/storacha/go-ucanto/transport/headercar/request"
	"github.com/storacha/go-ucanto/transport/headercar/response"
	thttp "github.com/storacha/go-ucanto/transport/http"
)

type config struct {
	bodyProvider hcmsg.BodyProvider
}

type Option func(c *config)

func WithBodyProvider(provider hcmsg.BodyProvider) Option {
	return func(c *config) {
		c.bodyProvider = provider
	}
}

type OutboundCodec struct {
	bodyProvider hcmsg.BodyProvider
}

func (oc *OutboundCodec) Encode(msg message.AgentMessage) (transport.HTTPRequest, error) {
	return request.Encode(msg, request.WithBodyProvider(oc.bodyProvider))
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
	return &OutboundCodec{bodyProvider: cfg.bodyProvider}
}

type InboundAcceptCodec struct {
	bodyProvider hcmsg.BodyProvider
}

func (cic *InboundAcceptCodec) Encoder() transport.ResponseEncoder {
	return cic
}

func (cic *InboundAcceptCodec) Decoder() transport.RequestDecoder {
	return cic
}

func (cic *InboundAcceptCodec) Encode(msg message.AgentMessage) (transport.HTTPResponse, error) {
	return response.Encode(msg, response.WithBodyProvider(cic.bodyProvider))
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
	return &InboundCodec{codec: &InboundAcceptCodec{bodyProvider: cfg.bodyProvider}}
}

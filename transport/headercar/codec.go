package headercar

import (
	"net/http"

	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/transport"
	hcmsg "github.com/storacha/go-ucanto/transport/headercar/message"
	"github.com/storacha/go-ucanto/transport/headercar/request"
	"github.com/storacha/go-ucanto/transport/headercar/response"
	thttp "github.com/storacha/go-ucanto/transport/http"
)

type outboundConfig struct {
	bodyProvider hcmsg.RequestBodyProvider
}

type OutboundOption func(c *outboundConfig)

func WithRequestBodyProvider(provider hcmsg.RequestBodyProvider) OutboundOption {
	return func(c *outboundConfig) {
		c.bodyProvider = provider
	}
}

type OutboundCodec struct {
	bodyProvider hcmsg.RequestBodyProvider
}

func (oc *OutboundCodec) Encode(msg message.AgentMessage) (transport.HTTPRequest, error) {
	return request.Encode(msg, request.WithBodyProvider(oc.bodyProvider))
}

func (oc *OutboundCodec) Decode(res transport.HTTPResponse) (message.AgentMessage, error) {
	return response.Decode(res)
}

var _ transport.OutboundCodec = (*OutboundCodec)(nil)

func NewOutboundCodec(opts ...OutboundOption) transport.OutboundCodec {
	cfg := outboundConfig{}
	for _, option := range opts {
		option(&cfg)
	}
	return &OutboundCodec{bodyProvider: cfg.bodyProvider}
}

type InboundAcceptCodec struct {
	delegationCache delegation.Store
	bodyProvider    hcmsg.ResponseBodyProvider
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

type inboundConfig struct {
	bodyProvider hcmsg.ResponseBodyProvider
}

type InboundOption func(c *inboundConfig)

func WithResponseBodyProvider(provider hcmsg.ResponseBodyProvider) InboundOption {
	return func(c *inboundConfig) {
		c.bodyProvider = provider
	}
}

func NewInboundCodec(delegationCache delegation.Store, opts ...InboundOption) transport.InboundCodec {
	cfg := inboundConfig{}
	for _, option := range opts {
		option(&cfg)
	}
	return &InboundCodec{codec: &InboundAcceptCodec{delegationCache: delegationCache, bodyProvider: cfg.bodyProvider}}
}

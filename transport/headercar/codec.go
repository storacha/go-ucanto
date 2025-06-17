package car

import (
	"net/http"

	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/transport"
	"github.com/storacha/go-ucanto/transport/car/request"
	"github.com/storacha/go-ucanto/transport/car/response"
	thttp "github.com/storacha/go-ucanto/transport/http"
)

type carOutbound struct{}

func (oc *carOutbound) Encode(msg message.AgentMessage) (transport.HTTPRequest, error) {
	return request.Encode(msg)
}

func (oc *carOutbound) Decode(res transport.HTTPResponse) (message.AgentMessage, error) {
	return response.Decode(res)
}

var _ transport.OutboundCodec = (*carOutbound)(nil)

func NewOutboundCodec() transport.OutboundCodec {
	return &carOutbound{}
}

type carInboundAcceptCodec struct{}

func (cic *carInboundAcceptCodec) Encoder() transport.ResponseEncoder {
	return cic
}

func (cic *carInboundAcceptCodec) Decoder() transport.RequestDecoder {
	return cic
}

func (cic *carInboundAcceptCodec) Encode(msg message.AgentMessage) (transport.HTTPResponse, error) {
	return response.Encode(msg)
}

func (cic *carInboundAcceptCodec) Decode(req transport.HTTPRequest) (message.AgentMessage, error) {
	return request.Decode(req)
}

type carInbound struct {
	codec transport.InboundAcceptCodec
}

func (ic *carInbound) Accept(req transport.HTTPRequest) (transport.InboundAcceptCodec, transport.HTTPError) {
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

var _ transport.InboundCodec = (*carInbound)(nil)

func NewInboundCodec() transport.InboundCodec {
	return &carInbound{codec: &carInboundAcceptCodec{}}
}

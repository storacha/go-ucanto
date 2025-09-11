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

type OutboundCodec struct{}

func (oc *OutboundCodec) Encode(msg message.AgentMessage) (transport.HTTPRequest, error) {
	return request.Encode(msg)
}

func (oc *OutboundCodec) Decode(res transport.HTTPResponse) (message.AgentMessage, error) {
	return response.Decode(res)
}

var _ transport.OutboundCodec = (*OutboundCodec)(nil)

func NewOutboundCodec() *OutboundCodec {
	return &OutboundCodec{}
}

type InboundAcceptCodec struct{}

func (cic *InboundAcceptCodec) Encoder() transport.ResponseEncoder {
	return cic
}

func (cic *InboundAcceptCodec) Decoder() transport.RequestDecoder {
	return cic
}

func (cic *InboundAcceptCodec) Encode(msg message.AgentMessage) (transport.HTTPResponse, error) {
	return response.Encode(msg)
}

func (cic *InboundAcceptCodec) Decode(req transport.HTTPRequest) (message.AgentMessage, error) {
	return request.Decode(req)
}

type InboundCodec struct {
	codec transport.InboundAcceptCodec
}

func (ic *InboundCodec) Accept(req transport.HTTPRequest) (transport.InboundAcceptCodec, transport.HTTPError) {
	msgHdr := req.Headers().Get(hcmsg.HeaderName)
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

func NewInboundCodec() transport.InboundCodec {
	return &InboundCodec{codec: &InboundAcceptCodec{}}
}

package car

import (
	"net/http"
	"strings"

	"github.com/storacha-network/go-ucanto/core/message"
	"github.com/storacha-network/go-ucanto/transport"
	"github.com/storacha-network/go-ucanto/transport/car/request"
	"github.com/storacha-network/go-ucanto/transport/car/response"
	thttp "github.com/storacha-network/go-ucanto/transport/http"
)

type carOutbound struct{}

func (oc *carOutbound) Encode(msg message.AgentMessage) (transport.HTTPRequest, error) {
	return request.Encode(msg)
}

func (oc *carOutbound) Decode(res transport.HTTPResponse) (message.AgentMessage, error) {
	return response.Decode(res)
}

var _ transport.OutboundCodec = (*carOutbound)(nil)

func NewCAROutboundCodec() transport.OutboundCodec {
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
	// TODO: select a decoder - we only support 1 ATM
	contentType := req.Headers().Get("Content-Type")
	if contentType != request.ContentType {
		headers := http.Header{}
		headers.Set("Accept", contentType)
		return nil, thttp.NewHTTPError(
			"The server cannot process the request because the payload format is not supported. Please check the content-type header and try again with a supported media type.",
			http.StatusUnsupportedMediaType,
			headers,
		)
	}

	// TODO: select an encoder by desired preference (q=) - we only support 1 ATM
	accept := req.Headers().Get("Accept")
	if accept == "" {
		accept = "*/*"
	}
	if accept != "*/*" && !strings.Contains(accept, contentType) {
		headers := http.Header{}
		headers.Set("Accept", contentType)
		return nil, thttp.NewHTTPError(
			"The requested resource cannot be served in the requested content type. Please specify a supported content type using the Accept header.",
			http.StatusNotAcceptable,
			headers,
		)
	}

	return ic.codec, nil
}

var _ transport.InboundCodec = (*carInbound)(nil)

func NewCARInboundCodec() transport.InboundCodec {
	return &carInbound{codec: &carInboundAcceptCodec{}}
}

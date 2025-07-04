package request

import (
	"fmt"

	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/transport"
	chmessage "github.com/storacha/go-ucanto/transport/headercar/message"
	uhttp "github.com/storacha/go-ucanto/transport/http"
)

func Encode(msg message.AgentMessage, data chmessage.AgentMessageDataStreamer) (transport.HTTPRequest, error) {
	xAgentMsg, err := chmessage.EncodeHeader(msg)
	if err != nil {
		return nil, fmt.Errorf("encoding %s header: %w", chmessage.AgentMessageHeader, err)
	}

	body, headers, err := data.Stream(msg)
	if err != nil {
		return nil, fmt.Errorf("streaming data: %w", err)
	}
	headers.Set(chmessage.AgentMessageHeader, xAgentMsg)

	return uhttp.NewHTTPRequest(body, headers), nil
}

func Decode(req transport.HTTPRequest) (message.AgentMessage, error) {
	msgHdr := req.Headers().Get(chmessage.AgentMessageHeader)
	if msgHdr == "" {
		return nil, fmt.Errorf("missing %s header in request", chmessage.AgentMessageHeader)
	}
	return chmessage.DecodeHeader(msgHdr)
}

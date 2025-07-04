package response

import (
	"fmt"
	"net/http"

	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/transport"
	hcmsg "github.com/storacha/go-ucanto/transport/headercar/message"
	uhttp "github.com/storacha/go-ucanto/transport/http"
)

func Encode(msg message.AgentMessage, data hcmsg.AgentMessageDataStreamer) (transport.HTTPResponse, error) {
	xAgentMsg, err := hcmsg.EncodeHeader(msg)
	if err != nil {
		return nil, fmt.Errorf("encoding %s header: %w", hcmsg.AgentMessageHeader, err)
	}

	body, headers, err := data.Stream(msg)
	if err != nil {
		return nil, fmt.Errorf("streaming data: %w", err)
	}
	headers.Set(hcmsg.AgentMessageHeader, xAgentMsg)

	return uhttp.NewHTTPResponse(http.StatusOK, body, headers), nil
}

func Decode(response transport.HTTPResponse) (message.AgentMessage, error) {
	msgHdr := response.Headers().Get(hcmsg.AgentMessageHeader)
	if msgHdr == "" {
		return nil, fmt.Errorf("missing %s header in response", hcmsg.AgentMessageHeader)
	}
	return hcmsg.DecodeHeader(msgHdr)
}

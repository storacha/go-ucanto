package response

import (
	"fmt"
	"net/http"

	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/transport"
	hcmsg "github.com/storacha/go-ucanto/transport/headercar/message"
	uhttp "github.com/storacha/go-ucanto/transport/http"
)

func Encode(msg message.AgentMessage) (transport.HTTPResponse, error) {
	xAgentMsg, err := hcmsg.EncodeHeader(msg)
	if err != nil {
		return nil, fmt.Errorf("encoding %s header: %w", hcmsg.HeaderName, err)
	}
	headers := http.Header{}
	headers.Set(hcmsg.HeaderName, xAgentMsg)
	return uhttp.NewResponse(http.StatusOK, nil, headers), nil
}

func Decode(response transport.HTTPResponse) (message.AgentMessage, error) {
	msgHdr := response.Headers().Get(hcmsg.HeaderName)
	if msgHdr == "" {
		return nil, fmt.Errorf("missing %s header in response", hcmsg.HeaderName)
	}
	return hcmsg.DecodeHeader(msgHdr)
}

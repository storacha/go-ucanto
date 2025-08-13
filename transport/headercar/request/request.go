package request

import (
	"fmt"
	"net/http"

	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/transport"
	hcmsg "github.com/storacha/go-ucanto/transport/headercar/message"
	uhttp "github.com/storacha/go-ucanto/transport/http"
)

func Encode(msg message.AgentMessage) (transport.HTTPRequest, error) {
	xAgentMsg, err := hcmsg.EncodeHeader(msg)
	if err != nil {
		return nil, fmt.Errorf("encoding %s header: %w", hcmsg.AgentMessageHeader, err)
	}
	headers := http.Header{}
	headers.Set(hcmsg.AgentMessageHeader, xAgentMsg)
	return uhttp.NewRequest(nil, headers), nil
}

func Decode(req transport.HTTPRequest) (message.AgentMessage, error) {
	msgHdr := req.Headers().Get(hcmsg.AgentMessageHeader)
	if msgHdr == "" {
		return nil, fmt.Errorf("missing %s header in request", hcmsg.AgentMessageHeader)
	}
	return hcmsg.DecodeHeader(msgHdr)
}

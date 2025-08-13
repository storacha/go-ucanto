package response

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/transport"
	hcmsg "github.com/storacha/go-ucanto/transport/headercar/message"
	uhttp "github.com/storacha/go-ucanto/transport/http"
)

func Encode(msg message.AgentMessage) (transport.HTTPResponse, error) {
	xAgentMsg, err := hcmsg.EncodeHeader(msg)
	if err != nil {
		return nil, fmt.Errorf("encoding %s header: %w", hcmsg.AgentMessageHeader, err)
	}

	status := http.StatusOK
	receipts := msg.Receipts()
	if len(receipts) != 1 {
		return nil, errors.New("unexpected number of receipts in response")
	}
	rcpt, ok, err := msg.Receipt(receipts[0])
	if !ok {
		return nil, fmt.Errorf("missing receipt in agent message: %s", receipts[0])
	}
	if err != nil {
		return nil, fmt.Errorf("getting receipt: %s: %w", receipts[0], err)
	}
	_, x := result.Unwrap(rcpt.Out())
	if x != nil {
		status = http.StatusInternalServerError
		n, err := x.LookupByString("name")
		if err != nil {
			name, err := n.AsString()
			if err != nil {
				switch name {
				case "Unauthorized":
					status = http.StatusUnauthorized
				}
			}
		}
	}

	headers := http.Header{}
	headers.Set(hcmsg.AgentMessageHeader, xAgentMsg)

	return uhttp.NewResponse(status, nil, headers), nil
}

func Decode(response transport.HTTPResponse) (message.AgentMessage, error) {
	msgHdr := response.Headers().Get(hcmsg.AgentMessageHeader)
	if msgHdr == "" {
		return nil, fmt.Errorf("missing %s header in response", hcmsg.AgentMessageHeader)
	}
	return hcmsg.DecodeHeader(msgHdr)
}

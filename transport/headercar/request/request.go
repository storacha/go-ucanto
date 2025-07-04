package request

import (
	"fmt"
	"io"
	"net/http"

	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/transport"
	hcmsg "github.com/storacha/go-ucanto/transport/headercar/message"
	uhttp "github.com/storacha/go-ucanto/transport/http"
)

type encodeOptions struct {
	bodyProvider hcmsg.BodyProvider
}

type EncodeOption func(c *encodeOptions)

func WithBodyProvider(provider hcmsg.BodyProvider) EncodeOption {
	return func(c *encodeOptions) {
		c.bodyProvider = provider
	}
}

func Encode(msg message.AgentMessage, options ...EncodeOption) (transport.HTTPRequest, error) {
	opts := encodeOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	xAgentMsg, err := hcmsg.EncodeHeader(msg)
	if err != nil {
		return nil, fmt.Errorf("encoding %s header: %w", hcmsg.AgentMessageHeader, err)
	}

	var headers http.Header
	var body io.Reader
	if opts.bodyProvider != nil {
		b, h, err := opts.bodyProvider.Stream(msg)
		if err != nil {
			return nil, fmt.Errorf("streaming data: %w", err)
		}
		headers = h
		body = b
	} else {
		headers = http.Header{}
	}
	headers.Set(hcmsg.AgentMessageHeader, xAgentMsg)

	return uhttp.NewHTTPRequest(body, headers), nil
}

func Decode(req transport.HTTPRequest) (message.AgentMessage, error) {
	msgHdr := req.Headers().Get(hcmsg.AgentMessageHeader)
	if msgHdr == "" {
		return nil, fmt.Errorf("missing %s header in request", hcmsg.AgentMessageHeader)
	}
	return hcmsg.DecodeHeader(msgHdr)
}

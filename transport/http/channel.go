package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/storacha/go-ucanto/transport"
)

type channel struct {
	url    *url.URL
	client *http.Client
}

func (c *channel) Request(ctx context.Context, req transport.HTTPRequest) (transport.HTTPResponse, error) {
	hr, err := http.NewRequestWithContext(ctx, "POST", c.url.String(), req.Body())
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	hr.Header = req.Headers()
	res, err := c.client.Do(hr)
	if err != nil {
		return nil, fmt.Errorf("doing HTTP request: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, NewHTTPError(fmt.Sprintf("HTTP Request failed. %s %s → %d", hr.Method, c.url.String(), res.StatusCode), res.StatusCode, res.Header)
	}

	return NewHTTPResponse(res.StatusCode, res.Body, res.Header), nil
}

func NewHTTPChannel(url *url.URL) transport.Channel {
	return &channel{url: url, client: &http.Client{}}
}

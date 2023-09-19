package http

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/alanshaw/go-ucanto/transport"
)

type channel struct {
	url    *url.URL
	client *http.Client
}

func (c *channel) Request(req transport.HTTPRequest) (transport.HTTPResponse, error) {
	hr, err := http.NewRequest("POST", c.url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %s", err)
	}

	hr.Header = req.Headers()
	res, err := c.client.Do(hr)
	if err != nil {
		return nil, fmt.Errorf("doing HTTP request: %s", err)
	}

	return NewHTTPResponse(res.Body, res.Header), nil
}

func NewHTTPChannel(url *url.URL) transport.Channel {
	return &channel{url: url, client: &http.Client{}}
}

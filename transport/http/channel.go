package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"slices"

	"github.com/storacha/go-ucanto/transport"
)

// Option is an option configuring a HTTP channel.
type Option func(cfg *chanConfig)

type chanConfig struct {
	client   *http.Client
	method   string
	statuses []int
}

// WithClient configures the HTTP client the channel should use to make
// requests.
func WithClient(c *http.Client) Option {
	return func(cfg *chanConfig) {
		cfg.client = c
	}
}

// WithMethod configures the HTTP method the channel should use when making
// requests.
func WithMethod(method string) Option {
	return func(cfg *chanConfig) {
		cfg.method = method
	}
}

// WithSuccessStatusCode configures the HTTP status code(s) that will indicate a
// successful request.
func WithSuccessStatusCode(codes ...int) Option {
	return func(cfg *chanConfig) {
		cfg.statuses = codes
	}
}

type channel struct {
	url      *url.URL
	client   *http.Client
	method   string
	statuses []int
}

func (c *channel) Request(ctx context.Context, req transport.HTTPRequest) (transport.HTTPResponse, error) {
	hr, err := http.NewRequestWithContext(ctx, c.method, c.url.String(), req.Body())
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	hr.Header = req.Headers()
	res, err := c.client.Do(hr)
	if err != nil {
		return nil, fmt.Errorf("doing HTTP request: %w", err)
	}
	if !slices.Contains(c.statuses, res.StatusCode) {
		return nil, NewHTTPError(fmt.Sprintf("HTTP Request failed. %s %s → %d", hr.Method, c.url.String(), res.StatusCode), res.StatusCode, res.Header)
	}

	return NewHTTPResponse(res.StatusCode, res.Body, res.Header), nil
}

func NewHTTPChannel(url *url.URL, options ...Option) transport.Channel {
	cfg := chanConfig{}
	for _, opt := range options {
		opt(&cfg)
	}
	if cfg.client == nil {
		cfg.client = &http.Client{}
	}
	if cfg.method == "" {
		cfg.method = "POST"
	}
	if len(cfg.statuses) == 0 {
		cfg.statuses = append(cfg.statuses, http.StatusOK)
	}
	return &channel{
		url:      url,
		client:   cfg.client,
		method:   cfg.method,
		statuses: cfg.statuses,
	}
}

package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"slices"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/storacha/go-ucanto/transport"
)

// Option is an option configuring a HTTP channel.
type Option func(cfg *chanConfig)

type chanConfig struct {
	client   *http.Client
	method   string
	statuses []int
	headers  http.Header
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

// WithHeaders configures additional HTTP headers to send with requests.
func WithHeaders(h http.Header) Option {
	return func(cfg *chanConfig) {
		cfg.headers = h
	}
}

type Channel struct {
	url      *url.URL
	client   *http.Client
	headers  http.Header
	method   string
	statuses []int
}

func (c *Channel) Request(ctx context.Context, req transport.HTTPRequest) (transport.HTTPResponse, error) {
	hr, err := http.NewRequestWithContext(ctx, c.method, c.url.String(), req.Body())
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	addAllHeaders(hr.Header, req.Headers(), c.headers)
	injectTraceContext(ctx, hr)

	res, err := c.client.Do(hr)
	if err != nil {
		return nil, fmt.Errorf("doing HTTP request: %w", err)
	}
	if !slices.Contains(c.statuses, res.StatusCode) {
		return nil, NewHTTPError(fmt.Sprintf("HTTP Request failed. %s %s → %d", hr.Method, c.url.String(), res.StatusCode), res.StatusCode, res.Header)
	}

	ctx = extractTraceContext(ctx, res.Header)
	return NewResponseWithContext(ctx, res.StatusCode, res.Body, res.Header), nil
}

func addAllHeaders(dst http.Header, srcs ...http.Header) {
	for _, src := range srcs {
		for name, values := range src {
			for _, value := range values {
				dst.Add(name, value)
			}
		}
	}
}

func injectTraceContext(ctx context.Context, req *http.Request) {
	if ctx == nil || req == nil {
		return
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
}

func extractTraceContext(ctx context.Context, headers http.Header) context.Context {
	if ctx == nil || headers == nil {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(headers))
}

var _ transport.Channel = (*Channel)(nil)

func NewChannel(url *url.URL, options ...Option) *Channel {
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
	return &Channel{
		url:      url,
		client:   cfg.client,
		headers:  cfg.headers,
		method:   cfg.method,
		statuses: cfg.statuses,
	}
}

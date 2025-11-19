package http

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/storacha/go-ucanto/transport"
)

type Request struct {
	url  *url.URL
	hdrs http.Header
	body io.Reader
}

func (req *Request) URL() *url.URL {
	return req.url
}

func (req *Request) Headers() http.Header {
	return req.hdrs
}

func (req *Request) Body() io.Reader {
	return req.body
}

var _ transport.HTTPRequest = (*Request)(nil)
var _ transport.InboundHTTPRequest = (*Request)(nil)

type Response struct {
	ctx    context.Context
	status int
	hdrs   http.Header
	body   io.ReadCloser
}

func (res *Response) Status() int {
	return res.status
}

func (res *Response) Headers() http.Header {
	return res.hdrs
}

func (res *Response) Body() io.ReadCloser {
	return res.body
}

func (res *Response) Context() context.Context {
	if res.ctx == nil {
		return context.Background()
	}
	return res.ctx
}

var _ transport.HTTPResponse = (*Response)(nil)

func NewResponse(status int, body io.ReadCloser, headers http.Header) *Response {
	return NewResponseWithContext(context.Background(), status, body, headers)
}

func NewResponseWithContext(ctx context.Context, status int, body io.ReadCloser, headers http.Header) *Response {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Response{ctx: ctx, status: status, hdrs: headers, body: body}
}

// NewRequest creates a [transport.HTTPRequest]
func NewRequest(body io.Reader, headers http.Header) *Request {
	return &Request{nil, headers, body}
}

// NewInboundRequest creates a [transport.InboundHTTPRequest] - a request that
// also has a URL.
func NewInboundRequest(url *url.URL, body io.Reader, headers http.Header) *Request {
	return &Request{url, headers, body}
}

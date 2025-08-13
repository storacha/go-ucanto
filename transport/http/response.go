package http

import (
	"io"
	"net/http"

	"github.com/storacha/go-ucanto/transport"
)

type Request struct {
	url  string
	hdrs http.Header
	body io.Reader
}

func (req *Request) URL() string {
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
	status int
	hdrs   http.Header
	body   io.Reader
}

func (res *Response) Status() int {
	return res.status
}

func (res *Response) Headers() http.Header {
	return res.hdrs
}

func (res *Response) Body() io.Reader {
	return res.body
}

var _ transport.HTTPResponse = (*Response)(nil)

func NewResponse(status int, body io.Reader, headers http.Header) *Response {
	return &Response{status, headers, body}
}

// NewRequest creates a [transport.HTTPRequest]
func NewRequest(body io.Reader, headers http.Header) *Request {
	return &Request{"", headers, body}
}

// NewInboundRequest creates a [transport.InboundHTTPRequest] - a request that
// also has a URL.
func NewInboundRequest(url string, body io.Reader, headers http.Header) *Request {
	return &Request{url, headers, body}
}

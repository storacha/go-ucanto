package http

import (
	"io"
	"net/http"

	"github.com/storacha/go-ucanto/transport"
)

type request struct {
	hdrs http.Header
	body io.Reader
}

func (req *request) Headers() http.Header {
	return req.hdrs
}

func (req *request) Body() io.Reader {
	return req.body
}

type response struct {
	status int
	hdrs   http.Header
	body   io.Reader
}

func (res *response) Status() int {
	return res.status
}

func (res *response) Headers() http.Header {
	return res.hdrs
}

func (res *response) Body() io.Reader {
	return res.body
}

func NewHTTPResponse(status int, body io.Reader, headers http.Header) transport.HTTPResponse {
	return &response{status, headers, body}
}

func NewHTTPRequest(body io.Reader, headers http.Header) transport.HTTPRequest {
	return &request{headers, body}
}

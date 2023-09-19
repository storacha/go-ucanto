package http

import (
	"io"
	"net/http"

	"github.com/alanshaw/go-ucanto/transport"
)

type response struct {
	hdrs http.Header
	body io.Reader
}

func (res *response) Headers() http.Header {
	return res.hdrs
}

func (res *response) Body() io.Reader {
	return res.body
}

func NewHTTPResponse(body io.Reader, headers http.Header) transport.HTTPResponse {
	return &response{headers, body}
}

func NewHTTPRequest(body io.Reader, headers http.Header) transport.HTTPRequest {
	return &response{headers, body}
}

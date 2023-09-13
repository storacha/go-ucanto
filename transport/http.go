package transport

import (
	"io"
	"net/http"
)

type HTTPRequest interface {
	Headers() http.Header
	Body() io.Reader
}

type HTTPResponse interface {
	HTTPRequest
}

type httpResponse struct {
	hdrs http.Header
	body io.Reader
}

func (res *httpResponse) Headers() http.Header {
	return res.hdrs
}

func (res *httpResponse) Body() io.Reader {
	return res.body
}

func NewHTTPResponse(body io.Reader, headers http.Header) HTTPResponse {
	return &httpResponse{headers, body}
}

func NewHTTPRequest(body io.Reader, headers http.Header) HTTPRequest {
	return &httpResponse{headers, body}
}

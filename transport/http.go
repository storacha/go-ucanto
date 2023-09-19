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

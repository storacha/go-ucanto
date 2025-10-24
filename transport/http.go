package transport

import (
	"io"
	"net/http"
	"net/url"

	"github.com/storacha/go-ucanto/core/result/failure"
)

type HTTPRequest interface {
	Headers() http.Header
	Body() io.Reader
}

type InboundHTTPRequest interface {
	HTTPRequest
	URL() *url.URL
}

type HTTPResponse interface {
	Status() int
	Headers() http.Header
	Body() io.ReadCloser
}

type HTTPError interface {
	failure.Failure
	Status() int
	Headers() http.Header
}

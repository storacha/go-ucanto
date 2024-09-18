package transport

import (
	"io"
	"net/http"

	"github.com/storacha/go-ucanto/core/result/failure"
)

type HTTPRequest interface {
	Headers() http.Header
	Body() io.Reader
}

type HTTPResponse interface {
	Status() int
	Headers() http.Header
	Body() io.Reader
}

type HTTPError interface {
	failure.Failure
	Status() int
	Headers() http.Header
}

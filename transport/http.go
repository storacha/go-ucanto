package transport

import (
	"io"
	"net/http"

	"github.com/storacha-network/go-ucanto/core/result"
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
	result.Failure
	Status() int
	Headers() http.Header
}

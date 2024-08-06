package http

import (
	nethttp "net/http"

	"github.com/web3-storage/go-ucanto/transport"
)

type httpError struct {
	message string
	status  int
	headers nethttp.Header
}

func (err *httpError) Error() string {
	return err.message
}

func (err *httpError) Name() string {
	return "HTTPError"
}

func (err *httpError) Status() int {
	return err.status
}

func (err *httpError) Headers() nethttp.Header {
	return err.headers
}

func NewHTTPError(message string, status int, headers nethttp.Header) transport.HTTPError {
	return &httpError{message, status, headers}
}

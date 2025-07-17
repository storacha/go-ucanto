package message

import (
	"io"
	"net/http"

	"github.com/storacha/go-ucanto/core/message"
)

// RequestBodyProvider allows body data for HTTP request/response to be obtained.
// It optionally allows setting additional headers in the response.
type RequestBodyProvider interface {
	// Stream obtains streamable data corresponding to an agent message.
	Stream(message message.AgentMessage) (body io.Reader, headers http.Header, err error)
}

// ResponseBodyProvider allows body data for HTTP request/response to be
// obtained. It optionally allows setting HTTP status and additional headers in
// the response.
type ResponseBodyProvider interface {
	// Stream obtains streamable data corresponding to an agent message.
	Stream(message message.AgentMessage) (body io.Reader, status int, headers http.Header, err error)
}

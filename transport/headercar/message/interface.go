package message

import (
	"io"
	"net/http"

	"github.com/storacha/go-ucanto/core/message"
)

// BodyProvider allows body data for HTTP request/response to be obtained and
// streamed.
type BodyProvider interface {
	// Stream obtains streamable data corresponding to an agent message.
	Stream(message message.AgentMessage) (io.Reader, http.Header, error)
}

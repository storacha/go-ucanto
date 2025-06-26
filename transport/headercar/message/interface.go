package message

import (
	"io"
	"net/http"

	"github.com/storacha/go-ucanto/core/message"
)

// AgentMessageDataStreamer allows data to be obtained and streamed.
type AgentMessageDataStreamer interface {
	// Stream obtains streamable data for an agent message.
	Stream(message message.AgentMessage) (io.Reader, http.Header, error)
}

package message

import (
	"bytes"
	"io"
	"net/http"

	"github.com/storacha/go-ucanto/core/message"
)

type EmptyDataStreamer struct{}

func (eds EmptyDataStreamer) Stream(m message.AgentMessage) (io.Reader, http.Header, error) {
	headers := http.Header{}
	headers.Set("Content-Length", "0")
	return bytes.NewReader(nil), headers, nil
}

var _ AgentMessageDataStreamer = (*EmptyDataStreamer)(nil)

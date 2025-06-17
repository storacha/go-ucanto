package transport

import "context"

type Channel interface {
	Request(ctx context.Context, request HTTPRequest) (HTTPResponse, error)
}

package transport

type Channel interface {
	Request(request HTTPRequest) (HTTPResponse, error)
}

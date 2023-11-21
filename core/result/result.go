package result

// https://github.com/ucan-wg/invocation/#6-result
type Result[O any, X any] interface {
	Ok() O
	Error() X
}

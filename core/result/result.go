package result

// https://github.com/ucan-wg/invocation/#6-result
type Result interface {
	Ok() any
	Error() any
}

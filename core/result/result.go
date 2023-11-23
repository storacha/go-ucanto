package result

// https://github.com/ucan-wg/invocation/#6-result
type Result[O any, X any] interface {
	Ok() O
	Error() X
}

// type result[O any, X any] struct {
// 	ok  O
// 	err X
// }

// func (r result[O, X]) Ok() O {
// 	return r.ok
// }

// func (r result[O, X]) Error() X {
// 	return r.err
// }

// func Ok[O any](value O) Result[O, any] {
// 	return result[O, any]{value, nil}
// }

// func Error[X any](value X) Result[any, X] {
// 	return result[any, X]{nil, value}
// }

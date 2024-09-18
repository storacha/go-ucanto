package result

import (
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/result/failure"
	"github.com/storacha/go-ucanto/core/result/failure/datamodel"
)

// Result is a golang compatible generic result type
type Result[O any, X any] interface {
	isResult(ok O, err X)
}

type okResult[O any, X any] struct {
	value O
}
type errResult[O any, X any] struct {
	value X
}

func (o *okResult[O, X]) isResult(ok O, err X)  {}
func (e *errResult[O, X]) isResult(ok O, err X) {}

// MatchResultR3 handles a result with functions returning 3 values
func MatchResultR3[O any, X any, R0, R1, R2 any](
	result Result[O, X],
	onOk func(ok O) (R0, R1, R2),
	onError func(err X) (R0, R1, R2),
) (R0, R1, R2) {
	switch v := result.(type) {
	case *okResult[O, X]:
		return onOk(v.value)
	case *errResult[O, X]:
		return onError(v.value)
	default:
		panic("unexpected result type")
	}
}

// MatchResultR2 handles a result with functions returning two values
func MatchResultR2[O any, X any, R0, R1 any](
	result Result[O, X],
	onOk func(ok O) (R0, R1),
	onError func(err X) (R0, R1),
) (R0, R1) {
	switch v := result.(type) {
	case *okResult[O, X]:
		return onOk(v.value)
	case *errResult[O, X]:
		return onError(v.value)
	default:
		panic("unexpected result type")
	}
}

// MatchResultR1 handles a result with functions returning one value
func MatchResultR1[O any, X any, T0 any](
	result Result[O, X],
	onOk func(ok O) T0,
	onError func(err X) T0,
) T0 {
	switch v := result.(type) {
	case *okResult[O, X]:
		return onOk(v.value)
	case *errResult[O, X]:
		return onError(v.value)
	default:
		panic("unexpected result type")
	}
}

// MatchResultR1 handles a result with a functions that has no return value
func MatchResultR0[O any, X any](
	result Result[O, X],
	onOk func(ok O),
	onError func(err X),
) {
	switch v := result.(type) {
	case *okResult[O, X]:
		onOk(v.value)
	case *errResult[O, X]:
		onError(v.value)
	default:
		panic("unexpected result type")
	}
}

// Ok returns a success result type
func Ok[O, X any](value O) Result[O, X] {
	return &okResult[O, X]{value}
}

// Error returns an error result type
func Error[O, X any](value X) Result[O, X] {
	return &errResult[O, X]{value}
}

// MapOk transforms a successful result while leaving an error result unchanged
func MapOk[O, X, O2 any](result Result[O, X], mapFn func(O) O2) Result[O2, X] {
	return MapResultR0(result, mapFn, func(err X) X { return err })
}

// MapError transforms an error result while leaving a success result unchanged
func MapError[O, X, X2 any](result Result[O, X], mapFn func(X) X2) Result[O, X2] {
	return MapResultR0(result, func(ok O) O { return ok }, mapFn)
}

// MapResultR0 transforms a result --
// with seperate functions to modify both the success type and error type
func MapResultR0[O, X, O2, X2 any](result Result[O, X], mapOkFn func(O) O2, mapErrFn func(X) X2) Result[O2, X2] {
	return MatchResultR1(result, func(ok O) Result[O2, X2] {
		return Ok[O2, X2](mapOkFn(ok))
	}, func(err X) Result[O2, X2] {
		return Error[O2, X2](mapErrFn(err))
	})
}

// MapResultR1 transforms a result --
// with seperate functions to modify both the success type and error type that also returna one additional value
func MapResultR1[O, X, O2, X2, R1 any](result Result[O, X], mapOkFn func(O) (O2, R1), mapErrFn func(X) (X2, R1)) (Result[O2, X2], R1) {
	return MatchResultR2(result, func(ok O) (Result[O2, X2], R1) {
		ok2, r1 := mapOkFn(ok)
		return Ok[O2, X2](ok2), r1
	}, func(err X) (Result[O2, X2], R1) {
		err2, r1 := mapErrFn(err)
		return Error[O2, X2](err2), r1
	})
}

// And treats a result as a boolean, returning the second result only if the
// the first is succcessful
func And[O, O2, X any](res1 Result[O, X], res2 Result[O2, X]) Result[O2, X] {
	return AndThen(res1, func(_ O) Result[O2, X] { return res2 })
}

// AndThen takes a result and if it is success type,
// runs an additional function that returns a subsequent result type
func AndThen[O, X, O2 any](result Result[O, X], thenFunc func(O) Result[O2, X]) Result[O2, X] {
	return MatchResultR1(result, func(ok O) Result[O2, X] {
		return thenFunc(ok)
	}, func(err X) Result[O2, X] {
		return Error[O2, X](err)
	})
}

// Or treats a result as a boolean, returning the second result if the first
// result is an error
func Or[O, X, X2 any](res1 Result[O, X], res2 Result[O, X2]) Result[O, X2] {
	return OrElse(res1, func(err X) Result[O, X2] { return res2 })
}

// OrElse takes a result and if it is an error type,
// runs an additional function that returns a subsequent result type
func OrElse[O, X, X2 any](result Result[O, X], elseFunc func(X) Result[O, X2]) Result[O, X2] {
	return MatchResultR1(result, func(ok O) Result[O, X2] {
		return Ok[O, X2](ok)
	}, func(err X) Result[O, X2] {
		return elseFunc(err)
	})
}

// Wrap wraps a traditional golang pattern for two value functions with the
// second being an error where the zero value indicates absence, converting
// it to a result
func Wrap[O any, X comparable](inner func() (O, X)) Result[O, X] {
	o, err := inner()
	var nilErr X
	if err != nilErr {
		return Error[O, X](err)
	}
	return Ok[O, X](o)
}

func NewFailure(err error) Result[ipld.Builder, ipld.Builder] {
	if ipldConvertableError, ok := err.(failure.IPLDConvertableError); ok {
		return Error[ipld.Builder, ipld.Builder](ipldConvertableError)
	}

	model := datamodel.FailureModel{Message: err.Error()}
	if named, ok := err.(failure.Named); ok {
		name := named.Name()
		model.Name = &name
	}
	if withStackTrace, ok := err.(failure.WithStackTrace); ok {
		stack := withStackTrace.Stack()
		model.Stack = &stack
	}
	return Error[ipld.Builder, ipld.Builder](&model)
}

// https://en.wikipedia.org/wiki/Unit_type
type Unit interface{}

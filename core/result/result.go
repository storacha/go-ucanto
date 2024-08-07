package result

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/core/result/datamodel"
)

// https://github.com/ucan-wg/invocation/#6-result

type Result[O any, X any] interface {
	isResult(ok O, err X)
}

type okResult[O any, X any] struct {
	value O
}
type errResult[O any, X any] struct {
	value X
}

func (o *okResult[O, X]) isResult(ok O, err X) {}

func (e *errResult[O, X]) isResult(ok O, err X) {}

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

func Ok[O, X any](value O) Result[O, X] {
	return &okResult[O, X]{value}
}

func Error[O, X any](value X) Result[O, X] {
	return &errResult[O, X]{value}
}

func MapOk[O, X, O2 any](result Result[O, X], mapFn func(O) O2) Result[O2, X] {
	return MapResultR0(result, mapFn, func(err X) X { return err })
}

func MapError[O, X, X2 any](result Result[O, X], mapFn func(X) X2) Result[O, X2] {
	return MapResultR0(result, func(ok O) O { return ok }, mapFn)
}

func MapResultR0[O, X, O2, X2 any](result Result[O, X], mapOkFn func(O) O2, mapErrFn func(X) X2) Result[O2, X2] {
	return MatchResultR1(result, func(ok O) Result[O2, X2] {
		return Ok[O2, X2](mapOkFn(ok))
	}, func(err X) Result[O2, X2] {
		return Error[O2, X2](mapErrFn(err))
	})
}

func MapResultR1[O, X, O2, X2, R1 any](result Result[O, X], mapOkFn func(O) (O2, R1), mapErrFn func(X) (X2, R1)) (Result[O2, X2], R1) {
	return MatchResultR2(result, func(ok O) (Result[O2, X2], R1) {
		ok2, r1 := mapOkFn(ok)
		return Ok[O2, X2](ok2), r1
	}, func(err X) (Result[O2, X2], R1) {
		err2, r1 := mapErrFn(err)
		return Error[O2, X2](err2), r1
	})
}

// Named is an error that you can read a name from
type Named interface {
	Name() string
}

// WithStackTrace is an error that you can read a stack trace from
type WithStackTrace interface {
	Stack() string
}

// IPLDConvertableError is an error with a custom method to convert to an IPLD Node
type IPLDConvertableError interface {
	error
	ipld.Builder
}

type Failure interface {
	error
	Named
}

type NamedWithStackTrace interface {
	Named
	WithStackTrace
}

type namedWithStackTrace struct {
	name  string
	stack errors.StackTrace
}

func (n namedWithStackTrace) Name() string {
	return n.name
}

func (n namedWithStackTrace) Stack() string {
	return fmt.Sprintf("%+v", n.stack)
}

func NamedWithCurrentStackTrace(name string) NamedWithStackTrace {
	const depth = 32

	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])

	f := make(errors.StackTrace, n)
	for i := 0; i < n; i++ {
		f[i] = errors.Frame(pcs[i])
	}

	return namedWithStackTrace{name, f}
}

func NewFailure(err error) Result[ipld.Builder, ipld.Builder] {
	if ipldConvertableError, ok := err.(IPLDConvertableError); ok {
		return Error[ipld.Builder, ipld.Builder](ipldConvertableError)
	}

	failure := datamodel.Failure{Message: err.Error()}
	if named, ok := err.(Named); ok {
		name := named.Name()
		failure.Name = &name
	}
	if withStackTrace, ok := err.(WithStackTrace); ok {
		stack := withStackTrace.Stack()
		failure.Stack = &stack
	}
	return Error[ipld.Builder, ipld.Builder](&failure)
}

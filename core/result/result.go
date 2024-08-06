package result

import (
	"fmt"
	"runtime"

	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/pkg/errors"
	"github.com/web3-storage/go-ucanto/core/ipld"
	"github.com/web3-storage/go-ucanto/core/result/datamodel"
)

// https://github.com/ucan-wg/invocation/#6-result
type Result[O any, X any] interface {
	Ok() (O, bool)
	Error() (X, bool)
}

type AnyResult Result[ipld.Node, ipld.Node]

type anyResult struct {
	ok  *ipld.Node
	err *ipld.Node
}

func (ar anyResult) Ok() (ipld.Node, bool) {
	if ar.ok != nil {
		return *ar.ok, true
	}
	return nil, false
}

func (ar anyResult) Error() (ipld.Node, bool) {
	if ar.err != nil {
		return *ar.err, true
	}
	return nil, false
}

func Ok(value ipld.Node) AnyResult {
	return anyResult{&value, nil}
}

func Error(err ipld.Node) AnyResult {
	return anyResult{nil, &err}
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
	ToIPLD() ipld.Node
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

type Failure interface {
	error
	Named
}

// Failure generates a Result from a golang error, using:
//  1. a custom conversion to IPLD if present
//  2. the golangs error message plus
//     a. a name, if it is a named error
//     b. a stack trace, if it has a stack trace
func NewFailure(err error) AnyResult {
	if ipldConvertableError, ok := err.(IPLDConvertableError); ok {
		return Error(ipldConvertableError.ToIPLD())
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
	return Error(bindnode.Wrap(&failure, datamodel.Type()))
}

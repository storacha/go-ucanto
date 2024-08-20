package failure

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/core/result/failure/datamodel"
)

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

type failure struct {
	model datamodel.Failure
	build func() (ipld.Node, error)
}

func (f failure) Name() string {
	return *f.model.Name
}

func (f failure) Message() string {
	return f.model.Message
}

func (f failure) Error() string {
	return f.model.Message
}

func (f failure) Stack() string {
	return *f.model.Stack
}

func (f failure) Build() (ipld.Node, error) {
	if f.build != nil {
		return f.build()
	}
	return f.model.Build()
}

func FromError(err error) Failure {
	model := datamodel.Failure{Message: err.Error()}
	if named, ok := err.(Named); ok {
		name := named.Name()
		model.Name = &name
	}
	if withStackTrace, ok := err.(WithStackTrace); ok {
		stack := withStackTrace.Stack()
		model.Stack = &stack
	}
	fail := failure{model: model}
	if builder, ok := err.(ipld.Builder); ok {
		fail.build = builder.Build
	}
	return fail
}
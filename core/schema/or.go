package schema

import (
	"errors"
	"fmt"
	"strings"

	"github.com/storacha-network/go-ucanto/core/result/failure"
)

type unionError struct {
	failures []failure.Failure
}

func (ue unionError) Unwrap() []error {
	errors := make([]error, 0, len(ue.failures))
	for _, failure := range ue.failures {
		errors = append(errors, failure)
	}
	return errors
}

func indent(message string) string {
	indent := "  "
	return indent + strings.Join(strings.Split(message, "\n"), "\n"+indent)
}

func li(message string) string {
	return indent("- " + message)
}

func (ue unionError) Error() string {
	return fmt.Sprintf("Value does not match any type of the union:\n%s", li(errors.Join(ue.Unwrap()...).Error()))
}

func (ue unionError) Name() string {
	return "Union Error"
}

type orReader[I, O any] struct {
	readers []Reader[I, O]
}

func (or orReader[I, O]) Read(input I) (O, failure.Failure) {
	failures := make([]failure.Failure, 0, len(or.readers))
	for _, reader := range or.readers {
		o, err := reader.Read(input)
		if err != nil {
			failures = append(failures, err)
		} else {
			return o, nil
		}
	}
	var o O
	return o, failure.FromError(unionError{failures: failures})
}

func Or[I, O any](readers ...Reader[I, O]) Reader[I, O] {
	return orReader[I, O]{readers}
}

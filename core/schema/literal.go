package schema

import (
	"fmt"

	"github.com/storacha-network/go-ucanto/core/result/failure"
)

func Literal(expected string) Reader[string, string] {
	return reader[string, string]{
		readFunc: func(input string) (string, failure.Failure) {
			if input != expected {
				return "", NewSchemaError(fmt.Sprintf("expected literal %s instead got %s", expected, input))
			}
			return input, nil
		},
	}
}

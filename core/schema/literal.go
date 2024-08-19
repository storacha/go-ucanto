package schema

import (
	"fmt"

	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/core/result/failure"
)

func Literal(expected string) Reader[string, string] {
	return reader[string, string]{
		readFunc: func(input string) result.Result[string, failure.Failure] {
			if input != expected {
				return result.Error[string](NewSchemaError(fmt.Sprintf("expected literal %s instead got %s", expected, input)))
			}
			return result.Ok[string, failure.Failure](input)
		},
	}
}

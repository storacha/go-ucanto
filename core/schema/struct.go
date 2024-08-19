package schema

import (
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/core/policy"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/core/result/failure"
)

type strukt[T any] struct {
	typ    schema.Type
	policy policy.Policy
}

func (s strukt[T]) Read(input any) result.Result[T, failure.Failure] {
	node, ok := input.(ipld.Node)
	if !ok {
		return result.Error[T](NewSchemaError("unexpected input: not an IPLD node"))
	}

	if s.policy != nil {
		ok, err := policy.Match(s.policy, node)
		if err != nil {
			return result.Error[T](NewSchemaError(err.Error()))
		}
		if !ok {
			return result.Error[T](NewSchemaError("input did not match policy"))
		}
	}

	bind, err := ipld.Rebind[T](node, s.typ)
	if err != nil {
		return result.Error[T](NewSchemaError(err.Error()))
	}

	return result.Ok[T, failure.Failure](bind)
}

func Struct[T any](typ schema.Type, policy policy.Policy) Reader[any, T] {
	return strukt[T]{typ, policy}
}

package schema

import (
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/result/failure"
	"github.com/ucan-wg/go-ucan/capability/policy"
)

type strukt[T any] struct {
	typ    schema.Type
	policy policy.Policy
	opts   []bindnode.Option
}

func (s strukt[T]) Read(input any) (T, failure.Failure) {
	var bind T
	node, ok := input.(ipld.Node)
	if !ok {
		// If input is not an IPLD node, can it be converted to one?
		if builder, ok := input.(ipld.Builder); ok {
			n, err := builder.ToIPLD()
			if err != nil {
				return bind, NewSchemaError(err.Error())
			}
			node = n
		} else {
			return bind, NewSchemaError("unexpected input: not an IPLD node")
		}
	}

	if s.policy != nil {
		ok := policy.Match(s.policy, node)
		if !ok {
			return bind, NewSchemaError("input did not match policy")
		}
	}

	bind, err := ipld.Rebind[T](node, s.typ, s.opts...)
	if err != nil {
		return bind, NewSchemaError(err.Error())
	}

	return bind, nil
}

func Struct[T any](typ schema.Type, policy policy.Policy, opts ...bindnode.Option) Reader[any, T] {
	return strukt[T]{typ, policy, opts}
}

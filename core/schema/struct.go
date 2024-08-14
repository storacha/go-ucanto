package schema

import (
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/core/result"
)

type strukt[T any] struct {
	typ schema.Type
}

func (s *strukt[T]) Read(input any) result.Result[T, result.Failure] {
	node, ok := input.(ipld.Node)
	if !ok {
		return result.Error[T](NewSchemaError("unexpected input: not an IPLD node"))
	}

	bind, err := ipld.Rebind[T](node, s.typ)
	if err != nil {
		return result.Error[T](NewSchemaError(err.Error()))
	}

	return result.Ok[T, result.Failure](bind)
}

func Struct[T any](typ schema.Type) Reader[any, T] {
	return &strukt[T]{typ: typ}
}

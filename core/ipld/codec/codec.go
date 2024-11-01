package codec

import (
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
)

type Encoder interface {
	Code() uint64
	Encode(value any, typ schema.Type, opts ...bindnode.Option) ([]byte, error)
}

type Decoder interface {
	Code() uint64
	Decode(bytes []byte, bind any, typ schema.Type, opts ...bindnode.Option) error
}

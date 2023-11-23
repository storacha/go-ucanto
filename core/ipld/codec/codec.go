package codec

import (
	"github.com/ipld/go-ipld-prime/schema"
)

type Encoder interface {
	Code() uint64
	Encode(value any, typ schema.Type) ([]byte, error)
}

type Decoder interface {
	Code() uint64
	Decode(bytes []byte, bind any, typ schema.Type) error
}

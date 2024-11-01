package json

import (
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagjson"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
)

const Code = 0x0129

type codec struct{}

func (codec) Code() uint64 {
	return Code
}

func (codec) Encode(val any, typ schema.Type, opts ...bindnode.Option) ([]byte, error) {
	return Encode(val, typ, opts...)
}

func (codec) Decode(b []byte, bind any, typ schema.Type, opts ...bindnode.Option) error {
	return Decode(b, bind, typ, opts...)
}

var Codec = codec{}

func Encode(val any, typ schema.Type, opts ...bindnode.Option) ([]byte, error) {
	return ipld.Marshal(dagjson.Encode, val, typ, opts...)
}

func Decode(b []byte, bind any, typ schema.Type, opts ...bindnode.Option) error {
	_, err := ipld.Unmarshal(b, dagjson.Decode, bind, typ, opts...)
	return err
}

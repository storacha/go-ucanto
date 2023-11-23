package cbor

import (
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/schema"
)

const Code = 0x71

type codec struct{}

func (codec) Code() uint64 {
	return Code
}

func (codec) Encode(val any, typ schema.Type) ([]byte, error) {
	return Encode(val, typ)
}

func (codec) Decode(b []byte, bind any, typ schema.Type) error {
	return Decode(b, bind, typ)
}

var Codec = codec{}

func Encode(val any, typ schema.Type) ([]byte, error) {
	return ipld.Marshal(dagcbor.Encode, val, typ)
}

func Decode(b []byte, bind any, typ schema.Type) error {
	_, err := ipld.Unmarshal(b, dagcbor.Decode, bind, typ)
	return err
}

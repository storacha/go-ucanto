package cbor

import (
	"errors"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
)

const Code = 0x71

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

func Encode(val any, typ schema.Type, opts ...bindnode.Option) (bytes []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			if asStr, ok := r.(string); ok {
				err = errors.New(asStr)
			} else if asErr, ok := r.(error); ok {
				err = asErr
			} else {
				err = errors.New("unknown panic encoding CBOR")
			}
		}
	}()
	bytes, err = ipld.Marshal(dagcbor.Encode, val, typ, opts...)
	return
}

func Decode(b []byte, bind any, typ schema.Type, opts ...bindnode.Option) error {
	_, err := ipld.Unmarshal(b, dagcbor.Decode, bind, typ, opts...)
	return err
}

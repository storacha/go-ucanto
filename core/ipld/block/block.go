package block

import (
	"bytes"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/storacha/go-ucanto/core/ipld/codec"
	"github.com/storacha/go-ucanto/core/ipld/hash"
)

type Block interface {
	Link() ipld.Link
	Bytes() []byte
}

type block struct {
	link  ipld.Link
	bytes []byte
}

func (b *block) Link() ipld.Link {
	return b.link
}

func (b *block) Bytes() []byte {
	return b.bytes
}

func NewBlock(link ipld.Link, bytes []byte) Block {
	return &block{link, bytes}
}

func Encode(value any, typ schema.Type, codec codec.Encoder, hasher hash.Hasher, opts ...bindnode.Option) (Block, error) {
	b, err := codec.Encode(value, typ, opts...)
	if err != nil {
		return nil, err
	}

	d, err := hasher.Sum(b)
	if err != nil {
		return nil, err
	}

	l := cidlink.Link{Cid: cid.NewCidV1(codec.Code(), d.Bytes())}
	return NewBlock(l, b), nil
}

func Decode(block Block, bind any, typ schema.Type, codec codec.Decoder, hasher hash.Hasher, opts ...bindnode.Option) error {
	err := codec.Decode(block.Bytes(), bind, typ, opts...)
	if err != nil {
		return err
	}

	d, err := hasher.Sum(block.Bytes())
	if err != nil {
		return err
	}

	c := cid.NewCidV1(codec.Code(), d.Bytes())
	if !bytes.Equal(c.Bytes(), []byte(block.Link().Binary())) {
		return fmt.Errorf("data integrity error")
	}

	return nil
}

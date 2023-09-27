package block

import "github.com/ipld/go-ipld-prime"

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

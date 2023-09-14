package ipld

type Block interface {
	Link() Link
	Bytes() []byte
}

type block struct {
	link  Link
	bytes []byte
}

func (b *block) Link() Link {
	return b.link
}

func (b *block) Bytes() []byte {
	return b.bytes
}

func NewBlockUnsafe(link Link, bytes []byte) Block {
	return &block{link, bytes}
}

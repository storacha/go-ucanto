package hash

type Hasher interface {
	Sum(bytes []byte) (Digest, error)
}

type Digest interface {
	Code() uint64
	Size() uint64
	Digest() []byte
	Bytes() []byte
}

type digest struct {
	code   uint64
	size   uint64
	digest []byte
	bytes  []byte
}

func (d *digest) Bytes() []byte {
	return d.bytes
}

func (d *digest) Code() uint64 {
	return d.code
}

func (d *digest) Digest() []byte {
	return d.bytes
}

func (d *digest) Size() uint64 {
	return d.size
}

func NewDigest(code uint64, size uint64, digst []byte, bytes []byte) Digest {
	return &digest{code, size, digst, bytes}
}

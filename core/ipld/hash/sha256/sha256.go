package sha256

import (
	"crypto/sha256"

	"github.com/multiformats/go-multihash"
	"github.com/web3-storage/go-ucanto/core/ipld/hash"
)

// sha2-256
const Code = 0x12

// sha2-256 hash has a 32-byte sum
const Size = sha256.Size

type hasher struct{}

func (hasher) Code() uint64 {
	return Code
}

func (hasher) Size() uint64 {
	return Size
}

func (hasher) Sum(b []byte) (hash.Digest, error) {
	s256h := sha256.New()
	_, err := s256h.Write(b)
	if err != nil {
		return nil, err
	}
	sum := s256h.Sum(nil)

	d, _ := multihash.Encode(sum, Code)
	if err != nil {
		return nil, err
	}

	return hash.NewDigest(Code, Size, sum, d), nil
}

var Hasher = hasher{}

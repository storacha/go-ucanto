package helpers

import (
	crand "crypto/rand"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/datamodel"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"
)

// Must takes return values from a function and returns the non-error one. If
// the error value is non-nil then it panics.
func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

func RandomBytes(size int) []byte {
	bytes := make([]byte, size)
	_, _ = crand.Read(bytes)
	return bytes
}

func RandomCID() datamodel.Link {
	bytes := RandomBytes(10)
	c, _ := cid.Prefix{
		Version:  1,
		Codec:    cid.Raw,
		MhType:   multihash.SHA2_256,
		MhLength: -1,
	}.Sum(bytes)
	return cidlink.Link{Cid: c}
}

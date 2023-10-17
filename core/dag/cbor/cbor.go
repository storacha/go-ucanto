package cbor

import (
	"bytes"

	"github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/alanshaw/go-ucanto/core/ipld/block"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/datamodel"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/basicnode"
)

func Decode(b []byte) (datamodel.Node, error) {
	np := basicnode.Prototype.Any
	nb := np.NewBuilder()
	err := dagcbor.Decode(nb, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return nb.Build(), nil
}

// Instantiate a new block from the CBOR encoded data.
func NewBlock(b []byte) (ipld.Block, error) {
	pfx := cid.Prefix{
		Version:  1,
		Codec:    0x71, // dag-cbor
		MhType:   0x12, // sha2-256
		MhLength: 32,   // sha2-256 hash has a 32-byte sum
	}
	cid, err := pfx.Sum(b)
	if err != nil {
		return nil, err
	}
	return block.NewBlock(cidlink.Link{Cid: cid}, b), nil
}

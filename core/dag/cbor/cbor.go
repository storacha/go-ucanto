package cbor

import (
	"bytes"

	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/datamodel"
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

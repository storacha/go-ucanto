package receipt

import (
	"bytes"
	_ "embed"
	"fmt"
	"sync"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed receipt.ipldsch
var receipt []byte

var (
	once sync.Once
	ts   *schema.TypeSystem
	err  error
)

func loadSchema() (*schema.TypeSystem, error) {
	once.Do(func() {
		ts, err = ipld.LoadSchemaBytes(receipt)
		if err != nil {
			return
		}
	})
	return ts, err
}

type Receipt struct {
	Ocm *Ocm
	Sig []byte
}

type Ocm struct {
	Ran  ipld.Link
	Out  any
	Fx   Fx
	Meta MetaMap
	Iss  []byte
	Prf  []ipld.Link
}

type Fx struct {
	Fork []ipld.Link
	Join ipld.Link
}

type MetaMap struct {
	Keys   []string
	Values map[string]any
}

func Encode(r *Receipt) ([]byte, error) {
	ts, err := loadSchema()
	if err != nil {
		return nil, err
	}
	schemaType := ts.TypeByName("Receipt")
	node := bindnode.Wrap(&r, schemaType)
	var buf bytes.Buffer
	err = dagcbor.Encode(node.Representation(), &buf)
	if err != nil {
		return nil, fmt.Errorf("encoding dag-cbor: %s", err)
	}
	return buf.Bytes(), nil
}

func Decode(b []byte) (*Receipt, error) {
	ts, err := loadSchema()
	if err != nil {
		return nil, err
	}
	npt := bindnode.Prototype((*Receipt)(nil), ts.TypeByName("Receipt"))
	nb := npt.Representation().NewBuilder()
	err = dagcbor.Decode(nb, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	receipt := bindnode.Unwrap(nb.Build()).(*Receipt)
	return receipt, nil
}

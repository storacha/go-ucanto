package receipt

import (
	"fmt"

	"github.com/alanshaw/go-ucanto/core/dag/blockstore"
	"github.com/alanshaw/go-ucanto/core/invocation"
	"github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/alanshaw/go-ucanto/core/ipld/block"
	"github.com/alanshaw/go-ucanto/core/ipld/codec/cbor"
	"github.com/alanshaw/go-ucanto/core/ipld/hash/sha256"
	"github.com/alanshaw/go-ucanto/core/iterable"
	"github.com/alanshaw/go-ucanto/core/message"
	rdm "github.com/alanshaw/go-ucanto/core/receipt/datamodel"
	"github.com/alanshaw/go-ucanto/core/result"
	"github.com/alanshaw/go-ucanto/did"
	"github.com/alanshaw/go-ucanto/ucan"
	"github.com/alanshaw/go-ucanto/ucan/crypto/signature"
	"github.com/ipld/go-ipld-prime/schema"
)

type Effects interface {
	Fork() []ipld.Link
	Join() ipld.Link
}

type Receipt[O, X any] interface {
	ipld.IPLDView
	Ran() invocation.Invocation
	Out() result.Result[O, X]
	Fx() Effects
	Meta() map[string]any
	Issuer() ucan.Principal
	Signature() signature.SignatureView
}

type results[O, X any] struct {
	model *rdm.ResultModel[O, X]
}

func (r results[O, X]) Error() X {
	return r.model.Err
}

func (r results[O, X]) Ok() O {
	return r.model.Ok
}

type effects struct {
	model rdm.EffectsModel
}

func (fx effects) Fork() []ipld.Link {
	return fx.model.Fork
}

func (fx effects) Join() ipld.Link {
	return fx.model.Join
}

type receipt[O, X any] struct {
	rt   block.Block
	blks blockstore.BlockReader
	data *rdm.ReceiptModel[O, X]
}

var _ Receipt[any, any] = (*receipt[any, any])(nil)

func (r *receipt[O, X]) Blocks() iterable.Iterator[block.Block] {
	panic("TODO: iterate only: ran, fx, proofs, root")
	return r.blks.Iterator()
}

func (r *receipt[O, X]) Fx() Effects {
	return effects{r.data.Ocm.Fx}
}

func (r *receipt[O, X]) Issuer() ucan.Principal {
	principal, err := did.Decode(r.data.Ocm.Iss)
	if err != nil {
		fmt.Printf("Error: decoding issuer DID: %s\n", err)
	}
	return principal
}

// Map values are datamodel.Node
func (r *receipt[O, X]) Meta() map[string]any {
	meta := map[string]any{}
	for k, v := range r.data.Ocm.Meta.Values {
		meta[k] = any(v)
	}
	return meta
}

func (r *receipt[O, X]) Out() result.Result[O, X] {
	return results[O, X]{&r.data.Ocm.Out}
}

func (r *receipt[O, X]) Ran() invocation.Invocation {
	inv, err := invocation.NewInvocationView(r.data.Ocm.Ran, r.blks)
	if err != nil {
		fmt.Printf("Error: creating invocation view: %s\n", err)
	}
	return inv
}

func (r *receipt[O, X]) Root() block.Block {
	return r.rt
}

func (r *receipt[O, X]) Signature() signature.SignatureView {
	return signature.NewSignatureView(signature.Decode(r.data.Sig))
}

func NewReceipt[O, X any](root ipld.Link, blocks blockstore.BlockReader, typ schema.Type) (Receipt[O, X], error) {
	rblock, ok, err := blocks.Get(root)
	if err != nil {
		return nil, fmt.Errorf("getting receipt root block: %s", err)
	}
	if !ok {
		return nil, fmt.Errorf("missing receipt root block: %s", root)
	}

	rmdl := rdm.ReceiptModel[O, X]{}
	err = block.Decode(rblock, &rmdl, typ, cbor.Codec, sha256.Hasher)
	if err != nil {
		return nil, fmt.Errorf("decoding receipt: %s", err)
	}

	rcpt := receipt[O, X]{
		rt:   rblock,
		blks: blocks,
		data: &rmdl,
	}

	return &rcpt, nil
}

type ReceiptReader[O, X any] interface {
	Get(msg message.AgentMessage, root ipld.Link) (Receipt[O, X], error)
}

type receiptReader[O, X any] struct {
	typ schema.Type
}

func (d *receiptReader[O, X]) Get(msg message.AgentMessage, inv ipld.Link) (Receipt[O, X], error) {
	rl, ok := msg.Get(inv)
	if !ok {
		return nil, fmt.Errorf("missing receipt for invocation: %s", inv)
	}

	br, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(msg.Blocks()))
	if err != nil {
		return nil, fmt.Errorf("creating block reader: %s", err)
	}

	return NewReceipt[O, X](rl, br, d.typ)
}

func NewReceiptReader[O, X any](resultschema []byte) (ReceiptReader[O, X], error) {
	typ, err := rdm.NewReceiptModelType(resultschema)
	if err != nil {
		return nil, fmt.Errorf("loading receipt data model: %s", err)
	}
	return &receiptReader[O, X]{typ}, nil
}

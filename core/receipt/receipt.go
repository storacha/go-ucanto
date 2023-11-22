package receipt

import (
	"fmt"

	"github.com/alanshaw/go-ucanto/core/dag/blockstore"
	"github.com/alanshaw/go-ucanto/core/invocation"
	"github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/alanshaw/go-ucanto/core/ipld/block"
	"github.com/alanshaw/go-ucanto/core/iterable"
	rdm "github.com/alanshaw/go-ucanto/core/receipt/datamodel/receipt"
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

type Receipt[O any, X any] interface {
	ipld.IPLDView
	Ran() invocation.Invocation
	Out() result.Result[O, X]
	Fx() Effects
	Meta() map[string]any
	Issuer() ucan.Principal
	Signature() signature.SignatureView
}

type results[O any, X any] struct {
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

type receipt[O any, X any] struct {
	rt   block.Block
	blks blockstore.BlockReader
	data *rdm.ReceiptModel[O, X]
}

func (r *receipt[O, X]) Blocks() iterable.Iterator[block.Block] {
	return r.blks.Iterator()
}

func (r *receipt[O, X]) Fx() Effects {
	return effects{r.data.Ocm.Fx}
}

func (r *receipt[O, X]) Issuer() ucan.Principal {
	principal, _ := did.Decode(r.data.Ocm.Iss)
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
	return results[O, X]{r.data.Ocm.Out}
}

// Ran implements Receipt.
func (receipt[O, X]) Ran() invocation.Invocation {
	panic("unimplemented")
}

func (r *receipt[O, X]) Root() block.Block {
	return r.rt
}

func (r *receipt[O, X]) Signature() signature.SignatureView {
	return signature.NewSignatureView(signature.Decode(r.data.Sig))
}

func NewReceipt[O any, X any](root ipld.Link, blocks blockstore.BlockReader, typ schema.Type) (Receipt[O, X], error) {
	block, ok, err := blocks.Get(root)
	if err != nil {
		return nil, fmt.Errorf("getting receipt root block: %s", err)
	}
	if !ok {
		return nil, fmt.Errorf("missing receipt root block: %s", root)
	}

	data, err := rdm.Decode[O, X](block.Bytes(), typ)
	if err != nil {
		return nil, fmt.Errorf("decoding receipt: %s", err)
	}

	rcpt := receipt[O, X]{
		rt:   block,
		blks: blocks,
		data: data,
	}

	return &rcpt, nil
}

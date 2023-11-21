package receipt

import (
	"fmt"

	"github.com/alanshaw/go-ucanto/core/dag/blockstore"
	"github.com/alanshaw/go-ucanto/core/invocation"
	"github.com/alanshaw/go-ucanto/core/ipld"
	rdm "github.com/alanshaw/go-ucanto/core/receipt/datamodel/receipt"
	"github.com/alanshaw/go-ucanto/core/result"
	"github.com/alanshaw/go-ucanto/ucan"
	"github.com/alanshaw/go-ucanto/ucan/crypto"
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
	Signature() crypto.Signature
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

	return nil, nil
}

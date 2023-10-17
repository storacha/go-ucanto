package receipt

import (
	"fmt"

	"github.com/alanshaw/go-ucanto/core/dag/blockstore"
	"github.com/alanshaw/go-ucanto/core/invocation"
	"github.com/alanshaw/go-ucanto/core/ipld"
	schema "github.com/alanshaw/go-ucanto/core/receipt/schema/receipt"
	"github.com/alanshaw/go-ucanto/core/result"
	"github.com/alanshaw/go-ucanto/ucan"
	"github.com/alanshaw/go-ucanto/ucan/crypto"
)

type Effects interface {
	Fork() []ipld.Link
	Join() ipld.Link
}

type Receipt interface {
	ipld.IPLDView
	Ran() invocation.Invocation
	Out() result.Result
	Fx() Effects
	Meta() map[string]any
	Issuer() ucan.Principal
	Signature() crypto.Signature
}

func NewReceipt(root ipld.Link, blocks blockstore.BlockReader) (Receipt, error) {
	block, ok, err := blocks.Get(root)
	if err != nil {
		return nil, fmt.Errorf("getting receipt root block: %s", err)
	}
	if !ok {
		return nil, fmt.Errorf("missing receipt root block: %s", root)
	}

	data, err := schema.Decode(block.Bytes())
	if err != nil {
		return nil, fmt.Errorf("decoding message: %s", err)
	}

	return nil, nil
}

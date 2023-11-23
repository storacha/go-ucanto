package invocation

import (
	"github.com/alanshaw/go-ucanto/core/dag/blockstore"
	"github.com/alanshaw/go-ucanto/core/delegation"
	"github.com/alanshaw/go-ucanto/core/ipld"
)

// Invocation represents a UCAN that can be presented to a service provider to
// invoke or "exercise" a Capability. You can think of invocations as a
// serialized function call, where the ability or `can` portion of the
// Capability acts as the function name, and the resource (`with`) and caveats
// (`nb`) of the capability act as function arguments.
//
// Most invocations will require valid proofs, which consist of a chain of
// Delegations. The service provider will inspect the proofs to verify that the
// invocation has sufficient privileges to execute.
type Invocation interface {
	delegation.Delegation
}

func NewInvocation(root ipld.Block, bs blockstore.BlockReader) Invocation {
	return delegation.NewDelegation(root, bs)
}

func NewInvocationView(root ipld.Link, bs blockstore.BlockReader) (Invocation, error) {
	return delegation.NewDelegationView(root, bs)
}

type IssuedInvocation interface {
	// TODO?
}

// TODO
func Invoke() IssuedInvocation {
	return nil
}

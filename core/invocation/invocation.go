package invocation

import (
	"github.com/web3-storage/go-ucanto/core/dag/blockstore"
	"github.com/web3-storage/go-ucanto/core/delegation"
	"github.com/web3-storage/go-ucanto/core/ipld"
	"github.com/web3-storage/go-ucanto/ucan"
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
	Invocation
}

func Invoke(issuer ucan.Signer, audience ucan.Principal, capability ucan.Capability[ucan.CaveatBuilder], options ...delegation.Option) (IssuedInvocation, error) {
	return delegation.Delegate(issuer, audience, []ucan.Capability[ucan.CaveatBuilder]{capability}, options...)
}

type Ran struct {
	invocation Invocation
	link       ucan.Link
}

func (r Ran) Invocation() (Invocation, bool) {
	return r.invocation, r.invocation != nil
}

func (r Ran) Link() ucan.Link {
	if r.invocation != nil {
		return r.invocation.Link()
	}
	return r.link
}

func FromInvocation(invocation Invocation) Ran {
	return Ran{invocation, nil}
}

func FromLink(link ucan.Link) Ran {
	return Ran{nil, link}
}

func (r Ran) WriteInto(bs blockstore.BlockWriter) (ipld.Link, error) {
	if invocation, ok := r.Invocation(); ok {
		return r.Link(), blockstore.WriteInto(invocation, bs)
	}
	return r.Link(), nil
}

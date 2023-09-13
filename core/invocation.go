package core

import (
	"github.com/alanshaw/go-ucanto/core/ipld"
	ucan "github.com/alanshaw/go-ucanto/ucan"
)

// Delagation is a materialized view of a UCAN delegation, which can be encoded
// into a UCAN token and used as proof for an invocation or further delegations.
type Delegation interface {
	// TODO
}

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
	ipld.View
	// Link returns the IPLD link of the root block of the invocation.
	Link() ucan.Link
	// Archive writes the invocation to a Content Addressed aRchive (CAR).
	// Archive() io.Reader
}

type IssuedInvocation interface {
	// TODO?
}

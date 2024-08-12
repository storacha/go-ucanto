package validator

import (
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/did"
	"github.com/storacha-network/go-ucanto/principal"
	"github.com/storacha-network/go-ucanto/ucan"
)

// PrincipalParser provides verifier instances that can validate UCANs issued
// by a given principal.
type PrincipalParser interface {
	Parse(str string) (principal.Verifier, error)
}

type CanIssuer[Caveats any] interface {
	// CanIssue informs validator whether given capability can be issued by a
	// given DID or whether it needs to be delegated to the issuer.
	CanIssue(capability ucan.Capability[Caveats], issuer did.DID) bool
}

type RevocationChecker[Caveats any] interface {
	// ValidateAuthorization validates that the passed authorization has not been
	// revoked.
	ValidateAuthorization(auth Authorization[Caveats]) result.Failure
}

type ValidationContext[Caveats any] interface {
	RevocationChecker[Caveats]
	CanIssuer[Caveats]
}

type Authorization[Caveats any] interface {
	Capability() ucan.Capability[Caveats]
}

type authorization[Caveats any] struct {
	capability ucan.Capability[Caveats]
}

func (a *authorization[Caveats]) Capability() ucan.Capability[Caveats] {
	return a.capability
}

func Access[Caveats any](invocation invocation.Invocation, context ValidationContext[Caveats]) (result.Result[Authorization[Caveats], result.Failure], error) {
	cap := invocation.Capabilities()[0]

	auth := authorization[Caveats]{capability: ucan.NewCapability(cap.Can(), cap.With(), nb)}
	return result.Ok[Authorization[Caveats], result.Failure](&auth), nil
}

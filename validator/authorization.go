package validator

import (
	"github.com/storacha-network/go-ucanto/core/delegation"
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

// CanIssue informs validator whether given capability can be issued by a
// given DID or whether it needs to be delegated to the issuer.
type CanIssueFunc[Caveats any] func(capability ucan.Capability[Caveats], issuer did.DID) bool

type RevocationChecker[Caveats any] interface {
	// ValidateAuthorization validates that the passed authorization has not been
	// revoked.
	ValidateAuthorization(auth Authorization[Caveats]) result.Failure
}

// RevocationCheckerFunc validates the passed authorization and returns
// a result indicating validity.
type RevocationCheckerFunc[Caveats any] func(auth Authorization[Caveats]) result.Failure

type ValidationContext[Caveats any] interface {
	RevocationChecker[Caveats]
	CanIssuer[Caveats]
	Capability() CapabilityParser[Caveats]
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

func NewAuthorization[Caveats any](capability ucan.Capability[Caveats]) Authorization[Caveats] {
	return &authorization[Caveats]{capability: capability}
}

type source struct {
	capability ucan.Capability[any]
}

func (s *source) Capability() ucan.Capability[any] {
	return s.capability
}

func (s *source) Delegation() delegation.Delegation {
	return nil
}

type validationContext[Caveats any] struct {
	capability            CapabilityParser[Caveats]
	canIssue              CanIssueFunc[Caveats]
	validateAuthorization RevocationCheckerFunc[Caveats]
}

func (vc *validationContext[Caveats]) CanIssue(capability ucan.Capability[Caveats], issuer did.DID) bool {
	return vc.canIssue(capability, issuer)
}

func (vc *validationContext[Caveats]) ValidateAuthorization(auth Authorization[Caveats]) result.Failure {
	return vc.validateAuthorization(auth)
}

func (vc *validationContext[Caveats]) Capability() CapabilityParser[Caveats] {
	return vc.capability
}

var _ ValidationContext[any] = (*validationContext[any])(nil)

func NewValidationContext[Caveats any](capability CapabilityParser[Caveats], canIssue CanIssueFunc[Caveats], validateAuthorization RevocationCheckerFunc[Caveats]) ValidationContext[Caveats] {
	vc := validationContext[Caveats]{capability, canIssue, validateAuthorization}
	return &vc
}

func Access[Caveats any](invocation invocation.Invocation, context ValidationContext[Caveats]) (result.Result[Authorization[Caveats], result.Failure], error) {
	cap := invocation.Capabilities()[0]
	src := source{capability: cap}

	// TODO: parser.Select()
	match := context.Capability().Match(&src)

	return result.MapOk(match, func(o ucan.Capability[Caveats]) Authorization[Caveats] {
		return &authorization[Caveats]{capability: o}
	}), nil
}

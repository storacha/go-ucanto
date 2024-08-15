package validator

import (
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/did"
	"github.com/storacha-network/go-ucanto/principal"
	"github.com/storacha-network/go-ucanto/ucan"
)

func IsSelfIssued[Caveats any](capability ucan.Capability[Caveats], issuer did.DID) bool {
	return capability.With() == issuer.DID().String()
}

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

type ClaimContext interface {
	RevocationChecker[any]
	CanIssuer[any]
}

type ValidationContext[Caveats any] interface {
	RevocationChecker[any]
	CanIssuer[any]
	Capability() CapabilityParser[Caveats]
}

type validationContext[Caveats any] struct {
	capability            CapabilityParser[Caveats]
	canIssue              CanIssueFunc[any]
	validateAuthorization RevocationCheckerFunc[any]
}

func (vc validationContext[Caveats]) CanIssue(capability ucan.Capability[any], issuer did.DID) bool {
	return vc.canIssue(capability, issuer)
}

func (vc validationContext[Caveats]) ValidateAuthorization(auth Authorization[any]) result.Failure {
	return vc.validateAuthorization(auth)
}

func (vc validationContext[Caveats]) Capability() CapabilityParser[Caveats] {
	return vc.capability
}

func NewValidationContext[Caveats any](capability CapabilityParser[Caveats], canIssue CanIssueFunc[any], validateAuthorization RevocationCheckerFunc[any]) ValidationContext[Caveats] {
	return validationContext[Caveats]{capability, canIssue, validateAuthorization}
}

// Access finds a valid path in a proof chain of the given `invocation` by
// exploring every possible option. On success an `Authorization` object is
// returned that illustrates the valid path. If no valid path is found
// `Unauthorized` error is returned detailing all explored paths and where they
// proved to fail.
func Access[Caveats any](invocation invocation.Invocation, context ValidationContext[Caveats]) (result.Result[Authorization[Caveats], result.Failure], error) {
	cap := invocation.Capabilities()[0]
	src := source{capability: cap}

	// TODO: parser.Select()
	match := context.Capability().Match(src)

	return result.MapOk(match, func(o ucan.Capability[Caveats]) Authorization[Caveats] {
		return authorization[Caveats]{capability: o}
	}), nil
}

// Claim attempts to find a valid proof chain for the claimed `capability` given
// set of `proofs`. On success an `Authorization` object with detailed proof
// chain is returned and on failure `Unauthorized` error is returned with
// details on paths explored and why they have failed.
// func Claim()

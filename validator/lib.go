package validator

import (
	"fmt"

	"github.com/storacha-network/go-ucanto/core/delegation"
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/did"
	"github.com/storacha-network/go-ucanto/principal"
	"github.com/storacha-network/go-ucanto/ucan"
)

func IsSelfIssued[Caveats any](capability ucan.Capability[Caveats], issuer did.DID) bool {
	return capability.With() == issuer.DID().String()
}

func ProofUnavailable(p ucan.Link) result.Result[delegation.Delegation, UnavailableProofError] {
	return result.Error[delegation.Delegation](NewUnavailableProofError(p, fmt.Errorf("no proof resolver configured")))
}

func FailDIDKeyResolution(d did.DID) result.Result[did.DID, DIDKeyResolutionError] {
	return result.Error[did.DID](NewDIDKeyResolutionError(d, fmt.Errorf("no DID resolver configured")))
}

// PrincipalParser provides verifier instances that can validate UCANs issued
// by a given principal.
type PrincipalParser interface {
	ParsePrincipal(str string) (principal.Verifier, error)
}

type PrincipalParserFunc = func(str string) (principal.Verifier, error)

// PrincipalResolver is used to resolve a key of the principal that is
// identified by DID different from did:key method. It can be passed into a
// UCAN validator in order to augmented it with additional DID methods support.
type PrincipalResolver interface {
	ResolveDIDKey(did did.DID) result.Result[did.DID, DIDKeyResolutionError]
}

// PrincipalResolverFunc resolves the key of a principal that is identified by
// DID different from did:key method.
type PrincipalResolverFunc = func(did did.DID) result.Result[did.DID, DIDKeyResolutionError]

// ProofResolver finds a delegations when external proof links are present in
// UCANs. If a resolver is not provided the validator may not be able to explore
// corresponding path within a proof chain.
type ProofResolver interface {
	// Resolve finds a delegation corresponding to an external proof link.
	ResolveProof(proof ucan.Link) result.Result[delegation.Delegation, UnavailableProofError]
}

// Resolve finds a delegation corresponding to an external proof link.
type ProofResolverFunc = func(proof ucan.Link) result.Result[delegation.Delegation, UnavailableProofError]

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
	ProofResolver
	PrincipalParser
	PrincipalResolver
}

type ValidationContext[Caveats any] interface {
	ClaimContext
	Capability() CapabilityParser[Caveats]
}

type validationContext[Caveats any] struct {
	capability            CapabilityParser[Caveats]
	canIssue              CanIssueFunc[any]
	validateAuthorization RevocationCheckerFunc[any]
	resolveProof          ProofResolverFunc
	parsePrincipal        PrincipalParserFunc
	resolveDIDKey         PrincipalResolverFunc
}

func (vc validationContext[Caveats]) Capability() CapabilityParser[Caveats] {
	return vc.capability
}

func (vc validationContext[Caveats]) CanIssue(capability ucan.Capability[any], issuer did.DID) bool {
	return vc.canIssue(capability, issuer)
}

func (vc validationContext[Caveats]) ValidateAuthorization(auth Authorization[any]) result.Failure {
	return vc.validateAuthorization(auth)
}

func (vc validationContext[Caveats]) ResolveProof(proof ucan.Link) result.Result[delegation.Delegation, UnavailableProofError] {
	return vc.resolveProof(proof)
}

func (vc validationContext[Caveats]) ParsePrincipal(str string) (principal.Verifier, error) {
	return vc.parsePrincipal(str)
}

func (vc validationContext[Caveats]) ResolveDIDKey(did did.DID) result.Result[did.DID, DIDKeyResolutionError] {
	return vc.resolveDIDKey(did)
}

func NewValidationContext[Caveats any](
	capability CapabilityParser[Caveats],
	canIssue CanIssueFunc[any],
	validateAuthorization RevocationCheckerFunc[any],
	resolveProof ProofResolverFunc,
	parsePrincipal PrincipalParserFunc,
	resolveDIDKey PrincipalResolverFunc,
) ValidationContext[Caveats] {
	return validationContext[Caveats]{
		capability,
		canIssue,
		validateAuthorization,
		resolveProof,
		parsePrincipal,
		resolveDIDKey,
	}
}

// Access finds a valid path in a proof chain of the given `invocation` by
// exploring every possible option. On success an `Authorization` object is
// returned that illustrates the valid path. If no valid path is found
// `Unauthorized` error is returned detailing all explored paths and where they
// proved to fail.
func Access[Caveats any](invocation invocation.Invocation, context ValidationContext[Caveats]) (result.Result[Authorization[Caveats], UnauthorizedError[Caveats]], error) {
	prf := []delegation.Proof{delegation.FromDelegation(invocation)}
	return Claim(context.Capability(), prf, context)
}

// Claim attempts to find a valid proof chain for the claimed `capability` given
// set of `proofs`. On success an `Authorization` object with detailed proof
// chain is returned and on failure `Unauthorized` error is returned with
// details on paths explored and why they have failed.
func Claim[Caveats any](capability CapabilityParser[Caveats], proofs []delegation.Proof, context ClaimContext) (result.Result[Authorization[Caveats], UnauthorizedError[Caveats]], error) {
	delegations, errors := resolveProofs(proofs, context)

	for _, d := range delegations {
		// Validate each proof if valid add each capability to the list of sources.
		// otherwise collect the error.
		result.MatchResultR0(
			validate(d, delegations, context),
		)
	}

	// cap := invocation.Capabilities()[0]

	// var sources []Source

	// src := source{capability: cap}

	// // TODO: parser.Select()
	// match := context.Capability().Match(src)

	// return result.MapOk(match, func(o ucan.Capability[Caveats]) Authorization[Caveats] {
	// 	return authorization[Caveats]{capability: o}
	// }), nil
}

// resolveProofs takes `proofs` from the delegation which may contain
// a `Delegation` or a link to one and attempts to resolve links by side loading
// them. It returns a set of resolved `Delegation`s and errors for the proofs
// that could not be resolved.
func resolveProofs(proofs []delegation.Proof, resolver ProofResolver) (dels []delegation.Delegation, errs []UnavailableProofError) {
	for _, p := range proofs {
		d, ok := p.Delegation()
		if ok {
			dels = append(dels, d)
		} else {
			result.MatchResultR0(
				resolver.ResolveProof(p.Link()),
				func(d delegation.Delegation) { dels = append(dels, d) },
				func(err UnavailableProofError) { errs = append(errs, err) },
			)
		}
	}
	return
}

// Validate a delegation to check it is within the time bound and that it is
// authorized by the issuer.
func validate(delegation delegation.Delegation, proofs []delegation.Delegation, context ClaimContext) {
	if ucan.IsExpired(delegation) {

	}
}

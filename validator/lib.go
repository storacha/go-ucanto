package validator

import (
	"fmt"
	"strings"

	"github.com/storacha-network/go-ucanto/core/delegation"
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/policy"
	"github.com/storacha-network/go-ucanto/core/policy/literal"
	"github.com/storacha-network/go-ucanto/core/policy/selector"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/core/schema"
	"github.com/storacha-network/go-ucanto/did"
	"github.com/storacha-network/go-ucanto/principal"
	"github.com/storacha-network/go-ucanto/ucan"
	vdm "github.com/storacha-network/go-ucanto/validator/datamodel"
)

func IsSelfIssued[Caveats any](capability ucan.Capability[Caveats], issuer did.DID) bool {
	return capability.With() == issuer.DID().String()
}

func ProofUnavailable(p ucan.Link) result.Result[delegation.Delegation, UnavailableProof] {
	return result.Error[delegation.Delegation](NewUnavailableProofError(p, fmt.Errorf("no proof resolver configured")))
}

func FailDIDKeyResolution(d did.DID) result.Result[did.DID, UnresolvedDID] {
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
	ResolveDIDKey(did did.DID) result.Result[did.DID, UnresolvedDID]
}

// PrincipalResolverFunc resolves the key of a principal that is identified by
// DID different from did:key method.
type PrincipalResolverFunc = func(did did.DID) result.Result[did.DID, UnresolvedDID]

// ProofResolver finds a delegations when external proof links are present in
// UCANs. If a resolver is not provided the validator may not be able to explore
// corresponding path within a proof chain.
type ProofResolver interface {
	// Resolve finds a delegation corresponding to an external proof link.
	ResolveProof(proof ucan.Link) result.Result[delegation.Delegation, UnavailableProof]
}

// Resolve finds a delegation corresponding to an external proof link.
type ProofResolverFunc = func(proof ucan.Link) result.Result[delegation.Delegation, UnavailableProof]

type CanIssuer[Caveats any] interface {
	// CanIssue informs validator whether given capability can be issued by a
	// given DID or whether it needs to be delegated to the issuer.
	CanIssue(capability ucan.Capability[Caveats], issuer did.DID) bool
}

// CanIssue informs validator whether given capability can be issued by a
// given DID or whether it needs to be delegated to the issuer.
type CanIssueFunc[Caveats any] func(capability ucan.Capability[Caveats], issuer did.DID) bool

// canissuer converts an CanIssuer[any] to CanIssuer[Caveats]
type canissuer[Caveats any] struct {
	canIssue CanIssueFunc[any]
}

func (ci canissuer[Caveats]) CanIssue(c ucan.Capability[Caveats], d did.DID) bool {
	return ci.canIssue(ucan.NewCapability[any](c.Can(), c.With(), c.Nb()), d)
}

type RevocationChecker[Caveats any] interface {
	// ValidateAuthorization validates that the passed authorization has not been
	// revoked.
	ValidateAuthorization(auth Authorization[Caveats]) result.Result[result.Unit, Revoked]
}

// RevocationCheckerFunc validates the passed authorization and returns
// a result indicating validity.
type RevocationCheckerFunc[Caveats any] func(auth Authorization[Caveats]) result.Result[result.Unit, Revoked]

// Validator must provide a `Verifier` corresponding to local authority.
//
// A capability provider service will use one corresponding to own DID or it's
// supervisor's DID if it acts under it's authority.
//
// This allows a service identified by non did:key e.g. did:web or did:dns to
// pass resolved key so it does not need to be resolved at runtime.
type Validator interface {
	Authority() principal.Verifier
}

type ClaimContext interface {
	Validator
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
	authority             principal.Verifier
	capability            CapabilityParser[Caveats]
	canIssue              CanIssueFunc[any]
	validateAuthorization RevocationCheckerFunc[any]
	resolveProof          ProofResolverFunc
	parsePrincipal        PrincipalParserFunc
	resolveDIDKey         PrincipalResolverFunc
}

func (vc validationContext[Caveats]) Authority() principal.Verifier {
	return vc.authority
}

func (vc validationContext[Caveats]) Capability() CapabilityParser[Caveats] {
	return vc.capability
}

func (vc validationContext[Caveats]) CanIssue(capability ucan.Capability[any], issuer did.DID) bool {
	return vc.canIssue(capability, issuer)
}

func (vc validationContext[Caveats]) ValidateAuthorization(auth Authorization[any]) result.Result[result.Unit, Revoked] {
	return vc.validateAuthorization(auth)
}

func (vc validationContext[Caveats]) ResolveProof(proof ucan.Link) result.Result[delegation.Delegation, UnavailableProof] {
	return vc.resolveProof(proof)
}

func (vc validationContext[Caveats]) ParsePrincipal(str string) (principal.Verifier, error) {
	return vc.parsePrincipal(str)
}

func (vc validationContext[Caveats]) ResolveDIDKey(did did.DID) result.Result[did.DID, UnresolvedDID] {
	return vc.resolveDIDKey(did)
}

func NewValidationContext[Caveats any](
	authority principal.Verifier,
	capability CapabilityParser[Caveats],
	canIssue CanIssueFunc[any],
	validateAuthorization RevocationCheckerFunc[any],
	resolveProof ProofResolverFunc,
	parsePrincipal PrincipalParserFunc,
	resolveDIDKey PrincipalResolverFunc,
) ValidationContext[Caveats] {
	return validationContext[Caveats]{
		authority,
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
func Access[Caveats any](invocation invocation.Invocation, context ValidationContext[Caveats]) (result.Result[Authorization[Caveats], Unauthorized], error) {
	prf := []delegation.Proof{delegation.FromDelegation(invocation)}
	return Claim(context.Capability(), prf, context)
}

// Claim attempts to find a valid proof chain for the claimed `capability` given
// set of `proofs`. On success an `Authorization` object with detailed proof
// chain is returned and on failure `Unauthorized` error is returned with
// details on paths explored and why they have failed.
func Claim[Caveats any](capability CapabilityParser[Caveats], proofs []delegation.Proof, context ClaimContext) (result.Result[Authorization[Caveats], Unauthorized], error) {
	var sources []Source
	var invalidprf []InvalidProof

	delegations, rerrs := resolveProofs(proofs, context)
	for _, err := range rerrs {
		invalidprf = append(invalidprf, err)
	}

	for _, d := range delegations {
		validation, err := validate(d, delegations, context)
		if err != nil {
			return nil, err
		}

		// Validate each proof if valid add each capability to the list of sources.
		// otherwise collect the error.
		result.MatchResultR0(
			validation,
			func(d delegation.Delegation) {
				for _, c := range d.Capabilities() {
					sources = append(sources, source{c, d})
				}
			},
			func(x InvalidProof) {
				invalidprf = append(invalidprf, x)
			},
		)
	}

	// look for the matching capability
	matches, dlgerrs, unknowns, err := capability.Select(sources)
	if err != nil {
		return nil, err
	}

	var failedprf []InvalidClaim
	for _, matched := range matches {
		selector := matched.Prune(canissuer[Caveats]{canIssue: context.CanIssue})
		if selector == nil {
			authorization := NewAuthorization(matched, nil)
			revoked := result.MatchResultR1(
				context.ValidateAuthorization(authorization),
				func(o result.Unit) Revoked { return nil },
				func(x Revoked) Revoked { return x },
			)
			if revoked == nil {
				return result.Ok[Authorization[Caveats], Unauthorized](authorization), nil
			}
			invalidprf = append(invalidprf, revoked)
		} else {
			authorize(matched, context)
		}
	}

	return result.Error[Authorization[Caveats]](
		NewUnauthorizedError(capability, dlgerrs, unknowns, invalidprf, failedprf),
	), nil
}

// resolveProofs takes `proofs` from the delegation which may contain
// a `Delegation` or a link to one and attempts to resolve links by side loading
// them. It returns a set of resolved `Delegation`s and errors for the proofs
// that could not be resolved.
func resolveProofs(proofs []delegation.Proof, resolver ProofResolver) (dels []delegation.Delegation, errs []UnavailableProof) {
	for _, p := range proofs {
		d, ok := p.Delegation()
		if ok {
			dels = append(dels, d)
		} else {
			result.MatchResultR0(
				resolver.ResolveProof(p.Link()),
				func(d delegation.Delegation) { dels = append(dels, d) },
				func(x UnavailableProof) { errs = append(errs, x) },
			)
		}
	}
	return
}

// Validate a delegation to check it is within the time bound and that it is
// authorized by the issuer.
func validate(dlg delegation.Delegation, prfs []delegation.Delegation, ctx ClaimContext) (result.Result[delegation.Delegation, InvalidProof], error) {
	if ucan.IsExpired(dlg) {
		return result.Error[delegation.Delegation, InvalidProof](NewExpiredError(dlg)), nil
	}
	if ucan.IsTooEarly(dlg) {
		return result.Error[delegation.Delegation, InvalidProof](NewNotValidBeforeError(dlg)), nil
	}
	return verifyAuthorization(dlg, prfs, ctx)
}

// verifyAuthorization verifies that delegation has been authorized by the
// issuer. If issued by the did:key principal checks that the signature is
// valid. If issued by the root authority checks that the signature is valid. If
// issued by the principal identified by other DID method attempts to resolve a
// valid `ucan/attest` attestation from the authority, if attestation is not
// found falls back to resolving did:key for the issuer and verifying its
// signature.
func verifyAuthorization(dlg delegation.Delegation, prfs []delegation.Delegation, ctx ClaimContext) (result.Result[delegation.Delegation, InvalidProof], error) {
	issuer := dlg.Issuer().DID()
	// If the issuer is a did:key we just verify a signature
	if strings.HasPrefix(issuer.String(), "did:key:") {
		vfr, err := ctx.ParsePrincipal(issuer.String())
		if err != nil {
			return nil, err
		}
		sig, err := verifySignature(dlg, vfr)
		if err != nil {
			return nil, err
		}
		return result.MapError(sig, func(err InvalidSignature) InvalidProof {
			return InvalidProof(err)
		}), nil
	}

	// Attempt to resolve embedded authorization session from the authority
	sess, err := verifySession(dlg, prfs, ctx)
	if err != nil {
		return nil, err
	}

	return result.MatchResultR2(
		sess,
		// If we have valid session we consider authorization valid
		func(a Authorization[vdm.AttestationModel]) (result.Result[delegation.Delegation, InvalidProof], error) {
			return result.Ok[delegation.Delegation, InvalidProof](dlg), nil
		},
		func(x Unauthorized) (result.Result[delegation.Delegation, InvalidProof], error) {
			if len(x.FailedProofs()) > 0 {
				return result.Error[delegation.Delegation, InvalidProof](NewSessionEscalationError(dlg, x)), nil
			}

			// Otherwise we try to resolve did:key from the DID instead
			// and use that to verify the signature
			vfr, err := result.MapResultR1(
				ctx.ResolveDIDKey(issuer),
				func(did did.DID) (principal.Verifier, error) {
					return ctx.ParsePrincipal(did.String())
				},
				func(err UnresolvedDID) (UnresolvedDID, error) {
					return err, nil
				},
			)
			if err != nil {
				return nil, err
			}

			return result.MatchResultR2(
				vfr,
				func(v principal.Verifier) (result.Result[delegation.Delegation, InvalidProof], error) {
					sig, err := verifySignature(dlg, v)
					if err != nil {
						return nil, err
					}
					return result.MapError(sig, func(err InvalidSignature) InvalidProof {
						return InvalidProof(err)
					}), nil
				},
				func(x UnresolvedDID) (result.Result[delegation.Delegation, InvalidProof], error) {
					return result.Error[delegation.Delegation, InvalidProof](x), nil
				},
			)
		},
	)
}

func verifySignature(dlg delegation.Delegation, vfr principal.Verifier) (result.Result[delegation.Delegation, InvalidSignature], error) {
	ok, err := ucan.VerifySignature(dlg.Data(), vfr)
	if err != nil {
		return nil, err
	}
	if !ok {
		return result.Error[delegation.Delegation](NewInvalidSignatureError(dlg, vfr)), nil
	}
	return result.Ok[delegation.Delegation, InvalidSignature](dlg), nil
}

// verifySession attempts to find an authorization session - an `ucan/attest`
// capability delegation where `with` matches `config.authority` and `nb.proof`
// matches given delegation.
//
// https://github.com/storacha-network/specs/blob/main/w3-session.md#authorization-session
func verifySession(dlg delegation.Delegation, prfs []delegation.Delegation, ctx ClaimContext) (result.Result[Authorization[vdm.AttestationModel], Unauthorized], error) {
	// Create a schema that will match an authorization for this exact delegation
	attestation := NewCapability(
		"ucan/attest",
		schema.Literal(ctx.Authority().DID().String()),
		schema.Struct[vdm.AttestationModel](
			vdm.AttestationType(),
			policy.Policy{
				policy.Equal(selector.MustParse(".proof"), literal.Link(dlg.Link())),
			},
		),
	)

	// We only consider attestations otherwise we will end up doing an
	// exponential scan if there are other proofs that require attestations.
	var aprfs []delegation.Proof
	for _, p := range prfs {
		if p.Capabilities()[0].Can() == "ucan/attest" {
			aprfs = append(aprfs, delegation.FromDelegation(p))
		}
	}

	return Claim(attestation, aprfs, ctx)
}

// authorize verifies whether any of the delegated proofs grant give capability.
func authorize[Caveats any](match Match[Caveats], context ClaimContext) (result.Result[Authorization[Caveats], InvalidClaim], error) {

}

package validator

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/result/failure"
	"github.com/storacha/go-ucanto/core/schema"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-ucanto/principal/verifier"
	"github.com/storacha/go-ucanto/ucan"
	vdm "github.com/storacha/go-ucanto/validator/datamodel"
	"github.com/ucan-wg/go-ucan/capability/policy"
	"github.com/ucan-wg/go-ucan/capability/policy/literal"
	"github.com/ucan-wg/go-ucan/capability/policy/selector"
)

func IsSelfIssued[Caveats any](capability ucan.Capability[Caveats], issuer did.DID) bool {
	return capability.With() == issuer.DID().String()
}

func ProofUnavailable(ctx context.Context, p ucan.Link) (delegation.Delegation, UnavailableProof) {
	return nil, NewUnavailableProofError(p, fmt.Errorf("no proof resolver configured"))
}

func FailDIDKeyResolution(ctx context.Context, d did.DID) (did.DID, UnresolvedDID) {
	return did.Undef, NewDIDKeyResolutionError(d, fmt.Errorf("no DID resolver configured"))
}

// PrincipalParser provides verifier instances that can validate UCANs issued
// by a given principal.
type PrincipalParser interface {
	ParsePrincipal(str string) (principal.Verifier, error)
}

type PrincipalParserFunc func(str string) (principal.Verifier, error)

// PrincipalResolver is used to resolve a key of the principal that is
// identified by DID different from did:key method. It can be passed into a
// UCAN validator in order to augmented it with additional DID methods support.
type PrincipalResolver interface {
	ResolveDIDKey(ctx context.Context, did did.DID) (did.DID, UnresolvedDID)
}

// PrincipalResolverFunc resolves the key of a principal that is identified by
// DID different from did:key method.
type PrincipalResolverFunc func(ctx context.Context, did did.DID) (did.DID, UnresolvedDID)

// ProofResolver finds a delegations when external proof links are present in
// UCANs. If a resolver is not provided the validator may not be able to explore
// corresponding path within a proof chain.
type ProofResolver interface {
	// Resolve finds a delegation corresponding to an external proof link.
	ResolveProof(ctx context.Context, proof ucan.Link) (delegation.Delegation, UnavailableProof)
}

// Resolve finds a delegation corresponding to an external proof link.
type ProofResolverFunc func(ctx context.Context, proof ucan.Link) (delegation.Delegation, UnavailableProof)

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
	// revoked. It returns `nil` if not revoked.
	ValidateAuthorization(ctx context.Context, auth Authorization[Caveats]) Revoked
}

// RevocationCheckerFunc validates that the passed authorization has not been
// revoked. It returns `nil` if not revoked.
type RevocationCheckerFunc[Caveats any] func(ctx context.Context, auth Authorization[Caveats]) Revoked

// AuthorityProver provides a set of proofs of authority
type AuthorityProver interface {
	AuthorityProofs() []delegation.Delegation
}

// Validator must provide a [principal.Verifier] corresponding to local authority.
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
	AuthorityProver
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
	authorityProofs       []delegation.Delegation
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

func (vc validationContext[Caveats]) ValidateAuthorization(ctx context.Context, auth Authorization[any]) Revoked {
	return vc.validateAuthorization(ctx, auth)
}

func (vc validationContext[Caveats]) ResolveProof(ctx context.Context, proof ucan.Link) (delegation.Delegation, UnavailableProof) {
	return vc.resolveProof(ctx, proof)
}

func (vc validationContext[Caveats]) ParsePrincipal(str string) (principal.Verifier, error) {
	return vc.parsePrincipal(str)
}

func (vc validationContext[Caveats]) ResolveDIDKey(ctx context.Context, did did.DID) (did.DID, UnresolvedDID) {
	return vc.resolveDIDKey(ctx, did)
}

func (vc validationContext[Caveats]) AuthorityProofs() []delegation.Delegation {
	return vc.authorityProofs
}

func NewValidationContext[Caveats any](
	authority principal.Verifier,
	capability CapabilityParser[Caveats],
	canIssue CanIssueFunc[any],
	validateAuthorization RevocationCheckerFunc[any],
	resolveProof ProofResolverFunc,
	parsePrincipal PrincipalParserFunc,
	resolveDIDKey PrincipalResolverFunc,
	authorityProofs ...delegation.Delegation,
) ValidationContext[Caveats] {
	return validationContext[Caveats]{
		authority,
		capability,
		canIssue,
		validateAuthorization,
		resolveProof,
		parsePrincipal,
		resolveDIDKey,
		authorityProofs,
	}
}

// Access finds a valid path in a proof chain of the given
// [invocation.Invocation] by exploring every possible option. On success an
// [Authorization] object is returned that illustrates the valid path. If no
// valid path is found [Unauthorized] error is returned detailing all explored
// paths and where they proved to fail.
func Access[Caveats any](ctx context.Context, invocation invocation.Invocation, vctx ValidationContext[Caveats]) (Authorization[Caveats], Unauthorized) {
	prf := []delegation.Proof{delegation.FromDelegation(invocation)}
	return Claim(ctx, vctx.Capability(), prf, vctx)
}

// Claim attempts to find a valid proof chain for the claimed [CapabilityParser]
// given set of `proofs`. On success an [Authorization] object with detailed
// proof chain is returned and on failure [Unauthorized] error is returned with
// details on paths explored and why they have failed.
func Claim[Caveats any](ctx context.Context, capability CapabilityParser[Caveats], proofs []delegation.Proof, cctx ClaimContext) (Authorization[Caveats], Unauthorized) {
	var sources []Source
	var invalidprf []InvalidProof

	delegations, rerrs := ResolveProofs(ctx, proofs, cctx)
	for _, err := range rerrs {
		invalidprf = append(invalidprf, err)
	}

	for _, prf := range delegations {
		// Validate each proof if valid add each capability to the list of sources
		// or collect the error.
		validation, err := Validate(ctx, prf, delegations, cctx)
		if err != nil {
			invalidprf = append(invalidprf, err)
			continue
		}

		for _, c := range validation.Capabilities() {
			sources = append(sources, NewSource(c, prf))
		}
	}

	// look for the matching capability
	matches, dlgerrs, unknowns := capability.Select(sources)

	var failedprf []InvalidClaim
	for _, matched := range matches {
		selector := matched.Prune(canissuer[Caveats]{canIssue: cctx.CanIssue})
		if selector == nil {
			auth := NewAuthorization(matched, nil)
			revoked := cctx.ValidateAuthorization(ctx, ConvertUnknownAuthorization(auth))
			if revoked != nil {
				invalidprf = append(invalidprf, revoked)
				continue
			}
			return auth, nil
		}

		a, err := Authorize(ctx, matched, cctx)
		if err != nil {
			failedprf = append(failedprf, err)
			continue
		}

		auth := NewAuthorization(matched, []Authorization[Caveats]{a})
		revoked := cctx.ValidateAuthorization(ctx, ConvertUnknownAuthorization(auth))
		if revoked != nil {
			invalidprf = append(invalidprf, revoked)
			continue
		}

		return auth, nil
	}

	return nil, NewUnauthorizedError(capability, dlgerrs, unknowns, invalidprf, failedprf)
}

// ResolveProofs takes `proofs` from the delegation which may contain
// a [delegation.Delegation] or a link to one and attempts to resolve links by
// side loading them. It returns a set of resolved [delegation.Delegation]s and
// errors for the proofs that could not be resolved.
func ResolveProofs(ctx context.Context, proofs []delegation.Proof, resolver ProofResolver) (dels []delegation.Delegation, errs []UnavailableProof) {
	for _, p := range proofs {
		d, ok := p.Delegation()
		if ok {
			dels = append(dels, d)
		} else {
			d, err := resolver.ResolveProof(ctx, p.Link())
			if err != nil {
				errs = append(errs, err)
				continue
			}
			dels = append(dels, d)
		}
	}
	return
}

// Validate a delegation to check it is within the time bound and that it is
// authorized by the issuer.
func Validate(ctx context.Context, dlg delegation.Delegation, prfs []delegation.Delegation, cctx ClaimContext) (delegation.Delegation, InvalidProof) {
	if ucan.IsExpired(dlg) {
		return nil, NewExpiredError(dlg)
	}
	if ucan.IsTooEarly(dlg) {
		return nil, NewNotValidBeforeError(dlg)
	}
	return VerifyAuthorization(ctx, dlg, prfs, cctx)
}

// VerifyAuthorization verifies that delegation has been authorized by the
// issuer. If issued by the did:key principal checks that the signature is
// valid. If issued by the root authority checks that the signature is valid. If
// issued by the principal identified by other DID method attempts to resolve a
// valid `ucan/attest` attestation from the authority, if attestation is not
// found falls back to resolving did:key for the issuer and verifying its
// signature.
func VerifyAuthorization(ctx context.Context, dlg delegation.Delegation, prfs []delegation.Delegation, cctx ClaimContext) (delegation.Delegation, InvalidProof) {
	issuer := dlg.Issuer().DID()
	// If the issuer is a did:key we just verify a signature
	if strings.HasPrefix(issuer.String(), "did:key:") {
		vfr, err := cctx.ParsePrincipal(issuer.String())
		if err != nil {
			return nil, NewUnverifiableSignatureError(dlg, err)
		}
		return VerifySignature(dlg, vfr)
	}

	if dlg.Issuer().DID() == cctx.Authority().DID() {
		return VerifySignature(dlg, cctx.Authority())
	}

	// If issuer is not a did:key principal nor configured authority, we
	// attempt to resolve embedded authorization session from the authority
	_, err := VerifySession(ctx, dlg, prfs, cctx)
	if err != nil {
		if len(err.FailedProofs()) > 0 {
			return nil, NewSessionEscalationError(dlg, err)
		}

		// Otherwise we try to resolve did:key from the DID instead
		// and use that to verify the signature
		did, err := cctx.ResolveDIDKey(ctx, issuer)
		if err != nil {
			return nil, err
		}

		vfr, perr := cctx.ParsePrincipal(did.String())
		if perr != nil {
			return nil, NewUnverifiableSignatureError(dlg, perr)
		}

		wvfr, werr := verifier.Wrap(vfr, issuer)
		if werr != nil {
			return nil, NewUnverifiableSignatureError(dlg, perr)
		}

		return VerifySignature(dlg, wvfr)
	}

	return dlg, nil
}

// VerifySignature verifies the delegation was signed by the passed verifier.
func VerifySignature(dlg delegation.Delegation, vfr principal.Verifier) (delegation.Delegation, BadSignature) {
	ok, err := ucan.VerifySignature(dlg.Data(), vfr)
	if err != nil {
		return nil, NewUnverifiableSignatureError(dlg, err)
	}
	if !ok {
		return nil, NewInvalidSignatureError(dlg, vfr)
	}
	return dlg, nil
}

// VerifySession attempts to find an authorization session - an `ucan/attest`
// capability delegation where `with` matches `ctx.Authority()` and `nb.proof`
// matches given delegation.
//
// https://github.com/storacha-network/specs/blob/main/w3-session.md#authorization-session
func VerifySession(ctx context.Context, dlg delegation.Delegation, prfs []delegation.Delegation, cctx ClaimContext) (Authorization[vdm.AttestationModel], Unauthorized) {
	// Recognize attestations from all authorized principals, not just authority
	var withSchemas []schema.Reader[string, string]
	for _, p := range cctx.AuthorityProofs() {
		if p.Capabilities()[0].Can() == "ucan/attest" && p.Capabilities()[0].With() == cctx.Authority().DID().String() {
			withSchemas = append(withSchemas, schema.Literal(p.Audience().DID().String()))
		}
	}

	withSchema := schema.Literal(cctx.Authority().DID().String())
	if len(withSchemas) > 0 {
		withSchemas = append(withSchemas, schema.Literal(cctx.Authority().DID().String()))
		withSchema = schema.Or(withSchemas...)
	}

	// Create a schema that will match an authorization for this exact delegation
	attestation := NewCapability(
		"ucan/attest",
		withSchema,
		schema.Struct[vdm.AttestationModel](
			vdm.AttestationType(),
			policy.Policy{
				policy.Equal(selector.MustParse(".proof"), literal.Link(dlg.Link())),
			},
		),
		func(claimed, delegated ucan.Capability[vdm.AttestationModel]) failure.Failure {
			err := DefaultDerives(claimed, delegated)
			if err != nil {
				return err
			}
			if claimed.Nb().Proof != delegated.Nb().Proof {
				return schema.NewSchemaError(fmt.Sprintf(`proof: %s violates %s`, claimed.Nb().Proof, delegated.Nb().Proof))
			}
			return nil
		},
	)

	// We only consider attestations otherwise we will end up doing an
	// exponential scan if there are other proofs that require attestations.
	// Also filter any proofs that _are_ the delegation we're verifying so
	// we don't recurse indefinitely.
	var aprfs []delegation.Proof
	for _, p := range prfs {
		if p.Link().String() == dlg.Link().String() {
			continue
		}

		if p.Capabilities()[0].Can() == "ucan/attest" {
			aprfs = append(aprfs, delegation.FromDelegation(p))
		}
	}

	return Claim(ctx, attestation, aprfs, cctx)
}

// Authorize verifies whether any of the delegated proofs grant capability.
func Authorize[Caveats any](ctx context.Context, match Match[Caveats], cctx ClaimContext) (Authorization[Caveats], InvalidClaim) {
	// load proofs from all delegations
	sources, invalidprf := ResolveMatch(ctx, match, cctx)

	matches, dlgerrs, unknowns := match.Select(sources)

	var failedprf []InvalidClaim
	for _, matched := range matches {
		selector := matched.Prune(canissuer[Caveats]{canIssue: cctx.CanIssue})
		if selector == nil {
			return NewAuthorization(matched, nil), nil
		}

		auth, err := Authorize(ctx, selector, cctx)
		if err != nil {
			failedprf = append(failedprf, err)
			continue
		}

		return NewAuthorization(matched, []Authorization[Caveats]{auth}), nil
	}

	return nil, NewInvalidClaimError(match, dlgerrs, unknowns, invalidprf, failedprf)
}

func ResolveMatch[Caveats any](ctx context.Context, match Match[Caveats], context ClaimContext) (sources []Source, errors []ProofError) {
	includes := map[string]struct{}{}
	var wg sync.WaitGroup
	var lock sync.RWMutex
	for _, source := range match.Source() {
		id := source.Delegation().Link().String()
		if _, ok := includes[id]; !ok {
			includes[id] = struct{}{}
			wg.Add(1)
			go func(s Source) {
				srcs, errs := ResolveSources(ctx, s, context)
				lock.Lock()
				defer lock.Unlock()
				defer wg.Done()
				sources = append(sources, srcs...)
				errors = append(errors, errs...)
			}(source)
		}
	}
	wg.Wait()
	return
}

func ResolveSources(ctx context.Context, source Source, cctx ClaimContext) (sources []Source, errors []ProofError) {
	dlg := source.Delegation()
	var prfs []delegation.Delegation

	br, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(dlg.Blocks()))
	if err != nil {
		errors = append(errors, NewProofError(dlg.Link(), err))
		return
	}

	dlgs, failedprf := ResolveProofs(
		ctx,
		delegation.NewProofsView(dlg.Proofs(), br),
		cctx,
	)

	// All the proofs that failed to resolve are saved as proof errors.
	for _, err := range failedprf {
		errors = append(errors, NewProofError(err.Link(), err))
	}

	// All the proofs that resolved are checked for principal alignment. Ones that
	// do not align are saved as proof errors.
	for _, prf := range dlgs {
		// If proof does not delegate to a matching audience save an proof error.
		if dlg.Issuer().DID() != prf.Audience().DID() {
			errors = append(errors, NewProofError(prf.Link(), NewPrincipalAlignmentError(dlg.Issuer(), prf)))
		} else {
			prfs = append(prfs, prf)
		}
	}
	// In the second pass we attempt to proofs that were resolved and are aligned.
	for _, prf := range prfs {
		_, err := Validate(ctx, prf, prfs, cctx)

		// If proof is not valid (expired, not active yet or has incorrect
		// signature) save a corresponding proof error.
		if err != nil {
			errors = append(errors, NewProofError(prf.Link(), err))
			continue
		}

		// Otherwise create source objects for it's capabilities, so we could
		// track which proof in which capability the are from.
		for _, cap := range prf.Capabilities() {
			sources = append(sources, NewSource(cap, prf))
		}
	}
	return
}

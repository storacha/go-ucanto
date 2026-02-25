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

func NotExpiredNotTooEarly(dlg delegation.Delegation) InvalidProof {
	if ucan.IsExpired(dlg) {
		return NewExpiredError(dlg)
	}
	if ucan.IsTooEarly(dlg) {
		return NewNotValidBeforeError(dlg)
	}

	return nil
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

type TimeBoundsValidator interface {
	ValidateTimeBounds(dlg delegation.Delegation) InvalidProof
}

type TimeBoundsValidatorFunc func(dlg delegation.Delegation) InvalidProof

type ClaimContext interface {
	Validator
	RevocationChecker[any]
	CanIssuer[any]
	ProofResolver
	PrincipalParser
	PrincipalResolver
	AuthorityProver
	TimeBoundsValidator
}

type claimContext struct {
	authority             principal.Verifier
	canIssue              CanIssueFunc[any]
	validateAuthorization RevocationCheckerFunc[any]
	resolveProof          ProofResolverFunc
	parsePrincipal        PrincipalParserFunc
	resolveDIDKey         PrincipalResolverFunc
	validateTimeBounds    TimeBoundsValidatorFunc
	authorityProofs       []delegation.Delegation
}

func NewClaimContext(
	authority principal.Verifier,
	canIssue CanIssueFunc[any],
	validateAuthorization RevocationCheckerFunc[any],
	resolveProof ProofResolverFunc,
	parsePrincipal PrincipalParserFunc,
	resolveDIDKey PrincipalResolverFunc,
	validateTimeBounds TimeBoundsValidatorFunc,
	authorityProofs ...delegation.Delegation,
) ClaimContext {
	return claimContext{
		authority,
		canIssue,
		validateAuthorization,
		resolveProof,
		parsePrincipal,
		resolveDIDKey,
		validateTimeBounds,
		authorityProofs,
	}
}

func (cc claimContext) Authority() principal.Verifier {
	return cc.authority
}

func (cc claimContext) CanIssue(capability ucan.Capability[any], issuer did.DID) bool {
	return cc.canIssue(capability, issuer)
}

func (cc claimContext) ValidateAuthorization(ctx context.Context, auth Authorization[any]) Revoked {
	return cc.validateAuthorization(ctx, auth)
}

func (cc claimContext) ResolveProof(ctx context.Context, proof ucan.Link) (delegation.Delegation, UnavailableProof) {
	return cc.resolveProof(ctx, proof)
}

func (cc claimContext) ParsePrincipal(str string) (principal.Verifier, error) {
	return cc.parsePrincipal(str)
}

func (cc claimContext) ResolveDIDKey(ctx context.Context, did did.DID) (did.DID, UnresolvedDID) {
	return cc.resolveDIDKey(ctx, did)
}

func (cc claimContext) ValidateTimeBounds(dlg delegation.Delegation) InvalidProof {
	return cc.validateTimeBounds(dlg)
}

func (cc claimContext) AuthorityProofs() []delegation.Delegation {
	return cc.authorityProofs
}

type ValidationContext[Caveats any] interface {
	ClaimContext
	Capability() CapabilityParser[Caveats]
}

type validationContext[Caveats any] struct {
	claimContext
	capability CapabilityParser[Caveats]
}

func NewValidationContext[Caveats any](
	authority principal.Verifier,
	capability CapabilityParser[Caveats],
	canIssue CanIssueFunc[any],
	validateAuthorization RevocationCheckerFunc[any],
	resolveProof ProofResolverFunc,
	parsePrincipal PrincipalParserFunc,
	resolveDIDKey PrincipalResolverFunc,
	validateTimeBounds TimeBoundsValidatorFunc,
	authorityProofs ...delegation.Delegation,
) ValidationContext[Caveats] {
	return validationContext[Caveats]{
		claimContext{
			authority,
			canIssue,
			validateAuthorization,
			resolveProof,
			parsePrincipal,
			resolveDIDKey,
			validateTimeBounds,
			authorityProofs,
		},
		capability,
	}
}

func (vc validationContext[Caveats]) Capability() CapabilityParser[Caveats] {
	return vc.capability
}

// PruneProofs finds the minimal set of supporting proofs for dlg from the
// proofs already embedded in dlg, and returns them without dlg itself.
//
// It is the client-side counterpart of [Access]: where Access validates an
// invocation arriving at a server, PruneProofs selects the proof subset for a
// delegation being built by a client to send over a size-constrained channel
// (e.g. an HTTP header).
//
// dlg must be built with the full candidate proof pool — all proof blocks
// present in its blockstore and all proof CIDs listed in its prf field.
// PruneProofs walks the delegation chain using the same logic as [Claim] and
// returns only the delegations actually needed to form a valid chain.
//
// The [ValidationContext] Authority must be the verifier of the service that
// issued the ucan/attest delegations in the pool (e.g. the upload-service in
// the storacha network). This is what allows non-did:key issuers (such as
// did:mailto accounts) to be authorized through attestation, without requiring
// the client to know the specific trust configuration of the server that will
// ultimately receive the delegation.
//
// Because the proof set is discovered from a root delegation, usage requires
// two steps:
//
//  1. Build a draft delegation with all candidate proofs attached.
//  2. Call PruneProofs to discover the minimal subset.
//  3. Build the final delegation with only the needed proofs.
//
// Note: the capability in vctx must match the capability in dlg, or
// PruneProofs will return [Unauthorized]. If dlg contains multiple
// capabilities, only the chain for the first matching one is discovered.
func PruneProofs[Caveats any](ctx context.Context, dlg delegation.Delegation, vctx ValidationContext[Caveats]) ([]delegation.Proof, Unauthorized) {
	all, unauth := SelectProofs(ctx, vctx.Capability(), []delegation.Proof{delegation.FromDelegation(dlg)}, vctx)
	if unauth != nil {
		return nil, unauth
	}
	dlgCID := dlg.Link().String()
	var result []delegation.Proof
	for _, d := range all {
		if d.Link().String() == dlgCID {
			continue
		}

		// Re-export the delegation into a fresh blockstore so it only carries
		// the blocks from its own proof chain, not the full candidate pool that
		// was inherited from the draft's blockstore.
		bs, err := blockstore.NewBlockStore(blockstore.WithBlocksIterator(d.Export()))
		if err != nil {
			return nil, NewUnauthorizedError(vctx.Capability(), nil, nil, []InvalidProof{NewUnavailableProofError(d.Link(), err)}, nil)
		}

		exported, err := delegation.NewDelegation(d.Root(), bs)
		if err != nil {
			return nil, NewUnauthorizedError(vctx.Capability(), nil, nil, []InvalidProof{NewUnavailableProofError(d.Link(), err)}, nil)
		}

		result = append(result, delegation.FromDelegation(exported))
	}

	return result, nil
}

// SelectProofs attempts to find a valid proof chain for the claimed capability
// given the set of proofs, and returns the minimal set of [delegation.Delegation]s
// needed to prove the claim. This includes any ucan/attest delegations found
// during validation (e.g. for did:mailto issuers). Returns [Unauthorized] if
// no valid chain exists.
func SelectProofs[C any](ctx context.Context, capability CapabilityParser[C], proofs []delegation.Proof, cctx ClaimContext) ([]delegation.Delegation, Unauthorized) {
	auth, err := Claim(ctx, capability, proofs, cctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var result []delegation.Delegation
	var walk func(Authorization[any])
	walk = func(a Authorization[any]) {
		cid := a.Delegation().Link().String()
		if _, ok := seen[cid]; !ok {
			seen[cid] = struct{}{}
			result = append(result, a.Delegation())
		}
		for _, attest := range a.Attestations() {
			walk(attest)
		}
		for _, prf := range a.Proofs() {
			walk(prf)
		}
	}
	walk(ConvertUnknownAuthorization(auth))
	return result, nil
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

	var claimAttestations []Authorization[any]
	for _, prf := range delegations {
		// Validate each proof if valid add each capability to the list of sources
		// or collect the error.
		validation, attest, err := Validate(ctx, prf, delegations, cctx)
		if err != nil {
			invalidprf = append(invalidprf, err)
			continue
		}
		if attest != nil {
			claimAttestations = append(claimAttestations, attest)
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
			auth := NewAuthorization(matched, nil, claimAttestations)
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

		auth := NewAuthorization(matched, []Authorization[Caveats]{a}, claimAttestations)
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
// authorized by the issuer. The second return value is a ucan/attest
// Authorization if one was used to authorize a non-did:key issuer, or nil
// otherwise.
func Validate(ctx context.Context, dlg delegation.Delegation, prfs []delegation.Delegation, cctx ClaimContext) (delegation.Delegation, Authorization[any], InvalidProof) {
	if invalid := cctx.ValidateTimeBounds(dlg); invalid != nil {
		return nil, nil, invalid
	}

	return VerifyAuthorization(ctx, dlg, prfs, cctx)
}

// VerifyAuthorization verifies that delegation has been authorized by the
// issuer. If issued by the did:key principal checks that the signature is
// valid. If issued by the root authority checks that the signature is valid. If
// issued by the principal identified by other DID method attempts to resolve a
// valid `ucan/attest` attestation from the authority, if attestation is not
// found falls back to resolving did:key for the issuer and verifying its
// signature. The second return value is the ucan/attest Authorization when one
// was used, or nil otherwise.
func VerifyAuthorization(ctx context.Context, dlg delegation.Delegation, prfs []delegation.Delegation, cctx ClaimContext) (delegation.Delegation, Authorization[any], InvalidProof) {
	issuer := dlg.Issuer().DID()
	// If the issuer is a did:key we just verify a signature
	if strings.HasPrefix(issuer.String(), "did:key:") {
		vfr, err := cctx.ParsePrincipal(issuer.String())
		if err != nil {
			return nil, nil, NewUnverifiableSignatureError(dlg, err)
		}
		dlg, invalid := VerifySignature(dlg, vfr)
		return dlg, nil, invalid
	}

	if dlg.Issuer().DID() == cctx.Authority().DID() {
		dlg, invalid := VerifySignature(dlg, cctx.Authority())
		return dlg, nil, invalid
	}

	// If issuer is not a did:key principal nor configured authority, we
	// attempt to resolve embedded authorization session from the authority
	attest, err := VerifySession(ctx, dlg, prfs, cctx)
	if err != nil {
		if len(err.FailedProofs()) > 0 {
			return nil, nil, NewSessionEscalationError(dlg, err)
		}

		// Otherwise we try to resolve did:key from the DID instead
		// and use that to verify the signature
		did, err := cctx.ResolveDIDKey(ctx, issuer)
		if err != nil {
			return nil, nil, err
		}

		vfr, perr := cctx.ParsePrincipal(did.String())
		if perr != nil {
			return nil, nil, NewUnverifiableSignatureError(dlg, perr)
		}

		wvfr, werr := verifier.Wrap(vfr, issuer)
		if werr != nil {
			return nil, nil, NewUnverifiableSignatureError(dlg, perr)
		}

		dlg, invalid := VerifySignature(dlg, wvfr)
		return dlg, nil, invalid
	}

	return dlg, ConvertUnknownAuthorization(attest), nil
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
	sources, attestations, invalidprf := ResolveMatch(ctx, match, cctx)

	matches, dlgerrs, unknowns := match.Select(sources)

	var failedprf []InvalidClaim
	for _, matched := range matches {
		relevantAttestations := attestationsFor(matched.Source()[0].Delegation().Link(), attestations)

		selector := matched.Prune(canissuer[Caveats]{canIssue: cctx.CanIssue})
		if selector == nil {
			return NewAuthorization(matched, nil, relevantAttestations), nil
		}

		auth, err := Authorize(ctx, selector, cctx)
		if err != nil {
			failedprf = append(failedprf, err)
			continue
		}

		return NewAuthorization(matched, []Authorization[Caveats]{auth}, relevantAttestations), nil
	}

	return nil, NewInvalidClaimError(match, dlgerrs, unknowns, invalidprf, failedprf)
}

// attestationsFor returns the attestations that attest to the given delegation
// link, discarding any that belong to other delegations in the pool.
func attestationsFor(link ucan.Link, attestations []Authorization[any]) []Authorization[any] {
	var result []Authorization[any]
	for _, attest := range attestations {
		if model, ok := attest.Capability().Nb().(vdm.AttestationModel); ok {
			if model.Proof.String() == link.String() {
				result = append(result, attest)
			}
		}
	}
	return result
}

func ResolveMatch[Caveats any](ctx context.Context, match Match[Caveats], context ClaimContext) (sources []Source, attestations []Authorization[any], errors []ProofError) {
	includes := map[string]struct{}{}
	var wg sync.WaitGroup
	var lock sync.RWMutex
	for _, source := range match.Source() {
		id := source.Delegation().Link().String()
		if _, ok := includes[id]; !ok {
			includes[id] = struct{}{}
			wg.Add(1)
			go func(s Source) {
				srcs, attests, errs := ResolveSources(ctx, s, context)
				lock.Lock()
				defer lock.Unlock()
				defer wg.Done()
				sources = append(sources, srcs...)
				attestations = append(attestations, attests...)
				errors = append(errors, errs...)
			}(source)
		}
	}
	wg.Wait()
	return
}

func ResolveSources(ctx context.Context, source Source, cctx ClaimContext) (sources []Source, attestations []Authorization[any], errors []ProofError) {
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
		_, attest, err := Validate(ctx, prf, prfs, cctx)

		// If proof is not valid (expired, not active yet or has incorrect
		// signature) save a corresponding proof error.
		if err != nil {
			errors = append(errors, NewProofError(prf.Link(), err))
			continue
		}
		if attest != nil {
			attestations = append(attestations, attest)
		}

		// Otherwise create source objects for it's capabilities, so we could
		// track which proof in which capability the are from.
		for _, cap := range prf.Capabilities() {
			sources = append(sources, NewSource(cap, prf))
		}
	}
	return
}

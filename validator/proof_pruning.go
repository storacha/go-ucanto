package validator

import (
	"context"
	"fmt"

	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/principal"
	edverifier "github.com/storacha/go-ucanto/principal/ed25519/verifier"
	"github.com/storacha/go-ucanto/ucan"
)

// NewProofPruner returns a [delegation.ProofPruner] that builds a draft
// delegation from the provided parameters, then selects the minimal subset of
// proofs needed to authorize cap.
//
// cap must match the capability being delegated — it is used to walk the proof
// chain and determine which proofs are actually required. Only the chain for
// cap is discovered.
//
// attestor must be the verifier of the service that issued the ucan/attest
// delegations in the proof pool (e.g. the upload-service in the storacha
// network).
func NewProofPruner[Caveats any](attestor principal.Verifier, cap CapabilityParser[Caveats]) delegation.ProofPruner {
	return func(issuer ucan.Signer, audience ucan.Principal, capabilities []ucan.Capability[ucan.CaveatBuilder], options ...delegation.Option) (delegation.Proofs, error) {
		draft, err := delegation.Delegate(issuer, audience, capabilities, options...)
		if err != nil {
			return nil, fmt.Errorf("building draft delegation: %w", err)
		}

		if len(draft.Proofs()) == 0 {
			return nil, nil
		}

		pruningCtx := NewValidationContext(
			attestor,
			cap,
			IsSelfIssued,
			func(context.Context, Authorization[any]) Revoked {
				return nil
			},
			ProofUnavailable,
			edverifier.Parse,
			FailDIDKeyResolution,
			NotExpiredNotTooEarly,
		)

		prunedPfs, unauth := pruneProofs(context.Background(), draft, pruningCtx)
		if unauth != nil {
			return nil, fmt.Errorf("pruning proofs: %w", unauth)
		}

		return prunedPfs, nil
	}
}

// PruneProofs selects the minimal subset of proofs from dlg's embedded proof
// pool that form a valid chain for dlg's issuer. The typical use case is
// building a delegation from scratch: assemble a draft with all candidate
// proofs, call PruneProofs to discover which are actually needed, then build
// the final delegation with only those proofs. This is useful to build delegations
// that are optimized for size, but can also be used to confirm the proof chain
// is valid before sending it over to the server.
//
// dlg serves as both the subject being optimized and the principal context for
// chain validation. [Claim] enforces principal alignment at each level of the
// chain by checking that each proof's audience matches the issuer of the
// delegation it proves. Passing a flat pool of proofs directly to [SelectProofs]
// would break this: any self-issued or authority-issued delegation in the pool
// would appear valid on its own, without establishing a chain back to dlg's
// issuer.
//
// The [ValidationContext] Authority must be the verifier of the service that
// issued the ucan/attest delegations in the pool (e.g. the upload-service in
// the storacha network). This is what allows non-did:key issuers (such as
// did:mailto accounts) to be authorized through attestation, without requiring
// the client to know the specific trust configuration of the server that will
// ultimately receive the delegation.
//
// Note: the capability in vctx must match the capability in dlg, or
// PruneProofs will return [Unauthorized]. If dlg contains multiple
// capabilities, only the chain for the first matching one is discovered.
func pruneProofs[Caveats any](ctx context.Context, dlg delegation.Delegation, vctx ValidationContext[Caveats]) ([]delegation.Proof, Unauthorized) {
	all, unauth := selectProofs(ctx, vctx.Capability(), []delegation.Proof{delegation.FromDelegation(dlg)}, vctx)
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
func selectProofs[C any](ctx context.Context, capability CapabilityParser[C], proofs []delegation.Proof, cctx ClaimContext) ([]delegation.Delegation, Unauthorized) {
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
		for _, p := range a.Proofs() {
			walk(p)
		}
		for _, attest := range a.Attestations() {
			walk(attest)
		}
	}
	walk(ConvertUnknownAuthorization(auth))
	return result, nil
}

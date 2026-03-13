package validator

import (
	"context"
	"fmt"

	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/principal"
	edverifier "github.com/storacha/go-ucanto/principal/ed25519/verifier"
)

// NewProofChainOptimizer returns a [delegation.ProofChainOptimizer] that uses
// proof chain validation to select the minimal subset of proofs needed to
// authorize cap in the delegation.
//
// attestor must be the verifier of the service that issued the ucan/attest
// delegations in the proof pool (e.g. the upload-service in the storacha
// network).
//
// cap must match the capability being delegated — it is used to walk the proof
// chain and determine which proofs are actually required. Only the chain for
// cap is discovered.
func NewProofChainOptimizer(attestor principal.Verifier, cap CapabilityParser[any]) delegation.ProofChainOptimizer {
	return func(draft delegation.Delegation) (delegation.Proofs, error) {
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

		prunedPfs, unauth := PruneProofs(context.Background(), draft, pruningCtx)
		if unauth != nil {
			return nil, fmt.Errorf("pruning proofs: %w", unauth)
		}

		return prunedPfs, nil
	}
}

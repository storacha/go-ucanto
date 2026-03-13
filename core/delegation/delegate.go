package delegation

import (
	"fmt"

	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/ipld/block"
	"github.com/storacha/go-ucanto/core/ipld/codec/cbor"
	"github.com/storacha/go-ucanto/core/ipld/hash/sha256"
	"github.com/storacha/go-ucanto/ucan"
	udm "github.com/storacha/go-ucanto/ucan/datamodel/ucan"
)

// Option is an option configuring a UCAN delegation.
type Option func(cfg *delegationConfig) error

type delegationConfig struct {
	exp       *int
	noexp     bool
	nbf       int
	nnc       string
	fct       []ucan.FactBuilder
	prf       Proofs
	optimizer ProofChainOptimizer
}

// WithExpiration configures the expiration time in UTC seconds since Unix
// epoch.
func WithExpiration(exp int) Option {
	return func(cfg *delegationConfig) error {
		cfg.exp = &exp
		cfg.noexp = false
		return nil
	}
}

// WithNoExpiration configures the UCAN to never expire.
//
// WARNING: this will cause the delegation to be valid FOREVER, unless revoked.
func WithNoExpiration() Option {
	return func(cfg *delegationConfig) error {
		cfg.exp = nil
		cfg.noexp = true
		return nil
	}
}

// WithNotBefore configures the time in UTC seconds since Unix epoch when the
// UCAN will become valid.
func WithNotBefore(nbf int) Option {
	return func(cfg *delegationConfig) error {
		cfg.nbf = nbf
		return nil
	}
}

// WithNonce configures the nonce value for the UCAN.
func WithNonce(nnc string) Option {
	return func(cfg *delegationConfig) error {
		cfg.nnc = nnc
		return nil
	}
}

// WithFacts configures the facts for the UCAN.
func WithFacts(fct []ucan.FactBuilder) Option {
	return func(cfg *delegationConfig) error {
		cfg.fct = fct
		return nil
	}
}

// WithProof configures the proof(s) for the UCAN. If the `issuer` of this
// `Delegation` is not the resource owner / service provider, for the delegated
// capabilities, the `proofs` must contain valid `Proof`s containing
// delegations to the `issuer`.
func WithProof(prf ...Proof) Option {
	return func(cfg *delegationConfig) error {
		cfg.prf = prf
		return nil
	}
}

// ProofChainOptimizer selects the minimal subset of proofs that form a valid
// chain from a delegation's full proof pool. It receives the base delegation
// and returns only the proofs required to authorize it.
//
// Use [validator.NewProofChainOptimizer] to create one.
type ProofChainOptimizer func(base Delegation) (Proofs, error)

// WithOptimizedProofChain configures proof chain optimization. The optimizer
// selects the minimal subset of proofs that form a valid chain. If it's not
// possible to build a valid proof chain, an error will be returned.
// Delegations with optimized proof chains don't include unnecessary proofs,
// which makes them suitable for size-constrained channels, such as HTTP headers.
//
// Use [validator.NewProofChainOptimizer] to create an optimizer.
func WithOptimizedProofChain(optimizer ProofChainOptimizer) Option {
	return func(cfg *delegationConfig) error {
		cfg.optimizer = optimizer
		return nil
	}
}

// Delegate creates a new signed token with a given `options.issuer`. If
// expiration is not set it defaults to 30 seconds from now. Returns UCAN in
// primary IPLD representation.
func Delegate[C ucan.CaveatBuilder](issuer ucan.Signer, audience ucan.Principal, capabilities []ucan.Capability[C], options ...Option) (Delegation, error) {
	cfg := delegationConfig{}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	bs, err := blockstore.NewBlockStore()
	if err != nil {
		return nil, err
	}

	links, err := cfg.prf.WriteInto(bs)
	if err != nil {
		return nil, err
	}

	opts := []ucan.Option{
		ucan.WithFacts(cfg.fct),
		ucan.WithNonce(cfg.nnc),
		ucan.WithNotBefore(cfg.nbf),
		ucan.WithProof(links...),
	}
	if cfg.noexp {
		opts = append(opts, ucan.WithNoExpiration())
	}
	if cfg.exp != nil {
		opts = append(opts, ucan.WithExpiration(*cfg.exp))
	}

	data, err := ucan.Issue(issuer, audience, capabilities, opts...)
	if err != nil {
		return nil, fmt.Errorf("issuing UCAN: %w", err)
	}

	rt, err := block.Encode(data.Model(), udm.Type(), cbor.Codec, sha256.Hasher)
	if err != nil {
		return nil, fmt.Errorf("encoding UCAN: %w", err)
	}

	err = bs.Put(rt)
	if err != nil {
		return nil, fmt.Errorf("adding delegation root to store: %w", err)
	}

	del, err := NewDelegation(rt, bs)
	if err != nil {
		return nil, fmt.Errorf("creating delegation: %w", err)
	}

	if cfg.optimizer != nil {
		prunedPfs, err := cfg.optimizer(del)
		if err != nil {
			return nil, fmt.Errorf("optimizing proof chain: %w", err)
		}
		// Rebuild with only the pruned proofs. cfg.optimizer is intentionally
		// omitted to avoid recursion.
		var rebuildOpts []Option
		if cfg.noexp {
			rebuildOpts = append(rebuildOpts, WithNoExpiration())
		} else if cfg.exp != nil {
			rebuildOpts = append(rebuildOpts, WithExpiration(*cfg.exp))
		}
		if cfg.nbf != 0 {
			rebuildOpts = append(rebuildOpts, WithNotBefore(cfg.nbf))
		}
		if cfg.nnc != "" {
			rebuildOpts = append(rebuildOpts, WithNonce(cfg.nnc))
		}
		if len(cfg.fct) > 0 {
			rebuildOpts = append(rebuildOpts, WithFacts(cfg.fct))
		}
		rebuildOpts = append(rebuildOpts, WithProof(prunedPfs...))
		return Delegate(issuer, audience, capabilities, rebuildOpts...)
	}

	return del, nil
}

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
	exp    *int
	noexp  bool
	nbf    int
	nnc    string
	fct    []ucan.FactBuilder
	prf    Proofs
	pruner ProofPruner
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

// ProofPruner selects the minimal subset of proofs that form a valid chain
// from a candidate proof pool. It has the same signature as [Delegate] but
// returns only the proofs required instead of the final delegation.
//
// Use [validator.NewProofPruner] to create one.
type ProofPruner func(issuer ucan.Signer, audience ucan.Principal, capabilities []ucan.Capability[ucan.CaveatBuilder], options ...Option) (Proofs, error)

// WithProofPruning configures proof pruning. The pruner selects the minimal
// subset of proofs that form a valid chain, and the delegation is rebuilt with
// only those proofs. If it's not possible to build a valid proof chain, an
// error will be returned.
// Delegations with pruned proofs don't include unnecessary proofs, which makes
// them suitable for size-constrained channels, such as HTTP headers.
//
// Use [validator.NewProofPruner] to create a pruner.
func WithProofPruning(pruner ProofPruner) Option {
	return func(cfg *delegationConfig) error {
		cfg.pruner = pruner
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

	if cfg.pruner != nil {
		// Cast capabilities to the CaveatBuilder-typed slice expected by ProofPruner.
		castedCaps := make([]ucan.Capability[ucan.CaveatBuilder], len(capabilities))
		for i, c := range capabilities {
			castedCaps[i] = ucan.NewCapability[ucan.CaveatBuilder](c.Can(), c.With(), c.Nb())
		}

		// Pass all options except the pruner so the pruner can build its own
		// draft without recursing.
		prunedPfs, err := cfg.pruner(issuer, audience, castedCaps, cfgToOptions(cfg)...)
		if err != nil {
			return nil, fmt.Errorf("pruning proofs: %w", err)
		}

		// Replace the proof pool with the pruned subset and build the final
		// delegation normally
		cfg.prf = prunedPfs
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

	return NewDelegation(rt, bs)
}

// cfgToOptions reconstructs a slice of Options from a parsed delegationConfig,
// excluding the pruner. Used to pass a clean option set to a ProofPruner.
func cfgToOptions(cfg delegationConfig) []Option {
	var opts []Option
	if cfg.noexp {
		opts = append(opts, WithNoExpiration())
	} else if cfg.exp != nil {
		opts = append(opts, WithExpiration(*cfg.exp))
	}
	if cfg.nbf != 0 {
		opts = append(opts, WithNotBefore(cfg.nbf))
	}
	if cfg.nnc != "" {
		opts = append(opts, WithNonce(cfg.nnc))
	}
	if len(cfg.fct) > 0 {
		opts = append(opts, WithFacts(cfg.fct))
	}
	if len(cfg.prf) > 0 {
		opts = append(opts, WithProof(cfg.prf...))
	}
	return opts
}

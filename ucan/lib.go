package ucan

import (
	"fmt"

	"github.com/alanshaw/go-ucanto/ucan/crypto"
	pdm "github.com/alanshaw/go-ucanto/ucan/datamodel/payload"
	udm "github.com/alanshaw/go-ucanto/ucan/datamodel/ucan"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
)

const version = "0.9.1"

// Option is an option configuring a UCAN.
type Option func(cfg *ucanConfig) error

type ucanConfig struct {
	exp uint64
	nbf uint64
	nnc string
	fct []map[string]any
	prf []Link
}

// WithExpiration configures the expiration time in UTC seconds since Unix
// epoch. Set this to -1 for no expiration.
func WithExpiration(exp uint64) Option {
	return func(cfg *ucanConfig) error {
		cfg.exp = exp
		return nil
	}
}

// WithNotBefore configures the time in UTC seconds since Unix epoch when the
// UCAN will become valid.
func WithNotBefore(nbf uint64) Option {
	return func(cfg *ucanConfig) error {
		cfg.nbf = nbf
		return nil
	}
}

// WithNonce configures the nonce value for the UCAN.
func WithNonce(nnc string) Option {
	return func(cfg *ucanConfig) error {
		cfg.nnc = nnc
		return nil
	}
}

// WithFacts configures the facts for the UCAN.
func WithFacts(fct []map[string]any) Option {
	return func(cfg *ucanConfig) error {
		cfg.fct = fct
		return nil
	}
}

// WithProofs configures the proofs for the UCAN.
func WithProofs(prf []Link) Option {
	return func(cfg *ucanConfig) error {
		cfg.prf = prf
		return nil
	}
}

// Issue creates a new signed token with a given issuer. If expiration is
// not set it defaults to 30 seconds from now.
func Issue(issuer crypto.Signer, audience Principal, capabilities []Capability[any], options ...Option) (UCANView, error) {
	cfg := ucanConfig{}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	var capsmdl []pdm.CapabilityModel
	for _, cap := range capabilities {
		nb, ok := cap.Nb().(datamodel.Node)
		if !ok {
			return nil, fmt.Errorf("expected caveats to be a datamodel.Node")
		}
		m := pdm.CapabilityModel{
			With: cap.With(),
			Can: cap.Can(),
			Nb: nb,
		}
		capsmdl = append(capsmdl, m)
	}


	payload := pdm.PayloadModel{
		Iss: issuer.DID().String(),
		Aud: audience.DID().String(),
		Att: capabilities
	}



	model := datamodel.UCANModel{
		V: version,
		Iss: issuer.DID().String(),
		Aud: audience.DID().String(),
		S   []byte
		Att []CapabilityModel
		Prf []ipld.Link
		Exp uint64
		Fct []FactModel
		Nnc string
		Nbf uint64
	}
	return NewUCANView(&model)
}

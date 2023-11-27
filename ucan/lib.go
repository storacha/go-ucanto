package ucan

import (
	"fmt"

	"github.com/alanshaw/go-ucanto/ucan/crypto"
	pdm "github.com/alanshaw/go-ucanto/ucan/datamodel/payload"
	udm "github.com/alanshaw/go-ucanto/ucan/datamodel/ucan"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagjson"
	"github.com/ipld/go-ipld-prime/datamodel"
)

const version = "0.9.1"

// Option is an option configuring a UCAN.
type Option func(cfg *ucanConfig) error

type ucanConfig struct {
	exp uint64
	nbf uint64
	nnc string
	fct []NodeBuilder
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
func WithFacts(fct []NodeBuilder) Option {
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

// NodeBuilder builds a datamodel.Node from the underlying data.
type NodeBuilder interface {
	Build() (datamodel.Node, error)
}

// Issue creates a new signed token with a given issuer. If expiration is
// not set it defaults to 30 seconds from now.
func Issue(issuer crypto.Signer, audience Principal, capabilities []Capability[NodeBuilder], options ...Option) (UCANView, error) {
	cfg := ucanConfig{}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	var capsmdl []udm.CapabilityModel
	for _, cap := range capabilities {
		nd, err := cap.Nb().Build()
		if err != nil {
			return nil, fmt.Errorf("building caveats: %s", err)
		}
		m := udm.CapabilityModel{
			With: cap.With(),
			Can:  cap.Can(),
			Nb:   nd,
		}
		capsmdl = append(capsmdl, m)
	}

	var prfstrs []string
	for _, link := range cfg.prf {
		prfstrs = append(prfstrs, link.String())
	}

	var fctsmdl []datamodel.Node
	for _, f := range cfg.fct {
		nd, err := f.Build()
		if err != nil {
			return nil, fmt.Errorf("building fact: %s", err)
		}
		fctsmdl = append(fctsmdl, nd)
	}

	payload := pdm.PayloadModel{
		Iss: issuer.DID().String(),
		Aud: audience.DID().String(),
		Att: capsmdl,
		Prf: prfstrs,
		Exp: cfg.exp,
		Fct: fctsmdl,
		Nnc: cfg.nnc,
		Nbf: cfg.nbf,
	}
	bytes, err := ipld.Marshal(dagjson.Encode, payload, pdm.Type())
	if err != nil {
		return nil, fmt.Errorf("encoding payload: %s", err)
	}

	model := udm.UCANModel{
		V:   version,
		Iss: issuer.DID().Bytes(),
		Aud: audience.DID().Bytes(),
		S:   []byte{},
		Att: capsmdl,
		Prf: cfg.prf,
		Exp: cfg.exp,
		Fct: fctsmdl,
		Nnc: cfg.nnc,
		Nbf: cfg.nbf,
	}
	return NewUCANView(&model)
}

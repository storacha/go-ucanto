package ucan

import (
	"fmt"
	"time"

	hdm "github.com/alanshaw/go-ucanto/ucan/datamodel/header"
	pdm "github.com/alanshaw/go-ucanto/ucan/datamodel/payload"
	udm "github.com/alanshaw/go-ucanto/ucan/datamodel/ucan"
	"github.com/alanshaw/go-ucanto/ucan/formatter"
	"github.com/ipld/go-ipld-prime/datamodel"
)

const version = "0.9.1"

// Option is an option configuring a UCAN.
type Option func(cfg *ucanConfig) error

type ucanConfig struct {
	exp uint64
	nbf uint64
	nnc string
	fct []FactBuilder
	prf []Link
}

// WithExpiration configures the expiration time in UTC seconds since Unix
// epoch.
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
func WithFacts(fct []FactBuilder) Option {
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

// MapBuilder builds a map of string => datamodel.Node from the underlying data.
type MapBuilder interface {
	Build() (map[string]datamodel.Node, error)
}

type CaveatBuilder = MapBuilder
type FactBuilder = MapBuilder

// Issue creates a new signed token with a given issuer. If expiration is
// not set it defaults to 30 seconds from now.
func Issue(issuer Signer, audience Principal, capabilities []Capability[CaveatBuilder], options ...Option) (UCANView, error) {
	cfg := ucanConfig{}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	if cfg.exp == 0 {
		cfg.exp = Now() + 30
	}

	var capsmdl []udm.CapabilityModel
	for _, cap := range capabilities {
		vals, err := cap.Nb().Build()
		if err != nil {
			return nil, fmt.Errorf("building caveats: %s", err)
		}
		var keys []string
		for k := range vals {
			keys = append(keys, k)
		}
		m := udm.CapabilityModel{
			With: cap.With(),
			Can:  cap.Can(),
			Nb: udm.NbModel{
				Keys:   keys,
				Values: vals,
			},
		}
		capsmdl = append(capsmdl, m)
	}

	var prfstrs []string
	for _, link := range cfg.prf {
		prfstrs = append(prfstrs, link.String())
	}

	var fctsmdl []udm.FactModel
	for _, f := range cfg.fct {
		vals, err := f.Build()
		if err != nil {
			return nil, fmt.Errorf("building fact: %s", err)
		}
		var keys []string
		for k := range vals {
			keys = append(keys, k)
		}
		fctsmdl = append(fctsmdl, udm.FactModel{
			Keys:   keys,
			Values: vals,
		})
	}

	header := hdm.HeaderModel{
		Alg: issuer.SignatureAlgorithm(),
		Ucv: version,
		Typ: "JWT",
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
	bytes, err := encodeSignaturePayload(&header, &payload)
	if err != nil {
		return nil, fmt.Errorf("encoding signature payload: %s", err)
	}

	model := udm.UCANModel{
		V:   version,
		Iss: issuer.DID().Bytes(),
		Aud: audience.DID().Bytes(),
		S:   issuer.Sign(bytes).Bytes(),
		Att: capsmdl,
		Prf: cfg.prf,
		Exp: cfg.exp,
		Fct: fctsmdl,
		Nnc: cfg.nnc,
		Nbf: cfg.nbf,
	}
	return NewUCANView(&model)
}

func encodeSignaturePayload(header *hdm.HeaderModel, payload *pdm.PayloadModel) ([]byte, error) {
	str, err := formatter.FormatSignPayload(header, payload)
	if err != nil {
		return nil, err
	}
	return []byte(str), nil
}

// IsExpired checks if a UCAN is expired.
func IsExpired(ucan UCANView) bool {
	return ucan.Expiration() <= Now()
}

// IsTooEarly checks if a UCAN is not active yet.
func IsTooEarly(ucan UCANView) bool {
	nbf := ucan.NotBefore()
	return nbf != 0 && Now() <= nbf
}

// Now returns a UTC Unix timestamp for comparing it against time window of the
// UCAN.
func Now() uint64 {
	return uint64(time.Now().Unix())
}

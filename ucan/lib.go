package ucan

import (
	"fmt"
	"time"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/storacha/go-ucanto/ucan/crypto/signature"
	pdm "github.com/storacha/go-ucanto/ucan/datamodel/payload"
	udm "github.com/storacha/go-ucanto/ucan/datamodel/ucan"
	"github.com/storacha/go-ucanto/ucan/formatter"
)

const version = "0.9.1"

// Option is an option configuring a UCAN.
type Option func(cfg *ucanConfig) error

type ucanConfig struct {
	exp   *UTCUnixTimestamp
	noexp bool
	nbf   UTCUnixTimestamp
	nnc   string
	fct   []FactBuilder
	prf   []Link
}

// WithExpiration configures the expiration time in UTC seconds since Unix
// epoch.
func WithExpiration(exp UTCUnixTimestamp) Option {
	return func(cfg *ucanConfig) error {
		cfg.exp = &exp
		cfg.noexp = false
		return nil
	}
}

// WithNoExpiration configures the UCAN to never expire.
//
// WARNING: this will cause the delegation to be valid FOREVER, unless revoked.
func WithNoExpiration() Option {
	return func(cfg *ucanConfig) error {
		cfg.exp = nil
		cfg.noexp = true
		return nil
	}
}

// WithNotBefore configures the time in UTC seconds since Unix epoch when the
// UCAN will become valid.
func WithNotBefore(nbf int) Option {
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

// WithProof configures the proofs for the UCAN.
func WithProof(prf ...Link) Option {
	return func(cfg *ucanConfig) error {
		cfg.prf = prf
		return nil
	}
}

// MapBuilder builds a map of string => datamodel.Node from the underlying data.
type MapBuilder interface {
	ToIPLD() (map[string]datamodel.Node, error)
}

type FactBuilder = MapBuilder

// CaveatBuilder builds a datamodel.Node from the underlying data.
type CaveatBuilder interface {
	ToIPLD() (datamodel.Node, error)
}

// Issue creates a new signed token with a given issuer. If expiration is
// not set it defaults to 30 seconds from now.
func Issue[C CaveatBuilder](issuer Signer, audience Principal, capabilities []Capability[C], options ...Option) (View, error) {
	cfg := ucanConfig{}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	var exp *int
	if !cfg.noexp {
		if cfg.exp == nil {
			in30s := int(Now() + 30)
			exp = &in30s
		} else {
			exp = cfg.exp
		}
	}

	var capsmdl []udm.CapabilityModel
	for _, cap := range capabilities {
		nb, err := cap.Nb().ToIPLD()
		if err != nil {
			return nil, fmt.Errorf("building caveats: %w", err)
		}
		m := udm.CapabilityModel{
			With: cap.With(),
			Can:  cap.Can(),
			Nb:   nb,
		}
		capsmdl = append(capsmdl, m)
	}

	var prfstrs []string
	for _, link := range cfg.prf {
		prfstrs = append(prfstrs, link.String())
	}

	var fctsmdl []udm.FactModel
	for _, f := range cfg.fct {
		vals, err := f.ToIPLD()
		if err != nil {
			return nil, fmt.Errorf("building fact: %w", err)
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

	payload := pdm.PayloadModel{
		Iss: issuer.DID().String(),
		Aud: audience.DID().String(),
		Att: capsmdl,
		Prf: prfstrs,
		Exp: exp,
		Fct: fctsmdl,
	}
	if cfg.nnc != "" {
		payload.Nnc = &cfg.nnc
	}
	if cfg.nbf != 0 {
		payload.Nbf = &cfg.nbf
	}
	bytes, err := encodeSignaturePayload(payload, version, issuer.SignatureAlgorithm())
	if err != nil {
		return nil, fmt.Errorf("encoding signature payload: %w", err)
	}

	model := udm.UCANModel{
		V:   version,
		S:   issuer.Sign(bytes).Bytes(),
		Iss: issuer.DID().Bytes(),
		Aud: audience.DID().Bytes(),
		Att: capsmdl,
		Prf: cfg.prf,
		Exp: exp,
		Fct: fctsmdl,
	}
	if cfg.nnc != "" {
		model.Nnc = &cfg.nnc
	}
	if cfg.nbf != 0 {
		model.Nbf = &cfg.nbf
	}
	return NewUCAN(&model)
}

func encodeSignaturePayload(payload pdm.PayloadModel, version string, algorithm string) ([]byte, error) {
	str, err := formatter.FormatSignPayload(payload, version, algorithm)
	if err != nil {
		return nil, err
	}
	return []byte(str), nil
}

func VerifySignature(ucan View, verifier Verifier) (bool, error) {
	alg, err := signature.CodeName(ucan.Signature().Code())
	if err != nil {
		return false, err
	}

	var prfstrs []string
	for _, link := range ucan.Proofs() {
		prfstrs = append(prfstrs, link.String())
	}

	payload := pdm.PayloadModel{
		Iss: ucan.Issuer().DID().String(),
		Aud: ucan.Audience().DID().String(),
		Att: ucan.Model().Att,
		Prf: prfstrs,
		Exp: ucan.Expiration(),
		Fct: ucan.Model().Fct,
		Nnc: ucan.Model().Nnc,
		Nbf: ucan.Model().Nbf,
	}

	msg, err := encodeSignaturePayload(payload, ucan.Version(), alg)
	if err != nil {
		return false, err
	}

	return ucan.Issuer().DID() == verifier.DID() && verifier.Verify(msg, ucan.Signature()), nil
}

// IsExpired checks if a UCAN is expired.
func IsExpired(ucan UCAN) bool {
	exp := ucan.Expiration()
	if exp == nil {
		return false
	}
	return *exp <= Now()
}

// IsTooEarly checks if a UCAN is not active yet.
func IsTooEarly(ucan UCAN) bool {
	nbf := ucan.NotBefore()
	return nbf != 0 && Now() <= nbf
}

// Now returns a UTC Unix timestamp for comparing it against time window of the
// UCAN.
func Now() UTCUnixTimestamp {
	return UTCUnixTimestamp(time.Now().Unix())
}

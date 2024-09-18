package validator

import (
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/ucan"
)

type Authorization[Caveats any] interface {
	Audience() ucan.Principal
	Capability() ucan.Capability[Caveats]
	Delegation() delegation.Delegation
	Issuer() ucan.Principal
	Proofs() []Authorization[Caveats]
}

type authorization[Caveats any] struct {
	match  Match[Caveats]
	proofs []Authorization[Caveats]
}

func (a authorization[Caveats]) Audience() ucan.Principal {
	return a.Delegation().Audience()
}

func (a authorization[Caveats]) Capability() ucan.Capability[Caveats] {
	return a.match.Value()
}

func (a authorization[Caveats]) Delegation() delegation.Delegation {
	return a.match.Source()[0].Delegation()
}

func (a authorization[Caveats]) Issuer() ucan.Principal {
	return a.Delegation().Issuer()
}

func (a authorization[Caveats]) Proofs() []Authorization[Caveats] {
	return a.proofs
}

func NewAuthorization[Caveats any](match Match[Caveats], proofs []Authorization[Caveats]) Authorization[Caveats] {
	return authorization[Caveats]{match, proofs}
}

type unknownauth[C any] struct {
	auth Authorization[C]
}

func (a unknownauth[C]) Audience() ucan.Principal {
	return a.auth.Audience()
}

func (a unknownauth[C]) Capability() ucan.Capability[any] {
	cap := a.auth.Capability()
	return ucan.NewCapability[any](cap.Can(), cap.With(), cap.Nb())
}

func (a unknownauth[C]) Delegation() delegation.Delegation {
	return a.auth.Delegation()
}

func (a unknownauth[C]) Issuer() ucan.Principal {
	return a.Delegation().Issuer()
}

func (a unknownauth[C]) Proofs() []Authorization[any] {
	var prf []Authorization[any]
	for _, p := range a.auth.Proofs() {
		prf = append(prf, ConvertUnknownAuthorization(p))
	}
	return prf
}

// ConvertUnknownAuthorization converts an Authorization[Caveats] to Authorization[any]
func ConvertUnknownAuthorization[Caveats any](auth Authorization[Caveats]) Authorization[any] {
	return unknownauth[Caveats]{auth}
}

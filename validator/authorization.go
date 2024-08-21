package validator

import (
	"github.com/storacha-network/go-ucanto/core/delegation"
	"github.com/storacha-network/go-ucanto/ucan"
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

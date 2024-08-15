package validator

import (
	"github.com/storacha-network/go-ucanto/ucan"
)

type Authorization[Caveats any] interface {
	Capability() ucan.Capability[Caveats]
}

type authorization[Caveats any] struct {
	capability ucan.Capability[Caveats]
}

func (a authorization[Caveats]) Capability() ucan.Capability[Caveats] {
	return a.capability
}

func NewAuthorization[Caveats any](capability ucan.Capability[Caveats]) Authorization[Caveats] {
	return authorization[Caveats]{capability: capability}
}

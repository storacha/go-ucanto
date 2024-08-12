package validator

import (
	"github.com/storacha-network/go-ucanto/core/delegation"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/core/schema"
	"github.com/storacha-network/go-ucanto/ucan"
)

type Source interface {
	Capability() ucan.Capability[any]
	Delegation() delegation.Delegation
}

type CapabilityParser[Caveats any] interface {
	// New instantiates a new capability from the passed options
	New(with ucan.Resource, nb Caveats) ucan.Capability[Caveats]
	Match(source Source) result.Result[ucan.Capability[Caveats], result.Failure]
}

type Descriptor[I, O any] interface {
	Can() ucan.Ability
	With() schema.Reader[string, ucan.Resource]
	Nb() schema.Reader[I, O]
}

type descriptor[Caveats any] struct {
	can  ucan.Ability
	with schema.Reader[string, ucan.Resource]
	nb   schema.Reader[any, Caveats]
}

func (d *descriptor[C]) Can() ucan.Ability {
	return d.can
}

func (d *descriptor[C]) With() schema.Reader[string, ucan.Resource] {
	return d.with
}

func (d *descriptor[C]) Nb() schema.Reader[any, C] {
	return d.nb
}

type capability[Caveats any] struct {
	descriptor Descriptor[any, Caveats]
}

func (c *capability[Caveats]) Match(source Source) result.Result[ucan.Capability[Caveats], result.Failure] {
	return parseCapability(c.descriptor, source)
}

func (c *capability[Caveats]) New(with ucan.Resource, nb Caveats) ucan.Capability[Caveats] {
	return ucan.NewCapability(c.descriptor.Can(), with, nb)
}

func NewCapability[Caveats any](can ucan.Ability, with schema.Reader[string, ucan.Resource], nb schema.Reader[any, Caveats]) CapabilityParser[Caveats] {
	d := descriptor[Caveats]{can: can, with: with, nb: nb}
	return &capability[Caveats]{descriptor: &d}
}

func parseCapability[O any](descriptor Descriptor[any, O], source Source) result.Result[ucan.Capability[O], result.Failure] {
	cap := source.Capability()
	return result.MatchResultR1(descriptor.With().Read(cap.With()), func(with ucan.Resource) result.Result[ucan.Capability[O], result.Failure] {
		return result.MapOk(descriptor.Nb().Read(cap.Nb()), func(nb O) ucan.Capability[O] {
			pcap := ucan.NewCapability(cap.Can(), with, nb)
			return pcap
		})
	}, func(x result.Failure) result.Result[ucan.Capability[O], result.Failure] {
		return result.Error[ucan.Capability[O]](x)
	})
}

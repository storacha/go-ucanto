package validator

import (
	"fmt"

	"github.com/storacha-network/go-ucanto/core/delegation"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/core/result/failure"
	"github.com/storacha-network/go-ucanto/core/schema"
	"github.com/storacha-network/go-ucanto/ucan"
)

type Source interface {
	Capability() ucan.Capability[any]
	Delegation() delegation.Delegation
}

type source struct {
	capability ucan.Capability[any]
	delegation delegation.Delegation
}

func (s source) Capability() ucan.Capability[any] {
	return s.capability
}

func (s source) Delegation() delegation.Delegation {
	return s.delegation
}

type Matcher[Caveats any] interface {
	Match(source Source) result.Result[Match[Caveats], InvalidCapability]
}

type Selector[Caveats any] interface {
	Select(sources []Source) ([]Match[Caveats], []DelegationError, []ucan.Capability[any], error)
}

type Match[Caveats any] interface {
	Source() []Source
	Value() ucan.Capability[Caveats]
	Proofs() []delegation.Delegation
	Prune(context CanIssuer[Caveats]) Match[Caveats]
}

type match[Caveats any] struct {
	sources    []Source
	value      ucan.Capability[Caveats]
	descriptor Descriptor[any, Caveats]
}

func (m match[Caveats]) Proofs() []delegation.Delegation {
	return []delegation.Delegation{m.sources[0].Delegation()}
}

func (m match[Caveats]) Prune(context CanIssuer[Caveats]) Match[Caveats] {
	if context.CanIssue(m.value, m.sources[0].Delegation().Issuer().DID()) {
		return m
	}
	return nil
}

func (m match[Caveats]) Source() []Source {
	return m.sources
}

func (m match[Caveats]) Value() ucan.Capability[Caveats] {
	return m.value
}

func NewMatch[Caveats any](source Source, capability ucan.Capability[Caveats], descriptor Descriptor[any, Caveats]) Match[Caveats] {
	return match[Caveats]{[]Source{source}, capability, descriptor}
}

type CapabilityParser[Caveats any] interface {
	Matcher[Caveats]
	Selector[Caveats]
	Can() ucan.Ability
	// New creates a new capability from the passed options.
	New(with ucan.Resource, nb Caveats) ucan.Capability[Caveats]
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

func (c *capability[Caveats]) Can() ucan.Ability {
	return c.descriptor.Can()
}

func (c *capability[Caveats]) Select(capabilities []Source) ([]Match[Caveats], []DelegationError, []ucan.Capability[any], error) {
	return Select(c, capabilities)
}

func (c *capability[Caveats]) Match(source Source) result.Result[Match[Caveats], InvalidCapability] {
	return result.MapOk(
		parseCapability(c.descriptor, source),
		func(cap ucan.Capability[Caveats]) Match[Caveats] {
			return NewMatch(source, cap, c.descriptor)
		},
	)
}

func (c *capability[Caveats]) String() string {
	return fmt.Sprintf(`{can:"%s"}`, c.Can())
}

func (c *capability[Caveats]) New(with ucan.Resource, nb Caveats) ucan.Capability[Caveats] {
	return ucan.NewCapability(c.descriptor.Can(), with, nb)
}

func NewCapability[Caveats any](can ucan.Ability, with schema.Reader[string, ucan.Resource], nb schema.Reader[any, Caveats]) CapabilityParser[Caveats] {
	d := descriptor[Caveats]{can: can, with: with, nb: nb}
	return &capability[Caveats]{descriptor: &d}
}

func parseCapability[O any](descriptor Descriptor[any, O], source Source) result.Result[ucan.Capability[O], InvalidCapability] {
	cap := source.Capability()

	if descriptor.Can() != cap.Can() {
		return result.Error[ucan.Capability[O], InvalidCapability](NewUnknownCapabilityError(cap))
	}

	return result.MatchResultR1(
		descriptor.With().Read(cap.With()),
		func(with ucan.Resource) result.Result[ucan.Capability[O], InvalidCapability] {
			return result.MapResultR0(
				descriptor.Nb().Read(cap.Nb()),
				func(nb O) ucan.Capability[O] {
					pcap := ucan.NewCapability(cap.Can(), with, nb)
					return pcap
				},
				func(x failure.Failure) InvalidCapability {
					return NewMalformedCapabilityError(cap, x)
				},
			)
		},
		func(x failure.Failure) result.Result[ucan.Capability[O], InvalidCapability] {
			return result.Error[ucan.Capability[O], InvalidCapability](NewMalformedCapabilityError(cap, x))
		},
	)
}

func Select[Caveats any](matcher Matcher[Caveats], capabilities []Source) (matches []Match[Caveats], errors []DelegationError, unknowns []ucan.Capability[any], err error) {
	for _, capability := range capabilities {
		err = result.MatchResultR1(
			matcher.Match(capability),
			func(match Match[Caveats]) error {
				matches = append(matches, match)
				return nil
			},
			func(err InvalidCapability) error {
				if uerr, ok := err.(UnknownCapability); ok {
					unknowns = append(unknowns, uerr.Capability())
				}
				if serr, ok := err.(DelegationSubError); ok {
					errors = append(errors, NewDelegationError([]DelegationSubError{serr}, capability.Capability()))
				}
				return fmt.Errorf("unexpected error type in match result")
			},
		)
		if err != nil {
			return
		}
	}
	return
}

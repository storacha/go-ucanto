package validator

import (
	"fmt"
	"strings"

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

func NewSource(capability ucan.Capability[any], delegation delegation.Delegation) Source {
	return source{capability, delegation}
}

type Matcher[Caveats any] interface {
	Match(source Source) result.Result[Match[Caveats], InvalidCapability]
}

type Selector[Caveats any] interface {
	Select(sources []Source) ([]Match[Caveats], []DelegationError, []ucan.Capability[any], error)
}

type Match[Caveats any] interface {
	Selector[Caveats]
	Source() []Source
	Value() ucan.Capability[Caveats]
	Proofs() []delegation.Delegation
	Prune(context CanIssuer[Caveats]) Match[Caveats]
}

type match[Caveats any] struct {
	sources    []Source
	value      ucan.Capability[Caveats]
	descriptor Descriptor[Caveats]
}

func (m match[Caveats]) Proofs() []delegation.Delegation {
	return []delegation.Delegation{m.sources[0].Delegation()}
}

func (m match[Caveats]) Prune(context CanIssuer[Caveats]) Match[Caveats] {
	if context.CanIssue(m.value, m.sources[0].Delegation().Issuer().DID()) {
		return nil
	}
	return m
}

func (m match[Caveats]) Source() []Source {
	return m.sources
}

func (m match[Caveats]) Value() ucan.Capability[Caveats] {
	return m.value
}

func (m match[Caveats]) Select(sources []Source) (matches []Match[Caveats], errors []DelegationError, unknowns []ucan.Capability[any], err error) {
	for _, source := range sources {
		err = result.MatchResultR1(
			ResolveCapability(m.descriptor, m.value, source),
			func(cap ucan.Capability[Caveats]) error {
				result.MatchResultR0(
					m.descriptor.Derives(m.value, cap),
					func(_ result.Unit) {
						matches = append(matches, NewMatch(source, cap, m.descriptor))
					},
					func(x failure.Failure) {
						errors = append(errors, NewDelegationError([]DelegationSubError{NewEscalatedCapabilityError(m.value, cap, x)}, m))
					},
				)
				return nil
			},
			func(x InvalidCapability) error {
				if uerr, ok := x.(UnknownCapability); ok {
					unknowns = append(unknowns, uerr.Capability())
					return nil
				}
				if merr, ok := x.(MalformedCapability); ok {
					errors = append(errors, NewDelegationError([]DelegationSubError{merr}, m))
					return nil
				}
				return fmt.Errorf("unexpected error type in resolved capability result: %w", err)
			},
		)
		if err != nil {
			return
		}
	}
	return
}

func (m match[Caveats]) String() string {
	s, _ := m.value.MarshalJSON()
	return string(s)
}

func NewMatch[Caveats any](source Source, capability ucan.Capability[Caveats], descriptor Descriptor[Caveats]) Match[Caveats] {
	return match[Caveats]{[]Source{source}, capability, descriptor}
}

type CapabilityParser[Caveats any] interface {
	Matcher[Caveats]
	Selector[Caveats]
	Can() ucan.Ability
	// New creates a new capability from the passed options.
	New(with ucan.Resource, nb Caveats) ucan.Capability[Caveats]
	// Invoke creates an invocation of this capability.
	// Invoke(with ucan.Resource, nb Caveats) (invocation.IssuedInvocation, error)
}

type Derivable[Caveats any] interface {
	// Derives determines if a capability is derivable from another.
	Derives(claimed, delegated ucan.Capability[Caveats]) result.Result[result.Unit, failure.Failure]
}

type DerivesFunc[Caveats any] func(claimed, delegated ucan.Capability[Caveats]) result.Result[result.Unit, failure.Failure]

type Descriptor[Caveats any] interface {
	Derivable[Caveats]
	Can() ucan.Ability
	With() schema.Reader[string, ucan.Resource]
	Nb() schema.Reader[any, Caveats]
}

type descriptor[Caveats any] struct {
	can     ucan.Ability
	with    schema.Reader[string, ucan.Resource]
	nb      schema.Reader[any, Caveats]
	derives DerivesFunc[Caveats]
}

func (d descriptor[C]) Can() ucan.Ability {
	return d.can
}

func (d descriptor[C]) With() schema.Reader[string, ucan.Resource] {
	return d.with
}

func (d descriptor[C]) Nb() schema.Reader[any, C] {
	return d.nb
}

func (d descriptor[C]) Derives(parent, child ucan.Capability[C]) result.Result[result.Unit, failure.Failure] {
	return d.derives(parent, child)
}

type capability[Caveats any] struct {
	descriptor Descriptor[Caveats]
}

func (c capability[Caveats]) Can() ucan.Ability {
	return c.descriptor.Can()
}

func (c capability[Caveats]) Select(capabilities []Source) ([]Match[Caveats], []DelegationError, []ucan.Capability[any], error) {
	return Select(c, capabilities)
}

func (c capability[Caveats]) Match(source Source) result.Result[Match[Caveats], InvalidCapability] {
	return result.MapOk(
		ParseCapability(c.descriptor, source),
		func(cap ucan.Capability[Caveats]) Match[Caveats] {
			return NewMatch(source, cap, c.descriptor)
		},
	)
}

func (c capability[Caveats]) String() string {
	return fmt.Sprintf(`{can:"%s"}`, c.Can())
}

func (c capability[Caveats]) New(with ucan.Resource, nb Caveats) ucan.Capability[Caveats] {
	return ucan.NewCapability(c.descriptor.Can(), with, nb)
}

// func (c capability[Caveats]) Invoke(issuer ucan.Signer, audience ucan.Principal, with ucan.Resource, nb Caveats, options ...delegation.Option) (invocation.IssuedInvocation, error) {
// 	return invocation.Invoke(issuer, audience, c.New(with, nb), options...)
// }

func NewCapability[Caveats any](
	can ucan.Ability,
	with schema.Reader[string, ucan.Resource],
	nb schema.Reader[any, Caveats],
	derives DerivesFunc[Caveats],
) CapabilityParser[Caveats] {
	if derives == nil {
		derives = DefaultDerives
	}
	d := descriptor[Caveats]{can, with, nb, derives}
	return &capability[Caveats]{descriptor: d}
}

func ParseCapability[O any](descriptor Descriptor[O], source Source) result.Result[ucan.Capability[O], InvalidCapability] {
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
			func(x InvalidCapability) error {
				if ux, ok := x.(UnknownCapability); ok {
					unknowns = append(unknowns, ux.Capability())
					return nil
				}
				if sx, ok := x.(DelegationSubError); ok {
					errors = append(errors, NewDelegationError([]DelegationSubError{sx}, capability.Capability()))
					return nil
				}
				return fmt.Errorf("unexpected error type in match result: %w", x)
			},
		)
		if err != nil {
			return
		}
	}
	return
}

// ResolveCapability resolves delegated capability `source` from the `claimed`
// capability using provided capability `parser`. It is similar to
// `parseCapability` except `source` here is treated as capability pattern which
// is matched against the `claimed` capability. This means we resolve `can` and
// `with` fields from the `claimed` capability and...
// TODO: inherit all missing `nb` fields from the claimed capability.
func ResolveCapability[Caveats any](descriptor Descriptor[Caveats], claimed ucan.Capability[Caveats], source Source) result.Result[ucan.Capability[Caveats], InvalidCapability] {
	can := ResolveAbility(source.Capability().Can(), claimed.Can())
	if can == "" {
		return result.Error[ucan.Capability[Caveats], InvalidCapability](NewUnknownCapabilityError(source.Capability()))
	}

	resource := ResolveResource(source.Capability().With(), claimed.With())
	if resource == "" {
		resource = source.Capability().With()
	}

	return result.MatchResultR1(
		descriptor.With().Read(resource),
		func(uri string) result.Result[ucan.Capability[Caveats], InvalidCapability] {
			return result.MapResultR0(
				// TODO: inherit missing fields
				descriptor.Nb().Read(claimed),
				func(nb Caveats) ucan.Capability[Caveats] {
					return ucan.NewCapability(can, resource, nb)
				},
				func(x failure.Failure) InvalidCapability {
					return NewMalformedCapabilityError(source.Capability(), x)
				},
			)
		},
		func(x failure.Failure) result.Result[ucan.Capability[Caveats], InvalidCapability] {
			return result.Error[ucan.Capability[Caveats], InvalidCapability](NewMalformedCapabilityError(source.Capability(), x))
		},
	)
}

// ResolveAbility resolves ability `pattern` of the delegated capability from
// the ability of the claimed capability. If pattern matches returns claimed
// ability otherwise returns "".
//
//   - pattern "*"       can "store/add" → "store/add"
//   - pattern "store/*" can "store/add" → "store/add"
//   - pattern "*"       can "store/add" → "store/add"
//   - pattern "*"       can "store/add" → ""
//   - pattern "*"       can "store/add" → ""
//   - pattern "*"       can "store/add" → ""
func ResolveAbility(pattern string, can ucan.Ability) ucan.Ability {
	if pattern == can || pattern == "*" {
		return can
	}
	if strings.HasSuffix(pattern, "/*") && strings.HasPrefix(can, pattern[0:len(pattern)-1]) {
		return can
	}
	return ""
}

// ResolveResource resolves `source` resource of the delegated capability from
// the resource `uri` of the claimed capability. If `source` is `"ucan:*""` or
// matches `uri` then it returns `uri` back otherwise it returns "".
//
//   - source "ucan:*"         uri "did:key:zAlice"      → "did:key:zAlice"
//   - source "ucan:*"         uri "https://example.com" → "https://example.com"
//   - source "did:*"          uri "did:key:zAlice"      → ""
//   - source "did:key:zAlice" uri "did:key:zAlice"      → "did:key:zAlice"
func ResolveResource(source string, uri ucan.Resource) ucan.Resource {
	if source == uri || source == "ucan:*" {
		return uri
	}
	return ""
}

func DefaultDerives[Caveats any](claimed, delegated ucan.Capability[Caveats]) result.Result[result.Unit, failure.Failure] {
	dres := delegated.With()
	cres := claimed.With()

	if strings.HasSuffix(dres, "*") {
		if !strings.HasPrefix(cres, dres[0:len(dres)-1]) {
			return result.Error[result.Unit](schema.NewSchemaError(fmt.Sprintf("Resource %s does not match delegated %s", cres, dres)))
		}
	} else if dres != cres {
		return result.Error[result.Unit](schema.NewSchemaError(fmt.Sprintf("Resource %s is not contained by %s", cres, dres)))
	}

	// TODO: is it possible to ensure claimed caveats match delegated caveats?

	return result.Ok[result.Unit, failure.Failure](struct{}{})
}

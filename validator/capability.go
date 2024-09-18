package validator

import (
	"fmt"
	"strings"

	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/result/failure"
	"github.com/storacha/go-ucanto/core/schema"
	"github.com/storacha/go-ucanto/ucan"
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
	Match(source Source) (Match[Caveats], InvalidCapability)
}

type Selector[Caveats any] interface {
	Select(sources []Source) ([]Match[Caveats], []DelegationError, []ucan.Capability[any])
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

func (m match[Caveats]) Select(sources []Source) (matches []Match[Caveats], errors []DelegationError, unknowns []ucan.Capability[any]) {
	for _, source := range sources {
		cap, err := ResolveCapability(m.descriptor, m.value, source)
		if err != nil {
			if uerr, ok := err.(UnknownCapability); ok {
				unknowns = append(unknowns, uerr.Capability())
			} else if merr, ok := err.(MalformedCapability); ok {
				errors = append(errors, NewDelegationError([]DelegationSubError{merr}, m))
			} else {
				panic(fmt.Errorf("unexpected error type in resolved capability result: %w", err))
			}
			continue
		}

		derr := m.descriptor.Derives(m.value, cap)
		if derr != nil {
			errors = append(errors, NewDelegationError([]DelegationSubError{NewEscalatedCapabilityError(m.value, cap, derr)}, m))
			continue
		}

		matches = append(matches, NewMatch(source, cap, m.descriptor))
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
	// Delegate creates a new signed token for this capability. If expiration is
	// not set it defaults to 30 seconds from now.
	Delegate(issuer ucan.Signer, audience ucan.Principal, with ucan.Resource, nb Caveats, options ...delegation.Option) (delegation.Delegation, error)
	// Invoke creates an invocation of this capability.
	Invoke(issuer ucan.Signer, audience ucan.Principal, with ucan.Resource, nb Caveats, options ...delegation.Option) (invocation.IssuedInvocation, error)
}

type Derivable[Caveats any] interface {
	// Derives determines if a capability is derivable from another. Return `nil`
	// to indicate the delegated capability can be derived from the claimed
	// capability.
	Derives(claimed, delegated ucan.Capability[Caveats]) failure.Failure
}

// DerivesFunc determines if a capability is derivable from another. Return
// `nil` to indicate the delegated capability can be derived from the claimed
// capability.
type DerivesFunc[Caveats any] func(claimed, delegated ucan.Capability[Caveats]) failure.Failure

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

func (d descriptor[C]) Derives(parent, child ucan.Capability[C]) failure.Failure {
	return d.derives(parent, child)
}

type capability[Caveats any] struct {
	descriptor Descriptor[Caveats]
}

func (c capability[Caveats]) Can() ucan.Ability {
	return c.descriptor.Can()
}

func (c capability[Caveats]) Select(capabilities []Source) ([]Match[Caveats], []DelegationError, []ucan.Capability[any]) {
	return Select(c, capabilities)
}

func (c capability[Caveats]) Match(source Source) (Match[Caveats], InvalidCapability) {
	cap, err := ParseCapability(c.descriptor, source)
	if err != nil {
		return nil, err
	}
	return NewMatch(source, cap, c.descriptor), nil
}

func (c capability[Caveats]) String() string {
	return fmt.Sprintf(`{can:"%s"}`, c.Can())
}

func (c capability[Caveats]) New(with ucan.Resource, nb Caveats) ucan.Capability[Caveats] {
	return ucan.NewCapability(c.descriptor.Can(), with, nb)
}

func (c capability[Caveats]) Delegate(issuer ucan.Signer, audience ucan.Principal, with ucan.Resource, nb Caveats, options ...delegation.Option) (delegation.Delegation, error) {
	if bc, ok := any(nb).(ucan.CaveatBuilder); ok {
		caps := []ucan.Capability[ucan.CaveatBuilder]{ucan.NewCapability(c.Can(), with, bc)}
		return delegation.Delegate(issuer, audience, caps, options...)
	}
	return nil, fmt.Errorf("not an IPLD builder: %v", nb)
}

func (c capability[Caveats]) Invoke(issuer ucan.Signer, audience ucan.Principal, with ucan.Resource, nb Caveats, options ...delegation.Option) (invocation.IssuedInvocation, error) {
	if bc, ok := any(nb).(ucan.CaveatBuilder); ok {
		cap := ucan.NewCapability(c.Can(), with, bc)
		return invocation.Invoke(issuer, audience, cap, options...)
	}
	return nil, fmt.Errorf("not an IPLD builder: %v", nb)
}

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

func ParseCapability[O any](descriptor Descriptor[O], source Source) (ucan.Capability[O], InvalidCapability) {
	cap := source.Capability()

	if descriptor.Can() != cap.Can() {
		return nil, NewUnknownCapabilityError(cap)
	}

	uri, err := descriptor.With().Read(cap.With())
	if err != nil {
		return nil, NewMalformedCapabilityError(cap, err)
	}

	nb, err := descriptor.Nb().Read(cap.Nb())
	if err != nil {
		return nil, NewMalformedCapabilityError(cap, err)
	}

	return ucan.NewCapability(cap.Can(), uri, nb), nil
}

func Select[Caveats any](matcher Matcher[Caveats], capabilities []Source) (matches []Match[Caveats], errors []DelegationError, unknowns []ucan.Capability[any]) {
	for _, capability := range capabilities {
		match, err := matcher.Match(capability)
		if err != nil {
			if ux, ok := err.(UnknownCapability); ok {
				unknowns = append(unknowns, ux.Capability())
			} else if sx, ok := err.(DelegationSubError); ok {
				errors = append(errors, NewDelegationError([]DelegationSubError{sx}, capability.Capability()))
			} else {
				panic(fmt.Errorf("unexpected error type in match result: %w", err))
			}
			continue
		}
		matches = append(matches, match)
	}
	return
}

// ResolveCapability resolves delegated capability `source` from the `claimed`
// capability using provided capability `parser`. It is similar to
// [ParseCapability] except `source` here is treated as capability pattern which
// is matched against the `claimed` capability. This means we resolve `can` and
// `with` fields from the `claimed` capability and...
// TODO: inherit all missing `nb` fields from the claimed capability.
func ResolveCapability[Caveats any](descriptor Descriptor[Caveats], claimed ucan.Capability[Caveats], source Source) (ucan.Capability[Caveats], InvalidCapability) {
	can := ResolveAbility(source.Capability().Can(), claimed.Can())
	if can == "" {
		return nil, NewUnknownCapabilityError(source.Capability())
	}

	resource := ResolveResource(source.Capability().With(), claimed.With())
	if resource == "" {
		resource = source.Capability().With()
	}

	uri, err := descriptor.With().Read(resource)
	if err != nil {
		return nil, NewMalformedCapabilityError(source.Capability(), err)
	}

	// TODO: inherit missing fields
	nb, err := descriptor.Nb().Read(claimed.Nb())
	if err != nil {
		return nil, NewMalformedCapabilityError(source.Capability(), err)
	}

	return ucan.NewCapability(can, uri, nb), nil
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

func DefaultDerives[Caveats any](claimed, delegated ucan.Capability[Caveats]) failure.Failure {
	dres := delegated.With()
	cres := claimed.With()

	if strings.HasSuffix(dres, "*") {
		if !strings.HasPrefix(cres, dres[0:len(dres)-1]) {
			return schema.NewSchemaError(fmt.Sprintf("Resource %s does not match delegated %s", cres, dres))
		}
	} else if dres != cres {
		return schema.NewSchemaError(fmt.Sprintf("Resource %s is not contained by %s", cres, dres))
	}

	// TODO: is it possible to ensure claimed caveats match delegated caveats?
	return nil
}

package ucan

type capability[T any] struct {
	can  Ability
	nb   T
	with Resource
}

var _ Capability[any] = (*capability[any])(nil)

func (c *capability[T]) Can() Ability {
	return c.can
}

func (c *capability[T]) Nb() T {
	return c.nb
}

func (c *capability[T]) With() Resource {
	return c.with
}

func NewCapability[Caveats any](can Ability, with Resource, nb Caveats) Capability[Caveats] {
	return &capability[Caveats]{
		can:  can,
		with: with,
		nb:   nb,
	}
}

package ucan

import (
	"encoding/json"
)

type jsonModel struct {
	With Resource    `json:"with"`
	Can  Ability     `json:"can"`
	Nb   interface{} `json:"nb,omitempty"`
}

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

func (c *capability[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonModel{
		With: c.with,
		Can:  c.can,
		Nb:   c.nb,
	})
}

func NewCapability[Caveats any](can Ability, with Resource, nb Caveats) Capability[Caveats] {
	return &capability[Caveats]{
		can:  can,
		with: with,
		nb:   nb,
	}
}

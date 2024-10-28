package fx

import (
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/ucan"
)

type Effects interface {
	Fork() []Effect
	Join() Effect
}

type effects struct {
	fork []Effect
	join Effect
}

func (fx effects) Fork() []Effect {
	return fx.fork
}

func (fx effects) Join() Effect {
	return fx.join
}

// Option is an option configuring effects.
type Option func(fx *effects) error

// WithFork configures the forks for the receipt.
func WithFork(forks ...Effect) Option {
	return func(fx *effects) error {
		fx.fork = forks
		return nil
	}
}

// WithJoin configures the join for the receipt.
func WithJoin(join Effect) Option {
	return func(fx *effects) error {
		fx.join = join
		return nil
	}
}

func NewEffects(opts ...Option) Effects {
	var fx effects
	for _, opt := range opts {
		opt(&fx)
	}
	return fx
}

// Effect is either an invocation or a link to one.
type Effect struct {
	invocation invocation.Invocation
	link       ucan.Link
}

// Invocation returns the invocation if it is available.
func (e Effect) Invocation() (invocation.Invocation, bool) {
	return e.invocation, e.invocation != nil
}

// Link returns the invocation root link.
func (e Effect) Link() ucan.Link {
	if e.invocation != nil {
		return e.invocation.Link()
	}
	return e.link
}

func FromLink(link ucan.Link) Effect {
	return Effect{nil, link}
}

func FromInvocation(invocation invocation.Invocation) Effect {
	return Effect{invocation, nil}
}

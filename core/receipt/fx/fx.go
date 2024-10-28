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

func NewEffects(fork []Effect, join Effect) Effects {
	return effects{fork, join}
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

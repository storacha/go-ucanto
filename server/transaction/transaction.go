package transaction

import (
	"github.com/storacha/go-ucanto/core/receipt/fx"
	"github.com/storacha/go-ucanto/core/result"
)

// Transaction defines a result & effect pair, used by provider that wishes to
// return results that have effects.
type Transaction[O any, X any] interface {
	Out() result.Result[O, X]
	Fx() fx.Effects
}

type transaction[O, X any] struct {
	out result.Result[O, X]
	fx  fx.Effects
}

func (t transaction[O, X]) Out() result.Result[O, X] {
	return t.out
}

func (t transaction[O, X]) Fx() fx.Effects {
	return t.fx
}

// Option is an option configuring a transaction.
type Option func(cfg *txConfig)

type txConfig struct {
	fx fx.Effects
}

// WithEffects configures the effects for the receipt.
func WithEffects(fx fx.Effects) Option {
	return func(cfg *txConfig) {
		cfg.fx = fx
	}
}

func NewTransaction[O, X any](result result.Result[O, X], options ...Option) Transaction[O, X] {
	cfg := txConfig{}
	for _, opt := range options {
		opt(&cfg)
	}
	return transaction[O, X]{out: result, fx: cfg.fx}
}

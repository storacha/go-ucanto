package transaction

import (
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/core/receipt"
	"github.com/storacha-network/go-ucanto/core/result"
)

// Transaction defines a result & effect pair, used by provider that wishes to
// return results that have effects.
type Transaction[O, X any] interface {
	Out() result.Result[O, X]
	Fx() receipt.Effects
}

type transaction[O, X any] struct {
	out result.Result[O, X]
	fx  receipt.Effects
}

func (t *transaction[O, X]) Out() result.Result[O, X] {
	return t.out
}

func (t *transaction[O, X]) Fx() receipt.Effects {
	return t.fx
}

var _ Transaction[any, any] = (*transaction[any, any])(nil)

type effects struct {
	fork []ipld.Link
	join ipld.Link
}

func (fx *effects) Fork() []ipld.Link {
	return fx.fork
}

func (fx *effects) Join() ipld.Link {
	return fx.join
}

var _ receipt.Effects = (*effects)(nil)

// Option is an option configuring a transaction.
type Option func(cfg *txConfig)

type txConfig struct {
	fork []ipld.Link
	join ipld.Link
}

// WithForks configures the forks for the receipt.
func WithForks(fork []ipld.Link) Option {
	return func(cfg *txConfig) {
		cfg.fork = fork
	}
}

// WithJoin configures the join for the receipt.
func WithJoin(join ipld.Link) Option {
	return func(cfg *txConfig) {
		cfg.join = join
	}
}

func NewTransaction[O, X any](result result.Result[O, X], options ...Option) Transaction[O, X] {
	cfg := txConfig{}
	for _, opt := range options {
		opt(&cfg)
	}

	fx := effects{}
	if len(cfg.fork) > 0 {
		fx.fork = cfg.fork
	}
	if cfg.join != nil {
		fx.join = cfg.join
	}

	return &transaction[O, X]{out: result, fx: &fx}
}

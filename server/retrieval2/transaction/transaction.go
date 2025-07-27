package transaction

import (
	"io"
	"net/http"

	"github.com/storacha/go-ucanto/core/receipt/fx"
	"github.com/storacha/go-ucanto/core/result"
)

// Transaction defines a result & effect pair, used by provider that wishes to
// return results that have effects.
type Transaction[O any, X any] struct {
	Out     result.Result[O, X]
	Fx      fx.Effects
	Status  int
	Headers http.Header
	Body    io.Reader
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
	fx      fx.Effects
	status  int
	headers http.Header
	body    io.Reader
}

// WithEffects configures the effects for the receipt.
func WithEffects(fx fx.Effects) Option {
	return func(cfg *txConfig) {
		cfg.fx = fx
	}
}

func WithResponseBody(body io.Reader) Option {
	return func(cfg *txConfig) {
		cfg.body = body
	}
}

func WithResponseStatus(status int) Option {
	return func(cfg *txConfig) {
		cfg.status = status
	}
}

func WithResponseHeaders(headers http.Header) Option {
	return func(cfg *txConfig) {
		cfg.headers = headers
	}
}

func NewTransaction[O, X any](result result.Result[O, X], options ...Option) Transaction[O, X] {
	cfg := txConfig{}
	for _, opt := range options {
		opt(&cfg)
	}
	return transaction[O, X]{out: result, fx: cfg.fx}
}

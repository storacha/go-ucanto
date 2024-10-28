package receipt

import (
	"slices"
	"testing"

	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/invocation/ran"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/receipt/fx"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/core/result/ok"
	"github.com/storacha/go-ucanto/testing/fixtures"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/stretchr/testify/require"
)

func TestEffects(t *testing.T) {
	ran := ran.FromLink(helpers.RandomCID())
	out := result.Ok[ok.Unit, ipld.Builder](ok.Unit{})

	t.Run("as links", func(t *testing.T) {
		f0 := fx.FromLink(helpers.RandomCID())
		f1 := fx.FromLink(helpers.RandomCID())
		j := fx.FromLink(helpers.RandomCID())

		receipt, err := Issue(fixtures.Alice, out, ran, WithFork(f0, f1), WithJoin(j))
		require.NoError(t, err)

		effects := receipt.Fx()
		require.True(t, slices.ContainsFunc(effects.Fork(), func(f fx.Effect) bool {
			return f.Link().String() == f0.Link().String()
		}))
		require.True(t, slices.ContainsFunc(effects.Fork(), func(f fx.Effect) bool {
			return f.Link().String() == f1.Link().String()
		}))
		require.Equal(t, effects.Join().Link(), j.Link())
	})

	t.Run("as invocations", func(t *testing.T) {
		i0, err := invocation.Invoke(
			fixtures.Alice,
			fixtures.Bob,
			ucan.NewCapability("fx/0", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		)
		require.NoError(t, err)
		i1, err := invocation.Invoke(
			fixtures.Alice,
			fixtures.Mallory,
			ucan.NewCapability("fx/1", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		)
		require.NoError(t, err)
		i2, err := invocation.Invoke(
			fixtures.Mallory,
			fixtures.Bob,
			ucan.NewCapability("fx/2", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		)
		require.NoError(t, err)

		f0 := fx.FromInvocation(i0)
		f1 := fx.FromInvocation(i1)
		j := fx.FromInvocation(i2)

		receipt, err := Issue(fixtures.Alice, out, ran, WithFork(f0, f1), WithJoin(j))
		require.NoError(t, err)

		effects := receipt.Fx()
		require.True(t, slices.ContainsFunc(effects.Fork(), func(f fx.Effect) bool {
			return f.Link().String() == f0.Link().String()
		}))
		require.True(t, slices.ContainsFunc(effects.Fork(), func(f fx.Effect) bool {
			return f.Link().String() == f1.Link().String()
		}))
		require.Equal(t, effects.Join().Link(), j.Link())

		for _, effect := range effects.Fork() {
			_, ok := effect.Invocation()
			require.True(t, ok)
		}

		_, ok := effects.Join().Invocation()
		require.True(t, ok)
	})
}

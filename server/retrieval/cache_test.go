package retrieval

import (
	"fmt"
	"testing"

	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/testing/fixtures"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/stretchr/testify/require"
)

func TestMemoryDelegationCache(t *testing.T) {
	dlg, err := delegation.Delegate(
		fixtures.Alice,
		fixtures.Alice,
		[]ucan.Capability[ucan.NoCaveats]{
			ucan.NewCapability(
				"test/cache",
				fixtures.Alice.DID().String(),
				ucan.NoCaveats{},
			),
		},
	)
	require.NoError(t, err)

	t.Run("put", func(t *testing.T) {
		cache, err := NewMemoryDelegationCache(5)
		require.NoError(t, err)
		err = cache.Put(t.Context(), dlg)
		require.NoError(t, err)
	})

	t.Run("get", func(t *testing.T) {
		cache, err := NewMemoryDelegationCache(5)
		require.NoError(t, err)

		err = cache.Put(t.Context(), dlg)
		require.NoError(t, err)

		cached, ok, err := cache.Get(t.Context(), dlg.Link())
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, dlg.Link().String(), cached.Link().String())
	})

	t.Run("miss", func(t *testing.T) {
		cache, err := NewMemoryDelegationCache(5)
		require.NoError(t, err)

		cached, ok, err := cache.Get(t.Context(), dlg.Link())
		require.NoError(t, err)
		require.False(t, ok)
		require.Nil(t, cached)
	})

	t.Run("uses default size if not specified", func(t *testing.T) {
		cache, err := NewMemoryDelegationCache(-1)
		require.NoError(t, err)

		for i := range MemoryDelegationCacheSize + 1 {
			dlg, err := delegation.Delegate(
				fixtures.Alice,
				fixtures.Alice,
				[]ucan.Capability[ucan.NoCaveats]{
					ucan.NewCapability(
						"test/cache",
						fixtures.Alice.DID().String(),
						ucan.NoCaveats{},
					),
				},
				delegation.WithNonce(fmt.Sprintf("%d", i)),
			)
			require.NoError(t, err)

			err = cache.Put(t.Context(), dlg)
			require.NoError(t, err)
		}

		require.Equal(t, MemoryDelegationCacheSize, cache.data.Len())
	})
}

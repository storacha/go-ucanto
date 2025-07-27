package retrieval2

import (
	"context"
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/ipld"
)

var MemoryDelegationCacheSize = 100

type MemoryDelegationCache struct {
	data *lru.Cache[string, delegation.Delegation]
}

func (m *MemoryDelegationCache) Get(ctx context.Context, root ipld.Link) (delegation.Delegation, bool, error) {
	d, ok := m.data.Get(root.String())
	return d, ok, nil
}

func (m *MemoryDelegationCache) Put(ctx context.Context, d delegation.Delegation) error {
	m.data.Add(d.Link().String(), d)
	return nil
}

var _ delegation.Store = (*MemoryDelegationCache)(nil)

// NewMemoryDelegationCache creates a new in memory LRU cache for delegations
// that implements [DelegationStore]. The size parameter controls the maximum
// number of delegations that can be cached. Pass a value less than 1 to use the
// default cache size [MemoryDelegationCacheSize].
func NewMemoryDelegationCache(size int) (*MemoryDelegationCache, error) {
	if size <= 0 {
		size = MemoryDelegationCacheSize
	}
	cache, err := lru.New[string, delegation.Delegation](size)
	if err != nil {
		return nil, fmt.Errorf("creating delegation LRU: %w", err)
	}
	return &MemoryDelegationCache{data: cache}, nil
}

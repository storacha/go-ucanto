package delegation

import (
	"context"

	"github.com/storacha/go-ucanto/core/ipld"
)

// Store is storage for delegations.
type Store interface {
	Put(ctx context.Context, delegation Delegation) error
	// Get a delegation by CID.
	Get(ctx context.Context, root ipld.Link) (Delegation, bool, error)
}

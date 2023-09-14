package ipld

import (
	"github.com/alanshaw/go-ucanto/core/iterable"
)

// View represents a materialized IPLD DAG View, which provides a generic
// traversal API. It is useful for encoding (potentially partial) IPLD DAGs
// into content archives (e.g. CARs).
type IPLDView interface {
	// Root is the root block of the IPLD DAG this is the view of. This is the
	// block from which all other blocks are linked directly or transitively.
	Root() Block
	// Blocks returns an iterator of all the IPLD blocks that are included in
	// this view.
	//
	// It is RECOMMENDED that implementations return blocks in bottom up order
	// (i.e. leaf blocks first, root block last).
	//
	// Iterator MUST include the root block otherwise it will lead encoders into
	// omitting it when encoding the view into a CAR archive.
	Blocks() iterable.Iterator[Block]
}

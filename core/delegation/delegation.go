package delegation

import (
	"fmt"
	"io"

	"github.com/alanshaw/go-ucanto/core/car"
	"github.com/alanshaw/go-ucanto/core/dag/blockstore"
	adm "github.com/alanshaw/go-ucanto/core/delegation/datamodel"
	"github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/alanshaw/go-ucanto/core/ipld/block"
	"github.com/alanshaw/go-ucanto/core/ipld/codec/cbor"
	"github.com/alanshaw/go-ucanto/core/ipld/hash/sha256"
	"github.com/alanshaw/go-ucanto/core/iterable"
	"github.com/alanshaw/go-ucanto/ucan"
)

// Delagation is a materialized view of a UCAN delegation, which can be encoded
// into a UCAN token and used as proof for an invocation or further delegations.
type Delegation interface {
	ipld.IPLDView
	// Link returns the IPLD link of the root block of the invocation.
	Link() ucan.Link
	// Archive writes the invocation to a Content Addressed aRchive (CAR).
	Archive() io.Reader
	Version() ucan.Version
}

type delegation struct {
	rt   ipld.Block
	blks blockstore.BlockReader
}

func (d *delegation) Root() ipld.Block {
	return d.rt
}

func (d *delegation) Link() ucan.Link {
	return d.rt.Link()
}

func (d *delegation) Blocks() iterable.Iterator[ipld.Block] {
	return d.blks.Iterator()
}

func (d *delegation) Archive() io.Reader {
	return Archive(d)
}

func (d *delegation) Version() ucan.Version {
	data, _ := ucan.Decode(d.rt.Bytes())
	return data.Version()
}

func NewDelegation(root ipld.Block, bs blockstore.BlockReader) Delegation {
	return &delegation{rt: root, blks: bs}
}

func NewDelegationView(root ipld.Link, bs blockstore.BlockReader) (Delegation, error) {
	blk, ok, err := bs.Get(root)
	if err != nil {
		return nil, fmt.Errorf("getting delegation root block: %s", err)
	}
	if !ok {
		return nil, fmt.Errorf("missing delegation root block: %s", root)
	}
	return NewDelegation(blk, bs), nil
}

func Archive(d Delegation) io.Reader {
	// We create a descriptor block to describe what this DAG represents
	variant, err := block.Encode(
		&adm.ArchiveModel{Ucan0_9_1: d.Link()},
		adm.Type(),
		cbor.Codec,
		sha256.Hasher,
	)
	if err != nil {
		reader, _ := io.Pipe()
		reader.CloseWithError(fmt.Errorf("hashing variant block bytes: %s", err))
		return reader
	}
	// Create a new reader that contains the new block as well as the others.
	blks, err := blockstore.NewBlockReader(
		blockstore.WithBlocks([]ipld.Block{variant}),
		blockstore.WithBlocksIterator(d.Blocks()),
	)
	if err != nil {
		reader, _ := io.Pipe()
		reader.CloseWithError(fmt.Errorf("creating new block reader: %s", err))
		return reader
	}
	return car.Encode([]ipld.Link{variant.Link()}, blks.Iterator())
}

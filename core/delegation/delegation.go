package delegation

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/web3-storage/go-ucanto/core/car"
	"github.com/web3-storage/go-ucanto/core/dag/blockstore"
	adm "github.com/web3-storage/go-ucanto/core/delegation/datamodel"
	"github.com/web3-storage/go-ucanto/core/ipld"
	"github.com/web3-storage/go-ucanto/core/ipld/block"
	"github.com/web3-storage/go-ucanto/core/ipld/codec/cbor"
	"github.com/web3-storage/go-ucanto/core/ipld/hash/sha256"
	"github.com/web3-storage/go-ucanto/core/iterable"
	"github.com/web3-storage/go-ucanto/ucan"
	"github.com/web3-storage/go-ucanto/ucan/crypto/signature"
	udm "github.com/web3-storage/go-ucanto/ucan/datamodel/ucan"
)

// Delagation is a materialized view of a UCAN delegation, which can be encoded
// into a UCAN token and used as proof for an invocation or further delegations.
type Delegation interface {
	ipld.View
	// Link returns the IPLD link of the root block of the delegation.
	Link() ucan.Link
	// Archive writes the delegation to a Content Addressed aRchive (CAR).
	Archive() io.Reader
	// Issuer is the signer of the UCAN.
	Issuer() ucan.Principal
	// Audience is the principal delegated to.
	Audience() ucan.Principal
	// Version is the spec version the UCAN conforms to.
	Version() ucan.Version
	// Capabilities are claimed abilities that can be performed on a resource.
	Capabilities() []ucan.Capability[any]
	// Expiration is the time in seconds since the Unix epoch that the UCAN
	// becomes invalid.
	Expiration() ucan.UTCUnixTimestamp
	// NotBefore is the time in seconds since the Unix epoch that the UCAN
	// becomes valid.
	NotBefore() ucan.UTCUnixTimestamp
	// Nonce is a randomly generated string to provide a unique UCAN.
	Nonce() ucan.Nonce
	// Facts are arbitrary facts and proofs of knowledge.
	Facts() []ucan.Fact
	// Proofs of delegation.
	Proofs() []ucan.Link
	// Signature of the UCAN issuer.
	Signature() signature.SignatureView
}

type delegation struct {
	rt   ipld.Block
	blks blockstore.BlockReader
	ucan ucan.UCANView
	once sync.Once
}

var _ Delegation = (*delegation)(nil)

func (d *delegation) data() ucan.UCANView {
	d.once.Do(func() {
		data := udm.UCANModel{}
		err := block.Decode(d.rt, &data, udm.Type(), cbor.Codec, sha256.Hasher)
		if err != nil {
			fmt.Printf("Error: decoding UCAN: %s\n", err)
		}
		d.ucan, err = ucan.NewUCANView(&data)
		if err != nil {
			fmt.Printf("Error: constructing UCAN view: %s\n", err)
		}
	})
	return d.ucan
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

func (d *delegation) Issuer() ucan.Principal {
	return d.data().Issuer()
}

func (d *delegation) Audience() ucan.Principal {
	return d.data().Audience()
}

func (d *delegation) Version() ucan.Version {
	return d.data().Version()
}

func (d *delegation) Capabilities() []ucan.Capability[any] {
	return d.data().Capabilities()
}

func (d *delegation) Expiration() ucan.UTCUnixTimestamp {
	return d.data().Expiration()
}

func (d *delegation) NotBefore() ucan.UTCUnixTimestamp {
	return d.data().NotBefore()
}

func (d *delegation) Nonce() ucan.Nonce {
	return d.data().Nonce()
}

func (d *delegation) Facts() []ucan.Fact {
	return d.data().Facts()
}

func (d *delegation) Proofs() []ucan.Link {
	return d.data().Proofs()
}

func (d *delegation) Signature() signature.SignatureView {
	return d.data().Signature()
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

func Extract(b []byte) (Delegation, error) {
	roots, blks, err := car.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("decoding CAR: %s", err)
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("missing root CID in delegation archive")
	}
	if len(roots) > 1 {
		return nil, fmt.Errorf("unexpected number of root CIDs in archive: %d", len(roots))
	}

	br, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(blks))
	if err != nil {
		return nil, fmt.Errorf("creating block reader: %s", err)
	}

	rt, ok, err := br.Get(roots[0])
	if err != nil {
		return nil, fmt.Errorf("getting root block: %s", err)
	}
	if !ok {
		return nil, fmt.Errorf("missing root block: %d", len(roots))
	}

	model := adm.ArchiveModel{}
	err = block.Decode(
		rt,
		&model,
		adm.Type(),
		cbor.Codec,
		sha256.Hasher,
	)
	if err != nil {
		return nil, fmt.Errorf("decoding root block: %s", err)
	}

	return NewDelegationView(model.Ucan0_9_1, br)
}

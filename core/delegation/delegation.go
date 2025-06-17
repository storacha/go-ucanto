package delegation

import (
	"bytes"
	"fmt"
	"io"
	"iter"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
	"github.com/storacha/go-ucanto/core/car"
	"github.com/storacha/go-ucanto/core/dag/blockstore"
	adm "github.com/storacha/go-ucanto/core/delegation/datamodel"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/ipld/block"
	"github.com/storacha/go-ucanto/core/ipld/codec/cbor"
	"github.com/storacha/go-ucanto/core/ipld/hash/sha256"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/storacha/go-ucanto/ucan/crypto/signature"
	udm "github.com/storacha/go-ucanto/ucan/datamodel/ucan"
)

// Delagation is a materialized view of a UCAN delegation, which can be encoded
// into a UCAN token and used as proof for an invocation or further delegations.
type Delegation interface {
	ipld.View
	ucan.UCAN
	// Data returns the UCAN view of the delegation.
	Data() ucan.View
	// Link returns the IPLD link of the root block of the delegation.
	Link() ucan.Link
	// Archive writes the delegation to a Content Addressed aRchive (CAR).
	Archive() io.Reader
	// Attach a block to the delegation DAG so it will be included in the block
	// iterator. You should only attach blocks that are referenced from
	// `Capabilities` or `Facts`.
	Attach(block block.Block) error
}

type delegation struct {
	rt       ipld.Block
	blks     blockstore.BlockReader
	atchblks blockstore.BlockStore
	ucan     ucan.View
}

var _ Delegation = (*delegation)(nil)

func (d *delegation) Data() ucan.View {
	return d.ucan
}

func (d *delegation) Root() ipld.Block {
	return d.rt
}

func (d *delegation) Link() ucan.Link {
	return d.rt.Link()
}

func (d *delegation) Blocks() iter.Seq2[ipld.Block, error] {
	return export(d.ucan, d.rt, d.blks, d.atchblks)
}

func (d *delegation) Archive() io.Reader {
	return Archive(d)
}

func (d *delegation) Issuer() ucan.Principal {
	return d.Data().Issuer()
}

func (d *delegation) Audience() ucan.Principal {
	return d.Data().Audience()
}

func (d *delegation) Version() ucan.Version {
	return d.Data().Version()
}

func (d *delegation) Capabilities() []ucan.Capability[any] {
	return d.Data().Capabilities()
}

func (d *delegation) Expiration() *ucan.UTCUnixTimestamp {
	return d.Data().Expiration()
}

func (d *delegation) NotBefore() ucan.UTCUnixTimestamp {
	return d.Data().NotBefore()
}

func (d *delegation) Nonce() ucan.Nonce {
	return d.Data().Nonce()
}

func (d *delegation) Facts() []ucan.Fact {
	return d.Data().Facts()
}

func (d *delegation) Proofs() []ucan.Link {
	return d.Data().Proofs()
}

func (d *delegation) Signature() signature.SignatureView {
	return d.Data().Signature()
}

func (d *delegation) Attach(b block.Block) error {
	return d.atchblks.Put(b)
}

func NewDelegation(root ipld.Block, bs blockstore.BlockReader) (Delegation, error) {
	ucan, err := decode(root)
	if err != nil {
		return nil, fmt.Errorf("decoding UCAN: %s", err)
	}
	attachments, err := blockstore.NewBlockStore()
	if err != nil {
		return nil, err
	}
	return &delegation{rt: root, ucan: ucan, blks: bs, atchblks: attachments}, nil
}

func NewDelegationView(root ipld.Link, bs blockstore.BlockReader) (Delegation, error) {
	blk, ok, err := bs.Get(root)
	if err != nil {
		return nil, fmt.Errorf("getting delegation root block: %s", err)
	}
	if !ok {
		return nil, fmt.Errorf("missing delegation root block: %s", root)
	}
	return NewDelegation(blk, bs)
}

// export the blocks that comprise the delegation, including all extra attached
// blocks.
func export(rt ucan.View, rtblk ipld.Block, blks blockstore.BlockReader, atchblks blockstore.BlockReader) iter.Seq2[ipld.Block, error] {
	return func(yield func(ipld.Block, error) bool) {
		for _, p := range rt.Proofs() {
			proofblk, ok, err := blks.Get(p)
			if err != nil {
				yield(nil, err)
				return
			}
			if !ok {
				continue
			}
			prf, err := decode(proofblk)
			if err != nil {
				yield(nil, err)
				return
			}
			for b, err := range export(prf, proofblk, blks, nil) {
				if !yield(b, err) {
					return
				}
				if err != nil {
					return
				}
			}
		}

		if atchblks != nil {
			for b, err := range atchblks.Iterator() {
				if !yield(b, err) {
					return
				}
				if err != nil {
					return
				}
			}
		}

		yield(rtblk, nil)
	}
}

func decode(root ipld.Block) (ucan.View, error) {
	data := udm.UCANModel{}
	err := block.Decode(root, &data, udm.Type(), cbor.Codec, sha256.Hasher)
	if err != nil {
		return nil, fmt.Errorf("decoding root block: %w", err)
	}
	ucan, err := ucan.NewUCAN(&data)
	if err != nil {
		return nil, fmt.Errorf("constructing UCAN view: %w", err)
	}
	return ucan, nil
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
	blks, err := blockstore.NewBlockStore(blockstore.WithBlocksIterator(d.Blocks()))
	if err != nil {
		reader, _ := io.Pipe()
		reader.CloseWithError(fmt.Errorf("creating new block reader: %s", err))
		return reader
	}
	err = blks.Put(variant)
	if err != nil {
		reader, _ := io.Pipe()
		reader.CloseWithError(fmt.Errorf("adding variant block: %s", err))
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

func Format(dlg Delegation) (string, error) {
	bytes, err := io.ReadAll(dlg.Archive())
	if err != nil {
		return "", fmt.Errorf("archiving delegation: %w", err)
	}
	digest, err := multihash.Sum(bytes, uint64(multicodec.Identity), -1)
	if err != nil {
		return "", fmt.Errorf("creating multihash: %w", err)
	}
	return cid.NewCidV1(uint64(multicodec.Car), digest).StringOfBase(multibase.Base64)
}

func Parse(input string) (Delegation, error) {
	cid, err := cid.Decode(input)
	if err != nil {
		return nil, fmt.Errorf("decoding CID: %w", err)
	}
	codec := multicodec.Code(cid.Prefix().Codec)
	if codec != multicodec.Car {
		return nil, fmt.Errorf("non CAR codec found: %s", codec.String())
	}
	mhinfo, err := multihash.Decode(cid.Hash())
	if err != nil {
		return nil, fmt.Errorf("decoding multihash: %w", err)
	}
	mhcodec := multicodec.Code(mhinfo.Code)
	if mhcodec != multicodec.Identity {
		return nil, fmt.Errorf("non identity multihash: %s", mhcodec.String())
	}
	return Extract(mhinfo.Digest)
}

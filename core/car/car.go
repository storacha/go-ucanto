package car

import (
	"bufio"
	"fmt"
	"io"
	"iter"

	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	ipldcar "github.com/ipld/go-car"
	"github.com/ipld/go-car/util"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-varint"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/ipld/block"
)

// ContentType is the value the HTTP Content-Type header should have for CARs.
// See https://www.iana.org/assignments/media-types/application/vnd.ipld.car
const ContentType = "application/vnd.ipld.car"

func Encode(roots []ipld.Link, blocks iter.Seq2[ipld.Block, error]) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		cids := []cid.Cid{}
		for _, r := range roots {
			_, cid, err := cid.CidFromBytes([]byte(r.Binary()))
			if err != nil {
				writer.CloseWithError(fmt.Errorf("decoding CAR root: %s: %w", r, err))
				return
			}
			cids = append(cids, cid)
		}
		h := ipldcar.CarHeader{
			Roots:   cids,
			Version: 1,
		}
		hb, err := cbor.DumpObject(h)
		if err != nil {
			writer.CloseWithError(fmt.Errorf("writing CAR header: %w", err))
			return
		}
		util.LdWrite(writer, hb)
		for block, err := range blocks {
			if err != nil {
				writer.CloseWithError(fmt.Errorf("writing CAR blocks: %w", err))
				return
			}
			err = util.LdWrite(writer, []byte(block.Link().Binary()), block.Bytes())
			if err != nil {
				writer.CloseWithError(fmt.Errorf("writing CAR blocks: %w", err))
				return
			}
		}
		writer.Close()
	}()
	return reader
}

type CarBlock interface {
	ipld.Block
	Offset() uint64
	Length() uint64
}

type carBlock struct {
	ipld.Block
	offset uint64
	length uint64
}

func (cb carBlock) Offset() uint64 {
	return cb.offset
}

func (cb carBlock) Length() uint64 {
	return cb.length
}

func Decode(reader io.Reader) ([]ipld.Link, iter.Seq2[ipld.Block, error], error) {
	br := bufio.NewReader(reader)

	h, err := ipldcar.ReadHeader(br)
	if err != nil {
		return nil, nil, err
	}

	if h.Version != 1 {
		return nil, nil, fmt.Errorf("invalid car version: %d", h.Version)
	}

	offset, err := ipldcar.HeaderSize(h)
	if err != nil {
		return nil, nil, err
	}

	var roots []ipld.Link
	for _, r := range h.Roots {
		roots = append(roots, cidlink.Link{Cid: r})
	}

	r := &blkReader{br, offset}
	return roots, func(yield func(ipld.Block, error) bool) {
		for {
			blk, err := r.next()
			if err == io.EOF {
				return
			}
			if !yield(blk, err) {
				return
			}
		}
	}, nil
}

type blkReader struct {
	br     *bufio.Reader
	offset uint64
}

func (r *blkReader) next() (CarBlock, error) {
	cid, bytes, err := util.ReadNode(r.br)
	if err != nil {
		return nil, err
	}

	hashed, err := cid.Prefix().Sum(bytes)
	if err != nil {
		return nil, err
	}

	if !hashed.Equals(cid) {
		return nil, fmt.Errorf("mismatch in content integrity, name: %s, data: %s", cid, hashed)
	}

	ss := uint64(cid.ByteLen()) + uint64(len(bytes))
	r.offset += uint64(varint.UvarintSize(ss)) + ss

	return carBlock{block.NewBlock(cidlink.Link{Cid: cid}, bytes), r.offset - uint64(len(bytes)), uint64(len(bytes))}, nil
}

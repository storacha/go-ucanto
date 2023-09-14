package car

import (
	"bufio"
	"fmt"
	"io"

	"github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/alanshaw/go-ucanto/core/iterable"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	ipldcar "github.com/ipld/go-car"
	"github.com/ipld/go-car/util"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

// ContentType is the value the HTTP Content-Type header should have for CARs.
// See https://www.iana.org/assignments/media-types/application/vnd.ipld.car
const ContentType = "application/vnd.ipld.car"

func Encode(roots []ipld.Link, blocks iterable.Iterator[ipld.Block]) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		var cids []cid.Cid
		for _, r := range roots {
			_, cid, err := cid.CidFromBytes([]byte(r.Binary()))
			if err != nil {
				writer.CloseWithError(fmt.Errorf("decoding CAR root: %s: %s", r, err))
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
			writer.CloseWithError(fmt.Errorf("writing CAR header: %s", err))
			return
		}
		util.LdWrite(writer, hb)
		for {
			block, err := blocks.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				writer.CloseWithError(fmt.Errorf("writing CAR blocks: %s", err))
				return
			}
			util.LdWrite(writer, []byte(block.Link().Binary()), block.Bytes())
		}
		writer.Close()
	}()
	return reader
}

func Decode(reader io.Reader) ([]ipld.Link, iterable.Iterator[ipld.Block], error) {
	br := bufio.NewReader(reader)

	h, err := ipldcar.ReadHeader(br)
	if err != nil {
		return nil, nil, err
	}

	if h.Version != 1 {
		return nil, nil, fmt.Errorf("invalid car version: %d", h.Version)
	}

	var roots []ipld.Link
	for _, r := range h.Roots {
		roots = append(roots, cidlink.Link{Cid: r})
	}

	return roots, iterable.NewIterator(func() (ipld.Block, error) {
		cid, bytes, err := util.ReadNode(br)
		if err != nil {
			if err == io.EOF {
				br = nil
			}
			return nil, err
		}

		hashed, err := cid.Prefix().Sum(bytes)
		if err != nil {
			return nil, err
		}

		if !hashed.Equals(cid) {
			return nil, fmt.Errorf("mismatch in content integrity, name: %s, data: %s", cid, hashed)
		}

		return ipld.NewBlockUnsafe(cidlink.Link{Cid: cid}, bytes), nil
	}), nil
}

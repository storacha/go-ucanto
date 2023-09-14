package dag

import (
	"fmt"
	"io"

	"github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/alanshaw/go-ucanto/core/iterable"
)

type BlockStore interface {
	Get(link ipld.Link) (ipld.Block, bool)
	Iterator() iterable.Iterator[ipld.Block]
}

type blockstore struct {
	keys []string
	blks map[string]ipld.Block
}

func (bs *blockstore) Get(link ipld.Link) (ipld.Block, bool) {
	b, ok := bs.blks[link.String()]
	return b, ok
}

func (bs *blockstore) Iterator() iterable.Iterator[ipld.Block] {
	i := 0
	return iterable.NewIterator(func() (ipld.Block, error) {
		if len(bs.keys) <= i {
			return nil, io.EOF
		}
		k := bs.keys[i]
		v, ok := bs.blks[k]
		if !ok {
			return nil, fmt.Errorf("missing block for key: %s", k)
		}
		i++
		return v, nil
	})
}

func NewBlockStore(blocks iterable.Iterator[ipld.Block]) (BlockStore, error) {
	keys := []string{}
	blks := map[string]ipld.Block{}
	for {
		b, err := blocks.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		blks[b.Link().String()] = b
		keys = append(keys, b.Link().String())
	}
	return &blockstore{keys, blks}, nil
}

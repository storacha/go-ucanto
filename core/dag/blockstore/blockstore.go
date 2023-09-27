package blockstore

import (
	"fmt"
	"io"
	"sync"

	"github.com/alanshaw/go-ucanto/core/ipld"
	"github.com/alanshaw/go-ucanto/core/iterable"
)

type BlockReader interface {
	Get(link ipld.Link) (ipld.Block, bool, error)
	Iterator() iterable.Iterator[ipld.Block]
}

type BlockWriter interface {
	Put(block ipld.Block) error
}

type BlockStore interface {
	BlockReader
	BlockWriter
}

type blockreader struct {
	keys []string
	blks map[string]ipld.Block
}

func (br *blockreader) Get(link ipld.Link) (ipld.Block, bool, error) {
	b, ok := br.blks[link.String()]
	return b, ok, nil
}

func (br *blockreader) Iterator() iterable.Iterator[ipld.Block] {
	i := 0
	return iterable.NewIterator(func() (ipld.Block, error) {
		if len(br.keys) <= i {
			return nil, io.EOF
		}
		k := br.keys[i]
		v, ok := br.blks[k]
		if !ok {
			return nil, fmt.Errorf("missing block for key: %s", k)
		}
		i++
		return v, nil
	})
}

type blockstore struct {
	sync.RWMutex
	blockreader
}

func (bs *blockstore) Put(block ipld.Block) error {
	bs.RLock()
	defer bs.RUnlock()

	_, ok := bs.blks[block.Link().String()]
	if ok {
		return nil
	}

	bs.blks[block.Link().String()] = block
	bs.keys = append(bs.keys, block.Link().String())

	return nil
}

func (bs *blockstore) Get(link ipld.Link) (ipld.Block, bool, error) {
	bs.Lock()
	defer bs.Unlock()
	return bs.blockreader.Get(link)
}

func (bs *blockstore) Iterator() iterable.Iterator[ipld.Block] {
	bs.Lock()
	defer bs.Unlock()
	keys := bs.keys[:]
	i := 0
	return iterable.NewIterator(func() (ipld.Block, error) {
		if len(keys) <= i {
			return nil, io.EOF
		}
		k := keys[i]
		v, ok := bs.blks[k]
		if !ok {
			return nil, fmt.Errorf("missing block for key: %s", k)
		}
		i++
		return v, nil
	})
}

// Option is an option configuring a block reader/writer.
type Option func(cfg *bsConfig) error

type bsConfig struct {
	blks     []ipld.Block
	blksiter iterable.Iterator[ipld.Block]
}

// WithBlocks configures the blocks the blockstore should contain.
func WithBlocks(blks []ipld.Block) Option {
	return func(cfg *bsConfig) error {
		cfg.blks = blks
		return nil
	}
}

// WithBlocksIterator configures the blocks the blockstore should contain.
func WithBlocksIterator(blks iterable.Iterator[ipld.Block]) Option {
	return func(cfg *bsConfig) error {
		cfg.blksiter = blks
		return nil
	}
}

func NewBlockStore(options ...Option) (BlockStore, error) {
	cfg := bsConfig{}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}
	bs := &blockstore{}
	for _, b := range cfg.blks {
		err := bs.Put(b)
		if err != nil {
			return nil, err
		}
	}
	if cfg.blksiter != nil {
		for {
			b, err := cfg.blksiter.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			err = bs.Put(b)
			if err != nil {
				return nil, err
			}
		}
	}
	return bs, nil
}

func NewBlockReader(options ...Option) (BlockReader, error) {
	cfg := bsConfig{}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	keys := []string{}
	blks := map[string]ipld.Block{}

	for _, b := range cfg.blks {
		_, ok := blks[b.Link().String()]
		if ok {
			continue
		}
		blks[b.Link().String()] = b
		keys = append(keys, b.Link().String())
	}
	if cfg.blksiter != nil {
		for {
			b, err := cfg.blksiter.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			_, ok := blks[b.Link().String()]
			if ok {
				continue
			}
			blks[b.Link().String()] = b
			keys = append(keys, b.Link().String())
		}
	}

	return &blockreader{keys, blks}, nil
}

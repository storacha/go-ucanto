package blockstore

import (
	"fmt"
	"iter"
	"sync"

	"github.com/storacha/go-ucanto/core/ipld"
)

type BlockReader interface {
	Get(link ipld.Link) (ipld.Block, bool, error)
	Iterator() iter.Seq2[ipld.Block, error]
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

func (br *blockreader) Iterator() iter.Seq2[ipld.Block, error] {
	return func(yield func(ipld.Block, error) bool) {
		for _, k := range br.keys {
			v, ok := br.blks[k]
			var err error
			if !ok {
				err = fmt.Errorf("missing block for key: %s", k)
			}
			if !yield(v, err) {
				return
			}
		}
	}
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

func (bs *blockstore) Iterator() iter.Seq2[ipld.Block, error] {
	bs.Lock()
	defer bs.Unlock()
	return func(yield func(ipld.Block, error) bool) {
		for _, k := range bs.keys {
			v, ok := bs.blks[k]
			var err error
			if !ok {
				err = fmt.Errorf("missing block for key: %s", k)
			}
			if !yield(v, err) {
				return
			}
		}
	}
}

// Option is an option configuring a block reader/writer.
type Option func(cfg *bsConfig) error

type bsConfig struct {
	blks     []ipld.Block
	blksiter iter.Seq2[ipld.Block, error]
}

// WithBlocks configures the blocks the blockstore should contain.
func WithBlocks(blks []ipld.Block) Option {
	return func(cfg *bsConfig) error {
		cfg.blks = blks
		return nil
	}
}

// WithBlocksIterator configures the blocks the blockstore should contain.
func WithBlocksIterator(blks iter.Seq2[ipld.Block, error]) Option {
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
	bs := &blockstore{
		blockreader: blockreader{
			keys: []string{},
			blks: map[string]ipld.Block{},
		},
	}
	for _, b := range cfg.blks {
		err := bs.Put(b)
		if err != nil {
			return nil, err
		}
	}
	if cfg.blksiter != nil {
		for b, err := range cfg.blksiter {
			if err != nil {
				return nil, err
			}
			err := bs.Put(b)
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
		for b, err := range cfg.blksiter {
			if err != nil {
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

func WriteInto(view ipld.View, bs BlockWriter) error {
	for b := range view.Blocks() {
		err := bs.Put(b)
		if err != nil {
			return fmt.Errorf("putting proof block: %s", err)
		}
	}
	return nil
}

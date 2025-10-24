package datamodel

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed archive.ipldsch
var archive []byte

var (
	once sync.Once
	ts   *schema.TypeSystem
	err  error
)

func mustLoadSchema() *schema.TypeSystem {
	once.Do(func() {
		ts, err = ipld.LoadSchemaBytes(archive)
	})
	if err != nil {
		panic(fmt.Errorf("failed to load IPLD schema: %w", err))
	}
	return ts
}

func ArchiveType() schema.Type {
	return mustLoadSchema().TypeByName("Archive")
}

type ArchiveModel struct {
	UcanReceipt0_9_1 ipld.Link
}

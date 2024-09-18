package datamodel_test

import (
	"testing"

	"github.com/ipfs/go-cid"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	adm "github.com/storacha/go-ucanto/core/delegation/datamodel"
	"github.com/storacha/go-ucanto/core/ipld/block"
	"github.com/storacha/go-ucanto/core/ipld/codec/cbor"
	"github.com/storacha/go-ucanto/core/ipld/hash/sha256"
)

func TestEncodeDecode(t *testing.T) {
	l := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
	m0 := adm.ArchiveModel{
		Ucan0_9_1: l,
	}
	mblk, err := block.Encode(&m0, adm.Type(), cbor.Codec, sha256.Hasher)
	if err != nil {
		t.Fatalf("encoding archive model: %s", err)
	}

	m1 := adm.ArchiveModel{}
	err = block.Decode(mblk, &m1, adm.Type(), cbor.Codec, sha256.Hasher)
	if err != nil {
		t.Fatalf("decoding agent message: %s", err)
	}

	d1 := m1.Ucan0_9_1
	if d1.String() != l.String() {
		t.Fatalf("failed round trip link")
	}
}

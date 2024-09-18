package datamodel_test

import (
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/storacha/go-ucanto/core/ipld/block"
	"github.com/storacha/go-ucanto/core/ipld/codec/cbor"
	"github.com/storacha/go-ucanto/core/ipld/hash/sha256"
	mdm "github.com/storacha/go-ucanto/core/message/datamodel"
)

func TestEncodeDecode(t *testing.T) {
	l := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
	ex := []ipld.Link{l}
	rp := mdm.ReportModel{
		Keys:   []string{l.String()},
		Values: map[string]ipld.Link{l.String(): l},
	}
	d0 := mdm.DataModel{Execute: ex, Report: &rp}
	m0 := mdm.AgentMessageModel{UcantoMessage7: &d0}
	mblk, err := block.Encode(&m0, mdm.Type(), cbor.Codec, sha256.Hasher)
	if err != nil {
		t.Fatalf("encoding agent message: %s", err)
	}

	m1 := mdm.AgentMessageModel{}
	err = block.Decode(mblk, &m1, mdm.Type(), cbor.Codec, sha256.Hasher)
	if err != nil {
		t.Fatalf("decoding agent message: %s", err)
	}

	d1 := m1.UcantoMessage7
	if d1.Execute[0] != l || d1.Execute[0] != d0.Execute[0] {
		t.Fatalf("failed round trip execute field")
	}
	if d1.Report.Values[l.String()] != l || d1.Report.Values[l.String()] != d0.Report.Values[l.String()] {
		t.Fatalf("failed round trip report field")
	}
}

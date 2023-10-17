package receipt_test

import (
	"testing"

	"github.com/alanshaw/go-ucanto/core/receipt/schema/receipt"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

func TestEncodeDecode(t *testing.T) {
	l := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
	ex := []ipld.Link{l}
	meta := receipt.MetaMap{}
	r0 := receipt.Receipt{
		Ocm: &receipt.Ocm{
			Ran: l,
			Out
		},
		Sig: []byte{},
	}
	bytes, err := agentmessage.Encode(&d0)
	if err != nil {
		t.Fatalf("encoding agent message: %s", err)
	}
	d1, err := agentmessage.Decode(bytes)
	if err != nil {
		t.Fatalf("decoding agent message: %s", err)
	}
	if d1.Execute[0] != l || d1.Execute[0] != d0.Execute[0] {
		t.Fatalf("failed round trip execute field")
	}
	if d1.Report.Values[l.String()] != l || d1.Report.Values[l.String()] != d0.Report.Values[l.String()] {
		t.Fatalf("failed round trip report field")
	}
}

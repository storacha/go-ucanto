package datamodel_test

import (
	"testing"

	"github.com/alanshaw/go-ucanto/core/ipld/block"
	"github.com/alanshaw/go-ucanto/core/ipld/codec/cbor"
	"github.com/alanshaw/go-ucanto/core/ipld/hash/sha256"
	rdm "github.com/alanshaw/go-ucanto/core/receipt/datamodel"
	"github.com/ipfs/go-cid"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

type resultOk struct {
	Status string
}

type resultErr struct {
	Message string
}

func TestEncodeDecode(t *testing.T) {
	typ, err := rdm.NewReceiptModelType([]byte(`
		type Result union {
			| Ok "ok"
			| Err "error"
		} representation keyed

		type Ok struct {
			status String (rename "Status")
		}

		type Err struct {
			message String (rename "Message")
		}
	`))
	if err != nil {
		t.Fatalf("loading result schema: %s", err)
	}

	l := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
	r0 := rdm.ReceiptModel[*resultOk, *resultErr]{
		Ocm: rdm.OutcomeModel[*resultOk, *resultErr]{
			Ran: l,
			Out: &rdm.ResultModel[*resultOk, *resultErr]{
				Ok: &resultOk{Status: "done"},
			},
		},
	}
	b0, err := block.Encode(&r0, typ, cbor.Codec, sha256.Hasher)
	if err != nil {
		t.Fatalf("encoding receipt: %s", err)
	}
	r1 := rdm.ReceiptModel[*resultOk, *resultErr]{}
	err = block.Decode(b0, &r1, typ, cbor.Codec, sha256.Hasher)
	if err != nil {
		t.Fatalf("decoding receipt: %s", err)
	}
	if r1.Ocm.Out.Err != nil {
		t.Fatalf("result err was not nil")
	}
	if r1.Ocm.Out.Ok.Status != "done" {
		t.Fatalf("status was not done")
	}

	r2 := rdm.ReceiptModel[*resultOk, *resultErr]{
		Ocm: rdm.OutcomeModel[*resultOk, *resultErr]{
			Ran: l,
			Out: &rdm.ResultModel[*resultOk, *resultErr]{
				Err: &resultErr{Message: "boom"},
			},
		},
	}
	b1, err := block.Encode(&r2, typ, cbor.Codec, sha256.Hasher)
	if err != nil {
		t.Fatalf("encoding receipt: %s", err)
	}
	r3 := rdm.ReceiptModel[*resultOk, *resultErr]{}
	err = block.Decode(b1, &r3, typ, cbor.Codec, sha256.Hasher)
	if err != nil {
		t.Fatalf("decoding receipt: %s", err)
	}
	if r3.Ocm.Out.Ok != nil {
		t.Fatalf("result ok was not nil")
	}
	if r3.Ocm.Out.Err.Message != "boom" {
		t.Fatalf("error message was not boom")
	}
}

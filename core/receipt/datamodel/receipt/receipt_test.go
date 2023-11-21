package receipt_test

import (
	"testing"

	"github.com/alanshaw/go-ucanto/core/receipt/datamodel/receipt"
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
	typ, err := receipt.NewReceiptType([]byte(`
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
	r0 := receipt.Receipt[*resultOk, *resultErr]{
		Ocm: receipt.Ocm[*resultOk, *resultErr]{
			Ran: l,
			Out: &receipt.Result[*resultOk, *resultErr]{
				Ok: &resultOk{Status: "done"},
			},
		},
	}
	b0, err := receipt.Encode(&r0, typ)
	if err != nil {
		t.Fatalf("encoding receipt: %s", err)
	}
	r1, err := receipt.Decode[*resultOk, *resultErr](b0, typ)
	if err != nil {
		t.Fatalf("decoding receipt: %s", err)
	}
	if r1.Ocm.Out.Err != nil {
		t.Fatalf("result err was not nil")
	}
	if r1.Ocm.Out.Ok.Status != "done" {
		t.Fatalf("status was not done")
	}

	r2 := receipt.Receipt[*resultOk, *resultErr]{
		Ocm: receipt.Ocm[*resultOk, *resultErr]{
			Ran: l,
			Out: &receipt.Result[*resultOk, *resultErr]{
				Err: &resultErr{Message: "boom"},
			},
		},
	}
	b1, err := receipt.Encode(&r2, typ)
	if err != nil {
		t.Fatalf("encoding receipt: %s", err)
	}
	r3, err := receipt.Decode[*resultOk, *resultErr](b1, typ)
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

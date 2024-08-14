package ipld

import (
	"fmt"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/storacha-network/go-ucanto/core/ipld/codec/cbor"
)

func TestRebind(t *testing.T) {
	type Target struct {
		Abool    bool
		Aint     int
		Afloat   float64
		Astring  string
		Abytes   []byte
		Aenumstr string
		Aenumint int
		Alist    []string
		Amap     struct {
			Keys   []string
			Values map[string]int
		}
		Alink        ipld.Link
		Aoptionalstr *string
		Anullablestr *string
	}
	type Origin struct {
		Value Target
	}
	type Base struct {
		Value datamodel.Node
	}

	ts, err := ipld.LoadSchemaBytes([]byte(`
		type Origin struct {
			value Target
		}
		type Base struct {
			value Any
		}
		type Target struct {
			abool Bool
			aint Int
			afloat Float
			astring String
			abytes Bytes
			aenumstr EnumString
			aenumint EnumInt
			alist [String]
			amap { String: Int }
			alink Link
			aoptionalstr optional String
			anullablestr nullable String
		}
		type EnumString enum {
			| Nope
			| Yep
			| Maybe
		}
		type EnumInt enum {
			| Nope  ("0")
			| Yep   ("1")
			| Maybe ("100")
		} representation int
	`))
	if err != nil {
		t.Fatalf("failed to load schema: %s", err)
	}
	otyp := ts.TypeByName("Origin")
	btyp := ts.TypeByName("Base")
	ttyp := ts.TypeByName("Target")

	str := "foo"
	origin := Origin{
		Value: Target{
			Abool:    true,
			Aint:     138,
			Afloat:   1.138,
			Astring:  "foo",
			Abytes:   []byte{1, 2, 3},
			Aenumstr: "Yep",
			Aenumint: 100,
			Alist:    []string{"bar"},
			Amap: struct {
				Keys   []string
				Values map[string]int
			}{
				Keys:   []string{"foo"},
				Values: map[string]int{"foo": 1138},
			},
			Alink:        cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")},
			Aoptionalstr: &str,
			Anullablestr: &str,
		},
	}
	b, err := cbor.Encode(&origin, otyp)
	if err != nil {
		t.Fatalf("encoding origin: %s", err)
	}

	var bind Base
	err = cbor.Decode(b, &bind, btyp)
	if err != nil {
		t.Fatalf("decoding base: %s", err)
	}

	var sub Target
	_, err = Rebind(bind.Value, &sub, ttyp)
	if err != nil {
		t.Fatalf("binding subtype: %s", err)
	}

	fmt.Printf("%+v\n", sub)

	if sub.Astring != "foo" {
		t.Fatalf("failed round trip")
	}
	if sub.Aint != 138 {
		t.Fatalf("failed round trip")
	}
}

func TestRebindNonCompatibleStruct(t *testing.T) {
	type Target struct {
		Astring string
		Aint    int
	}
	type TargetIncompatible struct {
		Abool bool
	}
	type Origin struct {
		Value Target
	}
	type Base struct {
		Value datamodel.Node
	}

	ts, err := ipld.LoadSchemaBytes([]byte(`
	  type Origin struct {
		  value Target
		}
		type Base struct {
			value Any
		}
		type Target struct {
			astring String
			aint Int
		}
	`))
	if err != nil {
		t.Fatalf("failed to load schema: %s", err)
	}
	otyp := ts.TypeByName("Origin")
	btyp := ts.TypeByName("Base")
	ttyp := ts.TypeByName("Target")

	origin := Origin{Value: Target{Astring: "foo", Aint: 138}}
	b, err := cbor.Encode(&origin, otyp)
	if err != nil {
		t.Fatalf("encoding origin: %s", err)
	}

	var bind Base
	err = cbor.Decode(b, &bind, btyp)
	if err != nil {
		t.Fatalf("decoding base: %s", err)
	}

	var sub TargetIncompatible
	_, err = Rebind(bind.Value, &sub, ttyp)
	if err == nil {
		t.Fatalf("expected error rebinding")
	}
	fmt.Println(err)
}

func TestRebindNonCompatibleSchema(t *testing.T) {
	type Target struct {
		Astring string
		Aint    int
	}
	type Origin struct {
		Value Target
	}
	type Base struct {
		Value datamodel.Node
	}

	ts, err := ipld.LoadSchemaBytes([]byte(`
	  type Origin struct {
		  value Target
		}
		type Base struct {
			value Any
		}
		type Target struct {
			astring String
			aint Int
		}
		type TargetIncompatible struct {
			abool Bool
		}
	`))
	if err != nil {
		t.Fatalf("failed to load schema: %s", err)
	}
	otyp := ts.TypeByName("Origin")
	btyp := ts.TypeByName("Base")
	ttyp := ts.TypeByName("TargetIncompatible")

	origin := Origin{Value: Target{Astring: "foo", Aint: 138}}
	b, err := cbor.Encode(&origin, otyp)
	if err != nil {
		t.Fatalf("encoding origin: %s", err)
	}

	var bind Base
	err = cbor.Decode(b, &bind, btyp)
	if err != nil {
		t.Fatalf("decoding base: %s", err)
	}

	var sub Target
	_, err = Rebind(bind.Value, &sub, ttyp)
	if err == nil {
		t.Fatalf("expected error rebinding")
	}
	fmt.Println(err)
}

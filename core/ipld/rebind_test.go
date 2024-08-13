package ipld

import (
	"fmt"
	"testing"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/storacha-network/go-ucanto/core/ipld/codec/cbor"
)

func TestRebindInt(t *testing.T) {
	type Origin struct {
		Value int
	}
	type Base struct {
		Value datamodel.Node
	}

	ts, err := ipld.LoadSchemaBytes([]byte(`
	  type Origin struct {
		  value Int
		}
		type Base struct {
			value Any
		}
	`))
	if err != nil {
		t.Fatalf("failed to load schema: %s", err)
	}
	otyp := ts.TypeByName("Origin")
	btyp := ts.TypeByName("Base")
	ttyp := ts.TypeByName("Int")

	origin := Origin{Value: 5}
	b, err := cbor.Encode(&origin, otyp)
	if err != nil {
		t.Fatalf("encoding origin: %s", err)
	}

	var bind Base
	err = cbor.Decode(b, &bind, btyp)
	if err != nil {
		t.Fatalf("decoding base: %s", err)
	}

	var sub int
	_, err = Rebind(bind.Value, &sub, ttyp)
	if err != nil {
		t.Fatalf("binding subtype: %s", err)
	}

	fmt.Println(sub)

	if sub != 5 {
		t.Fatalf("failed round trip")
	}
}

func TestRebindStruct(t *testing.T) {
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

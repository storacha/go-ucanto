package did

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDIDKey(t *testing.T) {
	str := "did:key:z6Mkod5Jr3yd5SC7UDueqK4dAAw5xYJYjksy722tA9Boxc4z"
	d, err := Parse(str)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if d.String() != str {
		t.Fatalf("expected %v to equal %v", d.String(), str)
	}
}

func TestDecodeDIDKey(t *testing.T) {
	str := "did:key:z6Mkod5Jr3yd5SC7UDueqK4dAAw5xYJYjksy722tA9Boxc4z"
	d0, err := Parse(str)
	if err != nil {
		t.Fatalf("%v", err)
	}
	d1, err := Decode(d0.Bytes())
	if err != nil {
		t.Fatalf("%v", err)
	}
	if d1.String() != str {
		t.Fatalf("expected %v to equal %v", d1.String(), str)
	}
}

func TestParseDIDWeb(t *testing.T) {
	str := "did:web:up.web3.storage"
	d, err := Parse(str)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if d.String() != str {
		t.Fatalf("expected %v to equal %v", d.String(), str)
	}
}

func TestDecodeDIDWeb(t *testing.T) {
	str := "did:web:up.web3.storage"
	d0, err := Parse(str)
	if err != nil {
		t.Fatalf("%v", err)
	}
	d1, err := Decode(d0.Bytes())
	if err != nil {
		t.Fatalf("%v", err)
	}
	if d1.String() != str {
		t.Fatalf("expected %v to equal %v", d1.String(), str)
	}
}

func TestEquivalence(t *testing.T) {
	u0 := DID{}
	u1 := Undef
	if u0 != u1 {
		t.Fatalf("undef DID not equivalent")
	}

	d0, err := Parse("did:key:z6Mkod5Jr3yd5SC7UDueqK4dAAw5xYJYjksy722tA9Boxc4z")
	if err != nil {
		t.Fatalf("%v", err)
	}

	d1, err := Parse("did:key:z6Mkod5Jr3yd5SC7UDueqK4dAAw5xYJYjksy722tA9Boxc4z")
	if err != nil {
		t.Fatalf("%v", err)
	}

	if d0 != d1 {
		t.Fatalf("two equivalent DID not equivalent")
	}
}

func TestRoundtripJSON(t *testing.T) {
	id, err := Parse("did:key:z6Mkod5Jr3yd5SC7UDueqK4dAAw5xYJYjksy722tA9Boxc4z")
	require.NoError(t, err)

	type Object struct {
		ID                DID  `json:"id"`
		UndefID           DID  `json:"undef_id"`
		OptionalPresentID *DID `json:"optional_present_id"`
		OptionalAbsentID  *DID `json:"optional_absent_id"`
	}
	obj := Object{
		ID:                id,
		UndefID:           Undef,
		OptionalPresentID: &id,
		OptionalAbsentID:  nil,
	}

	data, err := json.Marshal(obj)
	require.NoError(t, err)

	t.Log(string(data))

	var out Object
	err = json.Unmarshal(data, &out)
	require.NoError(t, err)

	require.Equal(t, obj.ID, out.ID)
	require.Equal(t, obj.UndefID, out.UndefID)
	require.Equal(t, obj.OptionalPresentID.String(), out.OptionalPresentID.String())
	require.Nil(t, out.OptionalAbsentID)
}

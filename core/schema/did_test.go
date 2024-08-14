package schema

import (
	"testing"

	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/did"
	"github.com/stretchr/testify/require"
)

func TestReadDID(t *testing.T) {
	res := DID().Read("notadid")
	result.MatchResultR0(res, func(ok did.DID) {
		t.Fatalf("unexpectedly parsed a non-DID as a DID: %s", ok.String())
	}, func(err result.Failure) {
		require.Equal(t, err.Name(), "SchemaError")
	})

	res = DID().Read("did:key:z6Mkod5Jr3yd5SC7UDueqK4dAAw5xYJYjksy722tA9Boxc4z")
	result.MatchResultR0(res, func(ok did.DID) {
		require.Equal(t, ok.DID().String(), "did:key:z6Mkod5Jr3yd5SC7UDueqK4dAAw5xYJYjksy722tA9Boxc4z")
	}, func(err result.Failure) {
		t.Fatalf("unexpected error reading DID: %s", err)
	})
}

func TestReadDIDString(t *testing.T) {
	res := DIDString().Read("did:key:z6Mkod5Jr3yd5SC7UDueqK4dAAw5xYJYjksy722tA9Boxc4z")
	result.MatchResultR0(res, func(ok string) {
		require.Equal(t, ok, "did:key:z6Mkod5Jr3yd5SC7UDueqK4dAAw5xYJYjksy722tA9Boxc4z")
	}, func(err result.Failure) {
		t.Fatalf("unexpected error reading DID: %s", err)
	})
}

package schema

import (
	"testing"

	"github.com/storacha-network/go-ucanto/did"
	"github.com/stretchr/testify/require"
)

func TestReadDID(t *testing.T) {
	res, err := DID().Read("notadid")
	require.Error(t, err)
	require.Equal(t, res, did.Undef)
	require.Equal(t, err.Name(), "SchemaError")

	res, err = DID().Read("did:key:z6Mkod5Jr3yd5SC7UDueqK4dAAw5xYJYjksy722tA9Boxc4z")
	require.NoError(t, err)
	require.Equal(t, res.String(), "did:key:z6Mkod5Jr3yd5SC7UDueqK4dAAw5xYJYjksy722tA9Boxc4z")
}

func TestReadDIDString(t *testing.T) {
	res, err := DIDString().Read("did:key:z6Mkod5Jr3yd5SC7UDueqK4dAAw5xYJYjksy722tA9Boxc4z")
	require.NoError(t, err)
	require.Equal(t, res, "did:key:z6Mkod5Jr3yd5SC7UDueqK4dAAw5xYJYjksy722tA9Boxc4z")
}

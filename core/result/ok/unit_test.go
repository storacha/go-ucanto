package ok

import (
	"testing"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/storacha/go-ucanto/core/ipld/codec/cbor"
	udm "github.com/storacha/go-ucanto/core/result/ok/datamodel"
	"github.com/stretchr/testify/require"
)

func TestUnit(t *testing.T) {
	u := Unit{}
	nd, err := u.ToIPLD()
	require.NoError(t, err)

	// should be represented as a map
	require.Equal(t, nd.Kind(), datamodel.Kind_Map)

	// should contain no items
	it := nd.MapIterator()
	require.True(t, it.Done())

	bytes, err := cbor.Encode(&u, udm.UnitType())
	require.NoError(t, err)

	u2 := Unit{}
	err = cbor.Decode(bytes, &u2, udm.UnitType())
	require.NoError(t, err)

	require.Equal(t, u, u2)
}

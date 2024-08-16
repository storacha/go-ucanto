package multiformat

import (
	"testing"

	"github.com/storacha-network/go-ucanto/testing/helpers"
	"github.com/stretchr/testify/require"
)

func TestTag(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		b := []byte{1, 2, 3}
		tb := TagWith(1, b)
		utb := helpers.Must(UntagWith(1, tb, 0))
		require.EqualValues(t, b, utb)
	})

	t.Run("incorrect tag", func(t *testing.T) {
		b := []byte{1, 2, 3}
		tb := TagWith(1, b)
		_, err := UntagWith(2, tb, 0)
		require.Error(t, err)
		require.Equal(t, "expected multiformat with 0x2 tag instead got 0x1", err.Error())
	})
}

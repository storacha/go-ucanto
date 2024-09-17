package signature

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSignature(t *testing.T) {
	t.Run("roundtrip", func(t *testing.T) {
		raw, err := CodeName(EdDSA)
		require.NoError(t, err)

		s := NewSignature(EdDSA, []byte(raw))
		d := Decode(Encode(s))
		require.Equal(t, EdDSA, int(d.Code()))
		require.Equal(t, raw, string(d.Raw()))
	})
}

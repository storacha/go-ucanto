package verifier

import (
	"crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	str := "did:key:z6MkgZN5cRgWqesJeaZCEs7eKzyQsfpzmhnSEqTL6FZt56Ym"
	v, err := Parse(str)
	if err != nil {
		t.Fatalf("parsing DID: %s", err)
	}
	if v.DID().String() != str {
		t.Fatalf("expected %s to equal %s", v.DID().String(), str)
	}
}

func TestFromRaw(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		pub, _, err := ed25519.GenerateKey(nil)
		require.NoError(t, err)

		v, err := FromRaw(pub)
		require.NoError(t, err)

		require.Equal(t, pub, ed25519.PublicKey(v.Raw()))
	})

	t.Run("invalid length", func(t *testing.T) {
		_, err := FromRaw([]byte{})
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid length")
	})
}

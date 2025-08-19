package decode_test

import (
	"testing"

	"github.com/storacha/go-ucanto/decode"
	"github.com/storacha/go-ucanto/principal"
	"github.com/storacha/go-ucanto/principal/ed25519/signer"
	rsasigner "github.com/storacha/go-ucanto/principal/rsa/signer"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/stretchr/testify/require"
)

func TestSigner(t *testing.T) {
	t.Run("Ed25519 signer", func(t *testing.T) {
		original, err := signer.Generate()
		require.NoError(t, err)

		encoded := original.Encode()
		decoded, err := decode.Signer(encoded)
		require.NoError(t, err)

		require.Equal(t, original.DID().String(), decoded.DID().String())
	})

	t.Run("RSA signer", func(t *testing.T) {
		original, err := rsasigner.Generate()
		require.NoError(t, err)

		encoded := original.Encode()
		decoded, err := decode.Signer(encoded)
		require.NoError(t, err)

		require.Equal(t, original.DID().String(), decoded.DID().String())
	})

	t.Run("Invalid data", func(t *testing.T) {
		_, err := decode.Signer([]byte{0xFF, 0xFF})
		require.Error(t, err)
	})
}

func TestVerifier(t *testing.T) {
	t.Run("Ed25519 verifier", func(t *testing.T) {
		s, err := signer.Generate()
		require.NoError(t, err)
		original := s.Verifier()

		encoded := original.Encode()
		decoded, err := decode.Verifier(encoded)
		require.NoError(t, err)

		require.Equal(t, original.DID().String(), decoded.DID().String())
	})

	t.Run("RSA verifier", func(t *testing.T) {
		s, err := rsasigner.Generate()
		require.NoError(t, err)
		original := s.Verifier()

		encoded := original.Encode()
		decoded, err := decode.Verifier(encoded)
		require.NoError(t, err)

		require.Equal(t, original.DID().String(), decoded.DID().String())
	})
}

func TestPrincipal(t *testing.T) {
	t.Run("Decode signer", func(t *testing.T) {
		s, err := signer.Generate()
		require.NoError(t, err)

		encoded := s.Encode()
		decoded, err := decode.Principal(encoded)
		require.NoError(t, err)

		signer, ok := decoded.(principal.Signer)
		require.True(t, ok)
		require.Equal(t, s.DID().String(), signer.DID().String())
	})

	t.Run("Decode verifier", func(t *testing.T) {
		s, err := signer.Generate()
		require.NoError(t, err)
		v := s.Verifier()

		encoded := v.Encode()
		decoded, err := decode.Principal(encoded)
		require.NoError(t, err)

		verifier, ok := decoded.(ucan.Verifier)
		require.True(t, ok)
		require.Equal(t, v.DID().String(), verifier.DID().String())
	})
}
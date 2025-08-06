package principal

import (
	"testing"

	"github.com/storacha/go-ucanto/principal/ed25519/signer"
	rsasigner "github.com/storacha/go-ucanto/principal/rsa/signer"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/stretchr/testify/require"
)

func TestDecodeSigner(t *testing.T) {
	t.Run("Ed25519 signer", func(t *testing.T) {
		original, err := signer.Generate()
		require.NoError(t, err)

		encoded := original.Encode()
		decoded, err := DecodeSigner(encoded)
		require.NoError(t, err)

		require.Equal(t, original.DID().String(), decoded.DID().String())
	})

	t.Run("RSA signer", func(t *testing.T) {
		original, err := rsasigner.Generate()
		require.NoError(t, err)

		encoded := original.Encode()
		decoded, err := DecodeSigner(encoded)
		require.NoError(t, err)

		require.Equal(t, original.DID().String(), decoded.DID().String())
	})

	t.Run("Invalid data", func(t *testing.T) {
		_, err := DecodeSigner([]byte{0xFF, 0xFF})
		require.Error(t, err)
	})
}

func TestDecodeVerifier(t *testing.T) {
	t.Run("Ed25519 verifier", func(t *testing.T) {
		s, err := signer.Generate()
		require.NoError(t, err)
		original := s.Verifier()

		encoded := original.Encode()
		decoded, err := DecodeVerifier(encoded)
		require.NoError(t, err)

		require.Equal(t, original.DID().String(), decoded.DID().String())
	})

	t.Run("RSA verifier", func(t *testing.T) {
		s, err := rsasigner.Generate()
		require.NoError(t, err)
		original := s.Verifier()

		encoded := original.Encode()
		decoded, err := DecodeVerifier(encoded)
		require.NoError(t, err)

		require.Equal(t, original.DID().String(), decoded.DID().String())
	})
}

func TestDecodePrincipal(t *testing.T) {
	t.Run("Decode signer", func(t *testing.T) {
		s, err := signer.Generate()
		require.NoError(t, err)

		encoded := s.Encode()
		decoded, err := DecodePrincipal(encoded)
		require.NoError(t, err)

		signer, ok := decoded.(Signer)
		require.True(t, ok)
		require.Equal(t, s.DID().String(), signer.DID().String())
	})

	t.Run("Decode verifier", func(t *testing.T) {
		s, err := signer.Generate()
		require.NoError(t, err)
		v := s.Verifier()

		encoded := v.Encode()
		decoded, err := DecodePrincipal(encoded)
		require.NoError(t, err)

		verifier, ok := decoded.(ucan.Verifier)
		require.True(t, ok)
		require.Equal(t, v.DID().String(), verifier.DID().String())
	})
}

func TestParseDID(t *testing.T) {
	t.Run("Parse Ed25519 DID", func(t *testing.T) {
		s, err := signer.Generate()
		require.NoError(t, err)

		verifier, err := ParseDID(s.DID().String())
		require.NoError(t, err)
		require.Equal(t, s.DID().String(), verifier.DID().String())
	})

	t.Run("Parse RSA DID", func(t *testing.T) {
		s, err := rsasigner.Generate()
		require.NoError(t, err)

		verifier, err := ParseDID(s.DID().String())
		require.NoError(t, err)
		require.Equal(t, s.DID().String(), verifier.DID().String())
	})

	t.Run("Invalid DID", func(t *testing.T) {
		_, err := ParseDID("not-a-did")
		require.Error(t, err)
	})
}

func TestComposedParser(t *testing.T) {
	parser := NewComposedParser(Ed25519Parser{}, RSAParser{})

	// Test with Ed25519
	s1, err := signer.Generate()
	require.NoError(t, err)

	v1, err := parser.Parse(s1.DID().String())
	require.NoError(t, err)
	require.Equal(t, s1.DID().String(), v1.DID().String())

	// Test with RSA
	s2, err := rsasigner.Generate()
	require.NoError(t, err)

	v2, err := parser.Parse(s2.DID().String())
	require.NoError(t, err)
	require.Equal(t, s2.DID().String(), v2.DID().String())
}

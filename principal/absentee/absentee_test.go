package absentee

import (
	"testing"

	"github.com/storacha-network/go-ucanto/did"
	"github.com/storacha-network/go-ucanto/ucan/crypto/signature"
	"github.com/stretchr/testify/require"
)

func TestAbsentee(t *testing.T) {
	t.Run("it can sign", func(t *testing.T) {
		alicedid, err := did.Parse("did:mailto:web.mail:alice")
		require.NoError(t, err)

		signer := From(alicedid)
		require.Equal(t, alicedid, signer.DID())
		require.Equal(t, "", signer.SignatureAlgorithm())
		require.Equal(t, signature.NON_STANDARD, int(signer.SignatureCode()))

		sig := signer.Sign([]byte("hello world"))
		require.Equal(t, signature.NON_STANDARD, int(sig.Code()))
		require.Equal(t, []byte{}, sig.Raw())
	})
}

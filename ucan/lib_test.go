package ucan_test

import (
	"testing"

	"github.com/ipld/go-ipld-prime"
	"github.com/storacha/go-ucanto/core/ipld/codec/cbor"
	"github.com/storacha/go-ucanto/principal/ed25519/signer"
	pdm "github.com/storacha/go-ucanto/ucan/datamodel/payload"
	udm "github.com/storacha/go-ucanto/ucan/datamodel/ucan"
	"github.com/storacha/go-ucanto/ucan/formatter"
	"github.com/stretchr/testify/require"
)

func TestDatamodel(t *testing.T) {
	t.Run("nil caveats", func(t *testing.T) {
		issuer, err := signer.Generate()
		require.NoError(t, err)

		audience, err := signer.Generate()
		require.NoError(t, err)

		caps := []udm.CapabilityModel{{
			With: issuer.DID().String(),
			Can:  "test/nilcaveats",
		}}

		payload := pdm.PayloadModel{
			Iss: issuer.DID().String(),
			Aud: audience.DID().String(),
			Att: caps,
			Prf: []string{},
			Fct: []udm.FactModel{},
		}

		sigPayload, err := formatter.FormatSignPayload(payload, "0.9.1", issuer.SignatureAlgorithm())
		require.NoError(t, err)

		model := udm.UCANModel{
			V:   "0.9.1",
			S:   issuer.Sign([]byte(sigPayload)).Bytes(),
			Iss: issuer.DID().Bytes(),
			Aud: audience.DID().Bytes(),
			Att: caps,
			Prf: []ipld.Link{},
			Fct: []udm.FactModel{},
		}

		bytes, err := cbor.Encode(&model, udm.Type())
		require.NoError(t, err)

		var decoded udm.UCANModel
		err = cbor.Decode(bytes, &decoded, udm.Type())
		require.NoError(t, err)
		require.Equal(t, model.Att, decoded.Att)
	})
}

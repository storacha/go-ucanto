package ucan_test

import (
	"testing"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/storacha/go-ucanto/core/ipld/codec/cbor"
	"github.com/storacha/go-ucanto/principal/ed25519/signer"
	"github.com/storacha/go-ucanto/testing/fixtures"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/storacha/go-ucanto/ucan"
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

type testCaveats struct {
	SomeCaveat string
}

func (c testCaveats) ToIPLD() (datamodel.Node, error) {
	np := basicnode.Prototype.Any
	nb := np.NewBuilder()
	ma, _ := nb.BeginMap(1)
	if c.SomeCaveat != "" {
		ma.AssembleKey().AssignString("someCaveat")
		ma.AssembleValue().AssignString(c.SomeCaveat)
	}
	ma.Finish()
	return nb.Build(), nil
}

type testFacts struct {
	SomeFact string
}

func (f testFacts) ToIPLD() (map[string]datamodel.Node, error) {
	np := basicnode.Prototype.Any
	nb := np.NewBuilder()
	ma, _ := nb.BeginMap(1)
	if f.SomeFact != "" {
		ma.AssembleKey().AssignString("someFact")
		ma.AssembleValue().AssignString(f.SomeFact)
	}
	ma.Finish()
	return map[string]datamodel.Node{"fact": nb.Build()}, nil
}

func TestVerifySignature(t *testing.T) {
	cap := ucan.NewCapability(
		"test/capability",
		fixtures.Alice.DID().String(),
		testCaveats{SomeCaveat: "some caveat"},
	)

	fact := testFacts{SomeFact: "some fact"}

	// use all available fields to ensure they are all included in the signature
	u, err := ucan.Issue(
		fixtures.Alice,
		fixtures.Bob,
		[]ucan.Capability[testCaveats]{cap},
		ucan.WithExpiration(ucan.Now()+30),
		ucan.WithNonce("1234567890"),
		ucan.WithNotBefore(ucan.Now()-30),
		ucan.WithFacts([]ucan.FactBuilder{fact}),
		ucan.WithProof(helpers.RandomCID()),
	)
	require.NoError(t, err)

	valid, err := ucan.VerifySignature(u, fixtures.Alice.Verifier())
	require.NoError(t, err)
	require.True(t, valid)
}

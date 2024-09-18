package validator

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/storacha-network/go-ucanto/core/delegation"
	"github.com/storacha-network/go-ucanto/core/result/failure"
	"github.com/storacha-network/go-ucanto/core/schema"
	"github.com/storacha-network/go-ucanto/did"
	"github.com/storacha-network/go-ucanto/principal/absentee"
	"github.com/storacha-network/go-ucanto/principal/ed25519/signer"
	"github.com/storacha-network/go-ucanto/testing/fixtures"
	"github.com/storacha-network/go-ucanto/testing/helpers"
	"github.com/storacha-network/go-ucanto/ucan"
	"github.com/stretchr/testify/require"
)

type debugEchoCaveats struct {
	Message *string
}

func (c debugEchoCaveats) Build() (ipld.Node, error) {
	np := basicnode.Prototype.Any
	nb := np.NewBuilder()
	ma, _ := nb.BeginMap(1)
	if c.Message != nil {
		ma.AssembleKey().AssignString("message")
		ma.AssembleValue().AssignString(*c.Message)
	}
	ma.Finish()
	return nb.Build(), nil
}

var debugEchoTyp = helpers.Must(ipld.LoadSchemaBytes([]byte(`
	type DebugEchoCaveats struct {
		message optional String
	}
`)))

var debugEcho = NewCapability(
	"debug/echo",
	schema.DIDString(schema.WithMethod("mailto")),
	schema.Struct[debugEchoCaveats](debugEchoTyp.TypeByName("DebugEchoCaveats"), nil),
	func(claimed, delegated ucan.Capability[debugEchoCaveats]) failure.Failure {
		if claimed.With() != delegated.With() {
			err := fmt.Errorf("Expected 'with: \"%s\"' instead got '%s'", delegated.With(), claimed.With())
			return failure.FromError(err)
		}
		return nil
	},
)

type attestCaveats struct {
	Proof ipld.Link
}

func (c attestCaveats) Build() (ipld.Node, error) {
	np := basicnode.Prototype.Any
	nb := np.NewBuilder()
	ma, _ := nb.BeginMap(1)
	ma.AssembleKey().AssignString("proof")
	ma.AssembleValue().AssignLink(c.Proof)
	ma.Finish()
	return nb.Build(), nil
}

var attestTyp = helpers.Must(ipld.LoadSchemaBytes([]byte(`
	type AttestCaveats struct {
		proof Link
	}
`)))

var attest = NewCapability(
	"ucan/attest",
	schema.DIDString(),
	schema.Struct[attestCaveats](attestTyp.TypeByName("AttestCaveats"), nil),
	func(claimed, delegated ucan.Capability[attestCaveats]) failure.Failure {
		if claimed.With() != delegated.With() {
			err := fmt.Errorf("Expected 'with: \"%s\"' instead got '%s'", delegated.With(), claimed.With())
			return failure.FromError(err)
		}
		return nil
	},
)

func TestSession(t *testing.T) {
	t.Run("validate mailto", func(t *testing.T) {
		agent := fixtures.Alice
		account := absentee.From(helpers.Must(did.Parse("did:mailto:web.mail:alice")))

		prf, err := debugEcho.Delegate(
			account,
			agent,
			account.DID().String(),
			debugEchoCaveats{},
		)
		require.NoError(t, err)

		session, err := attest.Delegate(
			fixtures.Service,
			agent,
			fixtures.Service.DID().String(),
			attestCaveats{Proof: prf.Link()},
		)
		require.NoError(t, err)

		msg := "Hello World"
		nb := debugEchoCaveats{Message: &msg}
		inv, err := debugEcho.Invoke(
			agent,
			fixtures.Service,
			account.DID().String(),
			nb,
			delegation.WithProofs(delegation.Proofs{
				delegation.FromDelegation(prf),
				delegation.FromDelegation(session),
			}),
		)
		require.NoError(t, err)

		context := NewValidationContext(
			fixtures.Service.Verifier(),
			debugEcho,
			IsSelfIssued,
			validateAuthOk,
			ProofUnavailable,
			parseEdPrincipal,
			FailDIDKeyResolution,
		)

		a, x := Access(inv, context)
		require.NoError(t, x)
		require.Equal(t, debugEcho.Can(), a.Capability().Can())
		require.Equal(t, account.DID().String(), a.Capability().With())
		require.Equal(t, nb, a.Capability().Nb())
	})

	t.Run("delegated ucan attest", func(t *testing.T) {
		agent := fixtures.Alice
		account := absentee.From(helpers.Must(did.Parse("did:mailto:web.mail:alice")))

		manager, err := signer.Generate()
		require.NoError(t, err)
		worker, err := signer.Generate()
		require.NoError(t, err)

		authority, err := delegation.Delegate(
			manager,
			worker,
			[]ucan.Capability[ucan.NoCaveats]{
				ucan.NewCapability("*", fixtures.Service.DID().String(), ucan.NoCaveats{}),
			},
			delegation.WithNoExpiration(),
			delegation.WithProof(
				delegation.FromDelegation(
					helpers.Must(
						delegation.Delegate(
							fixtures.Service,
							manager,
							[]ucan.Capability[ucan.NoCaveats]{
								ucan.NewCapability("*", fixtures.Service.DID().String(), ucan.NoCaveats{}),
							},
						),
					),
				),
			),
		)
		require.NoError(t, err)

		prf, err := debugEcho.Delegate(
			account,
			agent,
			account.DID().String(),
			debugEchoCaveats{},
			delegation.WithNoExpiration(),
		)
		require.NoError(t, err)

		require.Equal(
			t,
			helpers.Must(base64.RawStdEncoding.DecodeString("gKADAA")),
			prf.Signature().Bytes(),
			"should have blank signature",
		)

		session, err := attest.Delegate(
			worker,
			agent,
			fixtures.Service.DID().String(),
			attestCaveats{Proof: prf.Link()},
			delegation.WithProof(delegation.FromDelegation(authority)),
		)
		require.NoError(t, err)

		msg := "Hello World"
		nb := debugEchoCaveats{Message: &msg}
		inv, err := debugEcho.Invoke(
			agent,
			fixtures.Service,
			account.DID().String(),
			nb,
			delegation.WithProofs(delegation.Proofs{
				delegation.FromDelegation(session),
				delegation.FromDelegation(prf),
			}),
		)
		require.NoError(t, err)

		context := NewValidationContext(
			fixtures.Service.Verifier(),
			debugEcho,
			IsSelfIssued,
			validateAuthOk,
			ProofUnavailable,
			parseEdPrincipal,
			FailDIDKeyResolution,
		)

		a, x := Access(inv, context)
		require.NoError(t, x)
		require.Equal(t, debugEcho.Can(), a.Capability().Can())
		require.Equal(t, account.DID().String(), a.Capability().With())
		require.Equal(t, nb, a.Capability().Nb())
	})

	t.Run("fail without proofs", func(t *testing.T) {
		account := absentee.From(helpers.Must(did.Parse("did:mailto:web.mail:alice")))

		msg := "Hello World"
		nb := debugEchoCaveats{Message: &msg}
		inv, err := debugEcho.Invoke(
			account,
			fixtures.Service,
			account.DID().String(),
			nb,
		)
		require.NoError(t, err)

		context := NewValidationContext(
			fixtures.Service.Verifier(),
			debugEcho,
			IsSelfIssued,
			validateAuthOk,
			ProofUnavailable,
			parseEdPrincipal,
			FailDIDKeyResolution,
		)

		a, x := Access(inv, context)
		require.Nil(t, a)
		require.Error(t, x)
		require.Equal(t, x.Name(), "Unauthorized")
		errmsg := strings.Join([]string{
			fmt.Sprintf("Claim %s is not authorized", debugEcho),
			fmt.Sprintf(`  - Unable to resolve '%s' key`, account.DID()),
		}, "\n")
		require.Equal(t, errmsg, x.Error())
	})
}

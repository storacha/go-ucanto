package validator

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/result/failure"
	"github.com/storacha/go-ucanto/core/schema"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/principal/absentee"
	ed25519 "github.com/storacha/go-ucanto/principal/ed25519/signer"
	"github.com/storacha/go-ucanto/principal/signer"
	"github.com/storacha/go-ucanto/testing/fixtures"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/stretchr/testify/require"
)

var serviceDID = helpers.Must(did.Parse("did:web:example.com"))
var service = helpers.Must(signer.Wrap(fixtures.Service, serviceDID))

type debugEchoCaveats struct {
	Message *string
}

func (c debugEchoCaveats) ToIPLD() (ipld.Node, error) {
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

func (c attestCaveats) ToIPLD() (ipld.Node, error) {
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
			service,
			agent,
			service.DID().String(),
			attestCaveats{Proof: prf.Link()},
		)
		require.NoError(t, err)

		msg := "Hello World"
		nb := debugEchoCaveats{Message: &msg}
		inv, err := debugEcho.Invoke(
			agent,
			service,
			account.DID().String(),
			nb,
			delegation.WithProof(
				delegation.FromDelegation(prf),
				delegation.FromDelegation(session),
			),
		)
		require.NoError(t, err)

		context := NewValidationContext(
			service.Verifier(),
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

	t.Run("validate mailto attested by another service", func(t *testing.T) {
		agent := fixtures.Alice
		account := absentee.From(helpers.Must(did.Parse("did:mailto:web.mail:alice")))
		othersvc := helpers.Must(ed25519.Generate())
		othersvc = helpers.Must(signer.Wrap(othersvc, helpers.Must(did.Parse("did:web:other.storage"))))

		prf, err := debugEcho.Delegate(
			account,
			agent,
			account.DID().String(),
			debugEchoCaveats{},
			delegation.WithNoExpiration(),
		)
		require.NoError(t, err)

		session, err := attest.Delegate(
			othersvc,
			agent,
			othersvc.DID().String(),
			attestCaveats{Proof: prf.Link()},
			delegation.WithNoExpiration(),
		)
		require.NoError(t, err)

		msg := "Hello World"
		inv, err := debugEcho.Invoke(
			agent,
			service,
			account.DID().String(),
			debugEchoCaveats{Message: &msg},
			delegation.WithProof(
				delegation.FromDelegation(prf),
				delegation.FromDelegation(session),
			),
			delegation.WithNoExpiration(),
		)
		require.NoError(t, err)

		auth, err := attest.Delegate(
			service,
			othersvc,
			service.DID().String(),
			attestCaveats{Proof: session.Link()},
			delegation.WithNoExpiration(),
		)
		require.NoError(t, err)

		context := NewValidationContext(
			service.Verifier(),
			debugEcho,
			IsSelfIssued,
			validateAuthOk,
			ProofUnavailable,
			parseEdPrincipal,
			func(d did.DID) (did.DID, UnresolvedDID) {
				if d == othersvc.DID() {
					return othersvc.(signer.WrappedSigner).Unwrap().DID(), nil
				}

				return FailDIDKeyResolution(d)
			},
			auth,
		)

		a, x := Access(inv, context)
		require.NoError(t, x)
		require.Equal(t, debugEcho.Can(), a.Capability().Can())
		require.Equal(t, account.DID().String(), a.Capability().With())
		require.Equal(t, debugEchoCaveats{Message: &msg}, a.Capability().Nb())
	})

	t.Run("delegated ucan attest", func(t *testing.T) {
		agent := fixtures.Alice
		account := absentee.From(helpers.Must(did.Parse("did:mailto:web.mail:alice")))

		manager, err := ed25519.Generate()
		require.NoError(t, err)
		worker, err := ed25519.Generate()
		require.NoError(t, err)

		authority, err := delegation.Delegate(
			manager,
			worker,
			[]ucan.Capability[ucan.NoCaveats]{
				ucan.NewCapability("*", service.DID().String(), ucan.NoCaveats{}),
			},
			delegation.WithNoExpiration(),
			delegation.WithProof(
				delegation.FromDelegation(
					helpers.Must(
						delegation.Delegate(
							service,
							manager,
							[]ucan.Capability[ucan.NoCaveats]{
								ucan.NewCapability("*", service.DID().String(), ucan.NoCaveats{}),
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
			service.DID().String(),
			attestCaveats{Proof: prf.Link()},
			delegation.WithProof(delegation.FromDelegation(authority)),
		)
		require.NoError(t, err)

		msg := "Hello World"
		nb := debugEchoCaveats{Message: &msg}
		inv, err := debugEcho.Invoke(
			agent,
			service,
			account.DID().String(),
			nb,
			delegation.WithProof(
				delegation.FromDelegation(session),
				delegation.FromDelegation(prf),
			),
		)
		require.NoError(t, err)

		context := NewValidationContext(
			service.Verifier(),
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
			service,
			account.DID().String(),
			nb,
		)
		require.NoError(t, err)

		context := NewValidationContext(
			service.Verifier(),
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

	t.Run("fail without session", func(t *testing.T) {
		account := absentee.From(helpers.Must(did.Parse("did:mailto:web.mail:alice")))
		agent := fixtures.Alice

		prf, err := debugEcho.Delegate(
			account,
			agent,
			account.DID().String(),
			debugEchoCaveats{},
			delegation.WithNoExpiration(),
		)
		require.NoError(t, err)

		msg := "Hello World"
		nb := debugEchoCaveats{Message: &msg}
		inv, err := debugEcho.Invoke(
			account,
			service,
			account.DID().String(),
			nb,
			delegation.WithProof(delegation.FromDelegation(prf)),
		)
		require.NoError(t, err)

		context := NewValidationContext(
			service.Verifier(),
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
		require.Contains(t, x.Error(), fmt.Sprintf(`Unable to resolve '%s'`, account.DID()))
	})

	t.Run("fail invalid ucan/attest proof", func(t *testing.T) {
		account := absentee.From(helpers.Must(did.Parse("did:mailto:web.mail:alice")))
		agent := fixtures.Alice
		othersvc := helpers.Must(ed25519.Generate())

		prf, err := debugEcho.Delegate(
			account,
			agent,
			account.DID().String(),
			debugEchoCaveats{},
			delegation.WithNoExpiration(),
		)
		require.NoError(t, err)

		session, err := attest.Delegate(
			othersvc,
			agent,
			service.DID().String(),
			attestCaveats{Proof: prf.Link()},
			delegation.WithProof(
				delegation.FromDelegation(
					helpers.Must(
						delegation.Delegate(
							service,
							othersvc,
							[]ucan.Capability[ucan.NoCaveats]{
								// Noting that this is a DID key, not did:web:example.com
								// which is why session is invalid
								ucan.NewCapability("*", service.Unwrap().DID().String(), ucan.NoCaveats{}),
							},
						),
					),
				),
			),
		)
		require.NoError(t, err)

		msg := "Hello World"
		nb := debugEchoCaveats{Message: &msg}
		inv, err := debugEcho.Invoke(
			agent,
			service,
			account.DID().String(),
			nb,
			delegation.WithProof(
				delegation.FromDelegation(prf),
				delegation.FromDelegation(session),
			),
		)
		require.NoError(t, err)

		context := NewValidationContext(
			service.Verifier(),
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
		require.Contains(t, x.Error(), "has an invalid session")
	})

	t.Run("fail unknown ucan/attest proof", func(t *testing.T) {
		account := absentee.From(helpers.Must(did.Parse("did:mailto:web.mail:alice")))
		agent := fixtures.Alice
		othersvc := helpers.Must(ed25519.Generate())
		othersvc = helpers.Must(signer.Wrap(othersvc, helpers.Must(did.Parse("did:web:other.storage"))))

		prf, err := debugEcho.Delegate(
			account,
			agent,
			account.DID().String(),
			debugEchoCaveats{},
			delegation.WithNoExpiration(),
		)
		require.NoError(t, err)

		session, err := attest.Delegate(
			othersvc,
			agent,
			othersvc.DID().String(),
			attestCaveats{Proof: prf.Link()},
			delegation.WithNoExpiration(),
		)
		require.NoError(t, err)

		msg := "Hello World"
		inv, err := debugEcho.Invoke(
			agent,
			service,
			account.DID().String(),
			debugEchoCaveats{Message: &msg},
			delegation.WithProof(
				delegation.FromDelegation(prf),
				delegation.FromDelegation(session),
			),
			delegation.WithNoExpiration(),
		)
		require.NoError(t, err)

		// authority is service, but attestation was issued by othersvc
		context := NewValidationContext(
			service.Verifier(),
			debugEcho,
			IsSelfIssued,
			validateAuthOk,
			ProofUnavailable,
			parseEdPrincipal,
			func(d did.DID) (did.DID, UnresolvedDID) {
				if d == othersvc.DID() {
					return othersvc.(signer.WrappedSigner).Unwrap().DID(), nil
				}

				return FailDIDKeyResolution(d)
			},
		)

		a, x := Access(inv, context)
		require.Nil(t, a)
		require.Error(t, x)
		require.Equal(t, x.Name(), "Unauthorized")
		require.Contains(t, x.Error(), "Unable to resolve 'did:mailto:web.mail:alice'")
	})

	t.Run("resolve key", func(t *testing.T) {
		accountDID := helpers.Must(did.Parse("did:mailto:web.mail:alice"))
		account := helpers.Must(signer.Wrap(fixtures.Alice, accountDID))

		msg := "Hello World"
		nb := debugEchoCaveats{Message: &msg}
		inv, err := debugEcho.Invoke(
			account,
			service,
			account.DID().String(),
			nb,
		)
		require.NoError(t, err)

		context := NewValidationContext(
			service.Verifier(),
			debugEcho,
			IsSelfIssued,
			validateAuthOk,
			ProofUnavailable,
			parseEdPrincipal,
			func(d did.DID) (did.DID, UnresolvedDID) {
				return fixtures.Alice.DID(), nil
			},
		)

		a, x := Access(inv, context)
		require.NoError(t, x)
		require.Equal(t, debugEcho.Can(), a.Capability().Can())
		require.Equal(t, account.DID().String(), a.Capability().With())
		require.Equal(t, nb, a.Capability().Nb())
	})

	t.Run("service can not delegate access to account", func(t *testing.T) {
		accountDID := helpers.Must(did.Parse("did:mailto:web.mail:alice"))
		account := absentee.From(accountDID)

		// service should not be able to delegate access to account resource
		auth, err := debugEcho.Delegate(
			service,
			fixtures.Alice,
			account.DID().String(),
			debugEchoCaveats{},
		)
		require.NoError(t, err)

		session, err := attest.Delegate(
			service,
			fixtures.Alice,
			service.DID().String(),
			attestCaveats{Proof: auth.Link()},
			delegation.WithProof(delegation.FromDelegation(auth)),
		)
		require.NoError(t, err)

		msg := "Hello World"
		nb := debugEchoCaveats{Message: &msg}
		inv, err := debugEcho.Invoke(
			fixtures.Alice,
			service,
			account.DID().String(),
			nb,
			delegation.WithProof(
				delegation.FromDelegation(auth),
				delegation.FromDelegation(session),
			),
		)
		require.NoError(t, err)

		context := NewValidationContext(
			service.Verifier(),
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
	})

	t.Run("attest with an account DID", func(t *testing.T) {
		accountDID := helpers.Must(did.Parse("did:mailto:web.mail:alice"))
		account := absentee.From(accountDID)

		// service should not be able to delegate access to account resource
		auth, err := debugEcho.Delegate(
			service,
			fixtures.Alice,
			account.DID().String(),
			debugEchoCaveats{},
		)
		require.NoError(t, err)

		session, err := attest.Delegate(
			service,
			fixtures.Alice,
			// this should be an service did instead
			account.DID().String(),
			attestCaveats{Proof: auth.Link()},
			delegation.WithProof(delegation.FromDelegation(auth)),
		)
		require.NoError(t, err)

		msg := "Hello World"
		nb := debugEchoCaveats{Message: &msg}
		inv, err := debugEcho.Invoke(
			fixtures.Alice,
			service,
			account.DID().String(),
			nb,
			delegation.WithProof(
				delegation.FromDelegation(auth),
				delegation.FromDelegation(session),
			),
		)
		require.NoError(t, err)

		context := NewValidationContext(
			service.Verifier(),
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
	})

	t.Run("service cannot delegate account resource", func(t *testing.T) {
		accountDID := helpers.Must(did.Parse("did:mailto:web.mail:alice"))
		account := absentee.From(accountDID)

		prf, err := debugEcho.Delegate(
			service,
			fixtures.Alice,
			account.DID().String(),
			debugEchoCaveats{},
		)
		require.NoError(t, err)

		msg := "Hello World"
		nb := debugEchoCaveats{Message: &msg}
		inv, err := debugEcho.Invoke(
			fixtures.Alice,
			service,
			account.DID().String(),
			nb,
			delegation.WithProof(delegation.FromDelegation(prf)),
		)
		require.NoError(t, err)

		context := NewValidationContext(
			service.Verifier(),
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
	})

	t.Run("redundant proofs have no impact", func(t *testing.T) {
		accountDID := helpers.Must(did.Parse("did:mailto:web.mail:alice"))
		account := absentee.From(accountDID)

		var logins delegation.Proofs
		for i := range 6 {
			dlg, err := delegation.Delegate(
				account,
				fixtures.Alice,
				[]ucan.Capability[ucan.NoCaveats]{
					ucan.NewCapability("*", "ucan:*", ucan.NoCaveats{}),
				},
				delegation.WithNoExpiration(),
				delegation.WithNonce(fmt.Sprint(i)),
			)
			require.NoError(t, err)
			logins = append(logins, delegation.FromDelegation(dlg))
		}

		exp := ucan.Now() + 60*60*24*365 // 1 year
		var attestations delegation.Proofs
		for _, login := range logins {
			dlg, err := attest.Delegate(
				service,
				fixtures.Alice,
				service.DID().String(),
				attestCaveats{Proof: login.Link()},
				delegation.WithExpiration(exp),
			)
			require.NoError(t, err)
			attestations = append(attestations, delegation.FromDelegation(dlg))
		}

		var prfs delegation.Proofs
		prfs = append(prfs, logins...)
		prfs = append(prfs, attestations...)

		msg := "Hello World"
		nb := debugEchoCaveats{Message: &msg}
		inv, err := debugEcho.Invoke(
			fixtures.Alice,
			service,
			account.DID().String(),
			nb,
			delegation.WithProof(prfs...),
		)
		require.NoError(t, err)

		context := NewValidationContext(
			service.Verifier(),
			debugEcho,
			IsSelfIssued,
			validateAuthOk,
			ProofUnavailable,
			parseEdPrincipal,
			FailDIDKeyResolution,
		)

		a, x := Access(inv, context)
		require.NotEmpty(t, a)
		require.NoError(t, x)
	})
}

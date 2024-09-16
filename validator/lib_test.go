package validator

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/storacha-network/go-ucanto/core/dag/blockstore"
	"github.com/storacha-network/go-ucanto/core/delegation"
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/ipld/block"
	"github.com/storacha-network/go-ucanto/core/ipld/codec/cbor"
	"github.com/storacha-network/go-ucanto/core/ipld/hash/sha256"
	"github.com/storacha-network/go-ucanto/core/result/failure"
	"github.com/storacha-network/go-ucanto/core/schema"
	"github.com/storacha-network/go-ucanto/principal"
	"github.com/storacha-network/go-ucanto/principal/ed25519/verifier"
	"github.com/storacha-network/go-ucanto/testing/fixtures"
	"github.com/storacha-network/go-ucanto/ucan"
	udm "github.com/storacha-network/go-ucanto/ucan/datamodel/ucan"
	"github.com/stretchr/testify/require"
)

type storeAddCaveats struct {
	Link   ipld.Link
	Origin ipld.Link
}

func (c storeAddCaveats) Build() (ipld.Node, error) {
	np := basicnode.Prototype.Any
	nb := np.NewBuilder()
	ma, _ := nb.BeginMap(2)
	if c != (storeAddCaveats{}) {
		ma.AssembleKey().AssignString("link")
		ma.AssembleValue().AssignLink(c.Link)
		if c.Origin != nil {
			ma.AssembleKey().AssignString("origin")
			ma.AssembleValue().AssignLink(c.Origin)
		}
	}
	ma.Finish()
	return nb.Build(), nil
}

func newStoreAddCapability(t *testing.T) CapabilityParser[storeAddCaveats] {
	t.Helper()

	typ, err := ipld.LoadSchemaBytes([]byte(`
		type StoreAddCaveats struct {
			link Link
			origin optional Link
		}
	`))
	require.NoError(t, err)

	return NewCapability(
		"store/add",
		schema.DIDString(),
		schema.Struct[storeAddCaveats](typ.TypeByName("StoreAddCaveats"), nil),
		func(claimed, delegated ucan.Capability[storeAddCaveats]) failure.Failure {
			if claimed.With() != delegated.With() {
				err := fmt.Errorf("Expected 'with: \"%s\"' instead got '%s'", delegated.With(), claimed.With())
				return failure.FromError(err)
			}
			if delegated.Nb().Link != nil && delegated.Nb().Link != claimed.Nb().Link {
				var err error
				if claimed.Nb().Link == nil {
					err = fmt.Errorf("Link violates imposed %s constraint", delegated.Nb().Link)
				} else {
					err = fmt.Errorf("Link %s violates imposed %s constraint", claimed.Nb().Link, delegated.Nb().Link)
				}
				return failure.FromError(err)
			}
			return nil
		},
	)
}

func TestAccess(t *testing.T) {
	storeAdd := newStoreAddCapability(t)
	testLink := cidlink.Link{Cid: cid.MustParse("bafkqaaa")}
	validateAuthOk := func(auth Authorization[any]) Revoked { return nil }
	parseEdPrincipal := func(str string) (principal.Verifier, error) {
		return verifier.Parse(str)
	}

	t.Run("authorized", func(t *testing.T) {
		t.Run("self-issued invocation", func(t *testing.T) {
			inv, err := storeAdd.Invoke(
				fixtures.Alice,
				fixtures.Bob,
				fixtures.Alice.DID().String(),
				storeAddCaveats{Link: testLink},
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
				IsSelfIssued,
				validateAuthOk,
				ProofUnavailable,
				parseEdPrincipal,
				FailDIDKeyResolution,
			)

			a, x := Access(inv, context)
			require.NoError(t, x)
			require.Equal(t, storeAdd.Can(), a.Capability().Can())
			require.Equal(t, fixtures.Alice.DID().String(), a.Capability().With())
			require.Equal(t, fixtures.Alice.DID(), a.Issuer().DID())
			require.Equal(t, fixtures.Bob.DID(), a.Audience().DID())
		})

		t.Run("delegated invocation", func(t *testing.T) {
			dlg, err := storeAdd.Delegate(
				fixtures.Alice,
				fixtures.Bob,
				fixtures.Alice.DID().String(),
				storeAddCaveats{},
			)
			require.NoError(t, err)

			inv, err := storeAdd.Invoke(
				fixtures.Bob,
				fixtures.Service,
				fixtures.Alice.DID().String(),
				storeAddCaveats{Link: testLink},
				delegation.WithProof(delegation.FromDelegation(dlg)),
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
				IsSelfIssued,
				validateAuthOk,
				ProofUnavailable,
				parseEdPrincipal,
				FailDIDKeyResolution,
			)

			a, x := Access(inv, context)
			require.NoError(t, x)
			require.Equal(t, storeAdd.Can(), a.Capability().Can())
			require.Equal(t, fixtures.Alice.DID().String(), a.Capability().With())
			require.Equal(t, fixtures.Bob.DID(), a.Issuer().DID())
			require.Equal(t, fixtures.Service.DID(), a.Audience().DID())
		})

		t.Run("delegation chain", func(t *testing.T) {
			alice2bob, err := storeAdd.Delegate(
				fixtures.Alice,
				fixtures.Bob,
				fixtures.Alice.DID().String(),
				storeAddCaveats{},
			)
			require.NoError(t, err)

			bob2mallory, err := storeAdd.Delegate(
				fixtures.Bob,
				fixtures.Mallory,
				fixtures.Alice.DID().String(),
				storeAddCaveats{},
				delegation.WithProof(delegation.FromDelegation(alice2bob)),
			)
			require.NoError(t, err)

			inv, err := storeAdd.Invoke(
				fixtures.Mallory,
				fixtures.Service,
				fixtures.Alice.DID().String(),
				storeAddCaveats{Link: testLink},
				delegation.WithProof(delegation.FromDelegation(bob2mallory)),
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
				IsSelfIssued,
				validateAuthOk,
				ProofUnavailable,
				parseEdPrincipal,
				FailDIDKeyResolution,
			)

			a, x := Access(inv, context)
			require.NoError(t, x)
			require.Equal(t, storeAdd.Can(), a.Capability().Can())
			require.Equal(t, fixtures.Alice.DID().String(), a.Capability().With())
			require.Equal(t, fixtures.Mallory.DID(), a.Issuer().DID())
			require.Equal(t, fixtures.Service.DID(), a.Audience().DID())

			require.Equal(t, storeAdd.Can(), a.Proofs()[0].Capability().Can())
			require.Equal(t, fixtures.Alice.DID().String(), a.Proofs()[0].Capability().With())
			require.Equal(t, fixtures.Bob.DID(), a.Proofs()[0].Issuer().DID())
			require.Equal(t, fixtures.Mallory.DID(), a.Proofs()[0].Audience().DID())

			require.Equal(t, storeAdd.Can(), a.Proofs()[0].Proofs()[0].Capability().Can())
			require.Equal(t, fixtures.Alice.DID().String(), a.Proofs()[0].Proofs()[0].Capability().With())
			require.Equal(t, fixtures.Alice.DID(), a.Proofs()[0].Proofs()[0].Issuer().DID())
			require.Equal(t, fixtures.Bob.DID(), a.Proofs()[0].Proofs()[0].Audience().DID())
		})
	})

	t.Run("unauthorized", func(t *testing.T) {
		t.Run("expired invocation", func(t *testing.T) {
			exp := ucan.Now() - 5
			inv, err := storeAdd.Invoke(
				fixtures.Alice,
				fixtures.Service,
				fixtures.Alice.DID().String(),
				storeAddCaveats{Link: testLink},
				delegation.WithExpiration(exp),
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
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
			msg := strings.Join([]string{
				fmt.Sprintf("Claim %s is not authorized", storeAdd),
				fmt.Sprintf("  - Proof %s has expired on %s", inv.Link(), time.Unix(int64(exp), 0).Format(time.RFC3339)),
			}, "\n")
			require.Equal(t, msg, x.Error())
		})

		t.Run("not valid before", func(t *testing.T) {
			nbf := ucan.Now() + 500
			inv, err := storeAdd.Invoke(
				fixtures.Alice,
				fixtures.Service,
				fixtures.Alice.DID().String(),
				storeAddCaveats{Link: testLink},
				delegation.WithNotBefore(nbf),
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
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
			msg := strings.Join([]string{
				fmt.Sprintf("Claim %s is not authorized", storeAdd),
				fmt.Sprintf("  - Proof %s is not valid before %s", inv.Link(), time.Unix(int64(nbf), 0).Format(time.RFC3339)),
			}, "\n")
			require.Equal(t, msg, x.Error())
		})

		t.Run("invalid signature", func(t *testing.T) {
			inv, err := storeAdd.Invoke(
				fixtures.Alice,
				fixtures.Service,
				fixtures.Alice.DID().String(),
				storeAddCaveats{Link: testLink},
			)
			require.NoError(t, err)

			inv.Data().Model().S = fixtures.Bob.Sign(inv.Root().Bytes()).Bytes()

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
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
			msg := strings.Join([]string{
				fmt.Sprintf("Claim %s is not authorized", storeAdd),
				fmt.Sprintf("  - Proof %s does not have a valid signature from %s", inv.Link(), fixtures.Alice.DID()),
			}, "\n")
			require.Equal(t, msg, x.Error())
		})

		t.Run("unknown capability", func(t *testing.T) {
			inv, err := invocation.Invoke(
				fixtures.Alice,
				fixtures.Service,
				ucan.NewCapability(
					"store/write",
					fixtures.Alice.DID().String(),
					ucan.NoCaveats{},
				),
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
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
			msg := strings.Join([]string{
				fmt.Sprintf("Claim %s is not authorized", storeAdd),
				"  - No matching delegated capability found",
				"  - Encountered unknown capabilities",
				fmt.Sprintf("    - {\"can\":\"store/write\",\"with\":\"%s\",\"nb\":{}}", fixtures.Alice.DID()),
			}, "\n")
			require.Equal(t, msg, x.Error())
		})
	})

	t.Run("invalid claim", func(t *testing.T) {
		t.Run("no proofs", func(t *testing.T) {
			inv, err := storeAdd.Invoke(
				fixtures.Alice,
				fixtures.Bob,
				fixtures.Bob.DID().String(),
				storeAddCaveats{Link: testLink},
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
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
			msg := strings.Join([]string{
				fmt.Sprintf("Claim %s is not authorized", storeAdd),
				fmt.Sprintf(`  - Capability {"can":"%s","with":"%s","nb":{"Link":{"/":"%s"},"Origin":null}} is not authorized because:`, storeAdd.Can(), fixtures.Bob.DID(), testLink),
				fmt.Sprintf("    - Capability can not be (self) issued by '%s'", fixtures.Alice.DID()),
				"    - Delegated capability not found",
			}, "\n")
			require.Equal(t, msg, x.Error())
		})

		t.Run("expired", func(t *testing.T) {
			exp := ucan.Now() - 5
			dlg, err := storeAdd.Delegate(
				fixtures.Alice,
				fixtures.Bob,
				fixtures.Alice.DID().String(),
				storeAddCaveats{},
				delegation.WithExpiration(exp),
			)
			require.NoError(t, err)

			inv, err := storeAdd.Invoke(
				fixtures.Bob,
				fixtures.Service,
				fixtures.Alice.DID().String(),
				storeAddCaveats{Link: testLink},
				delegation.WithProof(delegation.FromDelegation(dlg)),
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
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
			msg := strings.Join([]string{
				fmt.Sprintf("Claim %s is not authorized", storeAdd),
				fmt.Sprintf(`  - Capability {"can":"%s","with":"%s","nb":{"Link":{"/":"%s"},"Origin":null}} is not authorized because:`, storeAdd.Can(), fixtures.Alice.DID(), testLink),
				fmt.Sprintf("    - Capability can not be (self) issued by '%s'", fixtures.Bob.DID()),
				fmt.Sprintf("    - Capability can not be derived from prf: %s because:", dlg.Link()),
				fmt.Sprintf("      - Proof %s has expired on %s", dlg.Link(), time.Unix(int64(exp), 0).Format(time.RFC3339)),
			}, "\n")
			require.Equal(t, msg, x.Error())
		})

		t.Run("not valid before", func(t *testing.T) {
			nbf := ucan.Now() + 60*60
			dlg, err := storeAdd.Delegate(
				fixtures.Alice,
				fixtures.Bob,
				fixtures.Alice.DID().String(),
				storeAddCaveats{},
				delegation.WithNotBefore(nbf),
			)
			require.NoError(t, err)

			inv, err := storeAdd.Invoke(
				fixtures.Bob,
				fixtures.Service,
				fixtures.Alice.DID().String(),
				storeAddCaveats{Link: testLink},
				delegation.WithProof(delegation.FromDelegation(dlg)),
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
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
			msg := strings.Join([]string{
				fmt.Sprintf("Claim %s is not authorized", storeAdd),
				fmt.Sprintf(`  - Capability {"can":"%s","with":"%s","nb":{"Link":{"/":"%s"},"Origin":null}} is not authorized because:`, storeAdd.Can(), fixtures.Alice.DID(), testLink),
				fmt.Sprintf("    - Capability can not be (self) issued by '%s'", fixtures.Bob.DID()),
				fmt.Sprintf("    - Capability can not be derived from prf: %s because:", dlg.Link()),
				fmt.Sprintf("      - Proof %s is not valid before %s", dlg.Link(), time.Unix(int64(nbf), 0).Format(time.RFC3339)),
			}, "\n")
			require.Equal(t, msg, x.Error())
		})

		t.Run("invalid signature", func(t *testing.T) {
			// In order to mess up the signature we need to reach deep in UCAN library
			// to create a UCAN model, manually setting the signature to something bad
			// and then encode it as the root block of the delegation.
			nb, _ := storeAddCaveats{Link: testLink}.Build()
			model := udm.UCANModel{
				V:   "0.9.1",
				S:   fixtures.Alice.Sign([]byte{}).Bytes(),
				Iss: fixtures.Alice.DID().Bytes(),
				Aud: fixtures.Bob.DID().Bytes(),
				Att: []udm.CapabilityModel{
					{
						Can:  storeAdd.Can(),
						With: fixtures.Alice.DID().String(),
						Nb:   nb,
					},
				},
				Exp: ucan.Now() + 30,
			}

			rt, err := block.Encode(&model, udm.Type(), cbor.Codec, sha256.Hasher)
			require.NoError(t, err)

			bs, err := blockstore.NewBlockStore(blockstore.WithBlocks([]block.Block{rt}))
			require.NoError(t, err)

			dlg := delegation.NewDelegation(rt, bs)

			inv, err := storeAdd.Invoke(
				fixtures.Bob,
				fixtures.Service,
				fixtures.Alice.DID().String(),
				storeAddCaveats{Link: testLink},
				delegation.WithProof(delegation.FromDelegation(dlg)),
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
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
			msg := strings.Join([]string{
				fmt.Sprintf("Claim %s is not authorized", storeAdd),
				fmt.Sprintf(`  - Capability {"can":"%s","with":"%s","nb":{"Link":{"/":"%s"},"Origin":null}} is not authorized because:`, storeAdd.Can(), fixtures.Alice.DID(), testLink),
				fmt.Sprintf("    - Capability can not be (self) issued by '%s'", fixtures.Bob.DID()),
				fmt.Sprintf("    - Capability can not be derived from prf: %s because:", dlg.Link()),
				fmt.Sprintf("      - Proof %s does not have a valid signature from %s", dlg.Link(), fixtures.Alice.DID()),
			}, "\n")
			require.Equal(t, msg, x.Error())
		})

		t.Run("unknown capability", func(t *testing.T) {
			dlg, err := delegation.Delegate(
				fixtures.Alice,
				fixtures.Bob,
				[]ucan.Capability[ucan.NoCaveats]{
					ucan.NewCapability("store/pin", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
				},
			)
			require.NoError(t, err)

			inv, err := storeAdd.Invoke(
				fixtures.Bob,
				fixtures.Service,
				fixtures.Alice.DID().String(),
				storeAddCaveats{Link: testLink},
				delegation.WithProof(delegation.FromDelegation(dlg)),
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
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
			msg := strings.Join([]string{
				fmt.Sprintf("Claim %s is not authorized", storeAdd),
				fmt.Sprintf(`  - Capability {"can":"%s","with":"%s","nb":{"Link":{"/":"%s"},"Origin":null}} is not authorized because:`, storeAdd.Can(), fixtures.Alice.DID(), testLink),
				fmt.Sprintf("    - Capability can not be (self) issued by '%s'", fixtures.Bob.DID()),
				"    - Delegated capability not found",
				"    - Encountered unknown capabilities",
				fmt.Sprintf(`      - {"can":"store/pin","with":"%s","nb":{}}`, fixtures.Alice.DID()),
			}, "\n")
			require.Equal(t, msg, x.Error())
		})

		t.Run("malformed capability", func(t *testing.T) {
			badDID := fmt.Sprintf("bib:%s", fixtures.Alice.DID().String()[4:])
			dlg, err := storeAdd.Delegate(
				fixtures.Alice,
				fixtures.Bob,
				badDID,
				storeAddCaveats{},
			)
			require.NoError(t, err)

			inv, err := storeAdd.Invoke(
				fixtures.Bob,
				fixtures.Service,
				fixtures.Alice.DID().String(),
				storeAddCaveats{Link: testLink},
				delegation.WithProof(delegation.FromDelegation(dlg)),
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
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
			msg := strings.Join([]string{
				fmt.Sprintf("Claim %s is not authorized", storeAdd),
				fmt.Sprintf(`  - Capability {"can":"%s","with":"%s","nb":{"Link":{"/":"%s"},"Origin":null}} is not authorized because:`, storeAdd.Can(), fixtures.Alice.DID(), testLink),
				fmt.Sprintf("    - Capability can not be (self) issued by '%s'", fixtures.Bob.DID()),
				fmt.Sprintf(`    - Cannot derive {"can":"%s","with":"%s","nb":{"Link":{"/":"%s"},"Origin":null}} from delegated capabilities:`, storeAdd.Can(), fixtures.Alice.DID(), testLink),
				fmt.Sprintf(`      - Encountered malformed '%s' capability: {"can":"%s","with":"%s","nb":{}}`, storeAdd.Can(), storeAdd.Can(), badDID),
				fmt.Sprintf(`        - Expected a "did:" but got "%s" instead`, badDID),
			}, "\n")
			require.Equal(t, msg, x.Error())
		})

		t.Run("unavailable proof", func(t *testing.T) {
			dlg, err := storeAdd.Delegate(
				fixtures.Alice,
				fixtures.Bob,
				fixtures.Alice.DID().String(),
				storeAddCaveats{},
			)
			require.NoError(t, err)

			inv, err := storeAdd.Invoke(
				fixtures.Bob,
				fixtures.Service,
				fixtures.Alice.DID().String(),
				storeAddCaveats{Link: testLink},
				delegation.WithProof(delegation.FromLink(dlg.Link())),
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
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
			msg := strings.Join([]string{
				fmt.Sprintf("Claim %s is not authorized", storeAdd),
				fmt.Sprintf(`  - Capability {"can":"%s","with":"%s","nb":{"Link":{"/":"%s"},"Origin":null}} is not authorized because:`, storeAdd.Can(), fixtures.Alice.DID(), testLink),
				fmt.Sprintf("    - Capability can not be (self) issued by '%s'", fixtures.Bob.DID()),
				fmt.Sprintf(`    - Capability can not be derived from prf: %s because:`, dlg.Link()),
				fmt.Sprintf(`      - Linked proof "%s" is not included and could not be resolved`, dlg.Link()),
				`        - Proof resolution failed with: no proof resolver configured`,
			}, "\n")
			require.Equal(t, msg, x.Error())
		})

		t.Run("invalid audience", func(t *testing.T) {
			dlg, err := storeAdd.Delegate(
				fixtures.Alice,
				fixtures.Bob,
				fixtures.Alice.DID().String(),
				storeAddCaveats{},
			)
			require.NoError(t, err)

			inv, err := storeAdd.Invoke(
				fixtures.Mallory,
				fixtures.Service,
				fixtures.Alice.DID().String(),
				storeAddCaveats{Link: testLink},
				delegation.WithProof(delegation.FromDelegation(dlg)),
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
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
			msg := strings.Join([]string{
				fmt.Sprintf("Claim %s is not authorized", storeAdd),
				fmt.Sprintf(`  - Capability {"can":"%s","with":"%s","nb":{"Link":{"/":"%s"},"Origin":null}} is not authorized because:`, storeAdd.Can(), fixtures.Alice.DID(), testLink),
				fmt.Sprintf("    - Capability can not be (self) issued by '%s'", fixtures.Mallory.DID()),
				fmt.Sprintf(`    - Capability can not be derived from prf: %s because:`, dlg.Link()),
				fmt.Sprintf(`      - Delegation audience is '%s' instead of '%s'`, fixtures.Bob.DID(), fixtures.Mallory.DID()),
			}, "\n")
			require.Equal(t, msg, x.Error())
		})

		t.Run("invalid claim", func(t *testing.T) {
			dlg, err := storeAdd.Delegate(
				fixtures.Alice,
				fixtures.Bob,
				fixtures.Mallory.DID().String(),
				storeAddCaveats{},
			)
			require.NoError(t, err)

			nb := storeAddCaveats{Link: testLink}
			inv, err := storeAdd.Invoke(
				fixtures.Bob,
				fixtures.Service,
				fixtures.Alice.DID().String(),
				nb,
				delegation.WithProof(delegation.FromDelegation(dlg)),
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
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
			msg := strings.Join([]string{
				fmt.Sprintf("Claim %s is not authorized", storeAdd),
				fmt.Sprintf(`  - Capability {"can":"%s","with":"%s","nb":{"Link":{"/":"%s"},"Origin":null}} is not authorized because:`, storeAdd.Can(), fixtures.Alice.DID(), testLink),
				fmt.Sprintf("    - Capability can not be (self) issued by '%s'", fixtures.Bob.DID()),
				fmt.Sprintf(`    - Cannot derive {"can":"%s","with":"%s","nb":{"Link":{"/":"%s"},"Origin":null}} from delegated capabilities:`, storeAdd.Can(), fixtures.Alice.DID(), testLink),
				fmt.Sprintf(`      - Constraint violation: Expected 'with: "%s"' instead got '%s'`, fixtures.Mallory.DID(), fixtures.Alice.DID()),
			}, "\n")
			require.Equal(t, msg, x.Error())
		})

		t.Run("invalid sub delegation", func(t *testing.T) {
			prf, err := storeAdd.Delegate(
				fixtures.Alice,
				fixtures.Bob,
				fixtures.Service.DID().String(),
				storeAddCaveats{},
			)
			require.NoError(t, err)

			dlg, err := storeAdd.Delegate(
				fixtures.Bob,
				fixtures.Mallory,
				fixtures.Service.DID().String(),
				storeAddCaveats{},
				delegation.WithProof(delegation.FromDelegation(prf)),
			)
			require.NoError(t, err)

			nb := storeAddCaveats{Link: testLink}
			inv, err := storeAdd.Invoke(
				fixtures.Mallory,
				fixtures.Service,
				fixtures.Service.DID().String(),
				nb,
				delegation.WithProof(delegation.FromDelegation(dlg)),
			)
			require.NoError(t, err)

			context := NewValidationContext(
				fixtures.Service.Verifier(),
				storeAdd,
				IsSelfIssued,
				validateAuthOk,
				ProofUnavailable,
				parseEdPrincipal,
				FailDIDKeyResolution,
			)

			cstr := fmt.Sprintf(`{"can":"%s","with":"%s","nb":{"Link":{"/":"%s"},"Origin":null}}`, storeAdd.Can(), fixtures.Service.DID(), testLink)
			a, x := Access(inv, context)
			require.Nil(t, a)
			require.Error(t, x)
			require.Equal(t, x.Name(), "Unauthorized")
			msg := strings.Join([]string{
				fmt.Sprintf("Claim %s is not authorized", storeAdd),
				fmt.Sprintf(`  - Capability %s is not authorized because:`, cstr),
				fmt.Sprintf("    - Capability can not be (self) issued by '%s'", fixtures.Mallory.DID()),
				fmt.Sprintf(`    - Capability %s is not authorized because:`, cstr),
				fmt.Sprintf(`      - Capability can not be (self) issued by '%s'`, fixtures.Bob.DID()),
				fmt.Sprintf(`      - Capability %s is not authorized because:`, cstr),
				fmt.Sprintf(`        - Capability can not be (self) issued by '%s'`, fixtures.Alice.DID()),
				"        - Delegated capability not found",
			}, "\n")
			require.Equal(t, msg, x.Error())
		})
	})
}

func TestIsSelfIssued(t *testing.T) {
	cap := ucan.NewCapability("upload/add", fixtures.Alice.DID().String(), struct{}{})

	canIssue := IsSelfIssued(cap, fixtures.Alice.DID())
	if canIssue == false {
		t.Fatal("capability self issued by alice")
	}

	canIssue = IsSelfIssued(cap, fixtures.Bob.DID())
	if canIssue == true {
		t.Fatal("capability not self issued by bob")
	}
}

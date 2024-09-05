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
	"github.com/storacha-network/go-ucanto/core/delegation"
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/result/failure"
	"github.com/storacha-network/go-ucanto/core/schema"
	"github.com/storacha-network/go-ucanto/principal"
	"github.com/storacha-network/go-ucanto/principal/ed25519/verifier"
	"github.com/storacha-network/go-ucanto/testing/fixtures"
	"github.com/storacha-network/go-ucanto/ucan"
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
	ma.AssembleKey().AssignString("link")
	ma.AssembleValue().AssignLink(c.Link)
	if c.Origin != nil {
		ma.AssembleKey().AssignString("origin")
		ma.AssembleValue().AssignLink(c.Origin)
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
			inv, err := invocation.Invoke(
				fixtures.Alice,
				fixtures.Bob,
				storeAdd.New(
					fixtures.Alice.DID().String(),
					storeAddCaveats{Link: testLink},
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
			require.NoError(t, x)
			require.Equal(t, storeAdd.Can(), a.Capability().Can())
			require.Equal(t, fixtures.Alice.DID().String(), a.Capability().With())
			require.Equal(t, fixtures.Alice.DID(), a.Issuer().DID())
			require.Equal(t, fixtures.Bob.DID(), a.Audience().DID())
		})
	})

	t.Run("unauthorized", func(t *testing.T) {
		t.Run("expired invocation", func(t *testing.T) {
			exp := ucan.Now() - 5
			inv, err := invocation.Invoke(
				fixtures.Alice,
				fixtures.Service,
				storeAdd.New(
					fixtures.Alice.DID().String(),
					storeAddCaveats{Link: testLink},
				),
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
			inv, err := invocation.Invoke(
				fixtures.Alice,
				fixtures.Service,
				storeAdd.New(
					fixtures.Alice.DID().String(),
					storeAddCaveats{Link: testLink},
				),
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
			inv, err := invocation.Invoke(
				fixtures.Alice,
				fixtures.Service,
				storeAdd.New(
					fixtures.Alice.DID().String(),
					storeAddCaveats{Link: testLink},
				),
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
				fmt.Sprintf("  - Proof %s does not has a valid signature from %s", inv.Link(), fixtures.Alice.DID()),
			}, "\n")
			require.Equal(t, msg, x.Error())
		})

		t.Run("unknown capability", func(t *testing.T) {
			type storeWriteCaveats = storeAddCaveats

			inv, err := invocation.Invoke(
				fixtures.Alice,
				fixtures.Service,
				ucan.NewCapability(
					"store/write",
					fixtures.Alice.DID().String(),
					storeWriteCaveats{Link: testLink},
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

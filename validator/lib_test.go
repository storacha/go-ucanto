package validator

import (
	"fmt"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/core/result/failure"
	"github.com/storacha-network/go-ucanto/core/schema"
	"github.com/storacha-network/go-ucanto/principal"
	"github.com/storacha-network/go-ucanto/principal/ed25519/signer"
	"github.com/storacha-network/go-ucanto/principal/ed25519/verifier"
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
		func(claimed, delegated ucan.Capability[storeAddCaveats]) result.Result[result.Unit, failure.Failure] {
			if claimed.With() != delegated.With() {
				err := fmt.Errorf("Expected 'with: \"%s\"' instead got '%s'", delegated.With(), claimed.With())
				return result.Error[result.Unit](failure.FromError(err))
			}

			if delegated.Nb().Link != nil && delegated.Nb().Link != claimed.Nb().Link {
				var err error
				if claimed.Nb().Link == nil {
					err = fmt.Errorf("Link violates imposed %s constraint", delegated.Nb().Link)
				} else {
					err = fmt.Errorf("Link %s violates imposed %s constraint", claimed.Nb().Link, delegated.Nb().Link)
				}
				return result.Error[result.Unit](failure.FromError(err))
			}

			return result.Ok[result.Unit, failure.Failure](struct{}{})
		},
	)
}

func TestAccess(t *testing.T) {
	storeAdd := newStoreAddCapability(t)
	testLink := cidlink.Link{Cid: cid.MustParse("bafkqaaa")}
	validateAuthOk := func(auth Authorization[any]) result.Result[result.Unit, Revoked] {
		return result.Ok[result.Unit, Revoked](nil)
	}
	parseEdPrincipal := func(str string) (principal.Verifier, error) {
		return verifier.Parse(str)
	}

	t.Run("authorize self-issued invocation", func(t *testing.T) {
		inv, err := invocation.Invoke(
			alice,
			bob,
			storeAdd.New(
				alice.DID().String(),
				storeAddCaveats{Link: testLink},
			),
		)
		require.NoError(t, err)

		context := NewValidationContext(
			service.Verifier(),
			storeAdd,
			IsSelfIssued,
			validateAuthOk,
			ProofUnavailable,
			parseEdPrincipal,
			FailDIDKeyResolution,
		)

		res, err := Access(inv, context)
		require.NoError(t, err)

		result.MatchResultR0(
			res,
			func(a Authorization[storeAddCaveats]) {
				require.Equal(t, storeAdd.Can(), a.Capability().Can())
				require.Equal(t, alice.DID().String(), a.Capability().With())
				require.Equal(t, alice.DID(), a.Issuer().DID())
				require.Equal(t, bob.DID(), a.Audience().DID())
			},
			func(x Unauthorized) {
				t.Fatalf("unexpected unauthorized failure: %s", x)
			},
		)
	})
}

func TestIsSelfIssued(t *testing.T) {
	alice, err := signer.Generate()
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}
	bob, err := signer.Generate()
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	cap := ucan.NewCapability("upload/add", alice.DID().String(), struct{}{})

	canIssue := IsSelfIssued(cap, alice.DID())
	if canIssue == false {
		t.Fatal("capability self issued by alice")
	}

	canIssue = IsSelfIssued(cap, bob.DID())
	if canIssue == true {
		t.Fatal("capability not self issued by bob")
	}
}

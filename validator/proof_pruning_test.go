package validator

import (
	"testing"

	goipld "github.com/ipld/go-ipld-prime"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/schema"
	"github.com/storacha/go-ucanto/testing/fixtures"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/stretchr/testify/require"
)

// testFetchCaveats is used by the test/fetch capability. The URL field is
// required (not optional) and has a meaningful zero value (""), which
// demonstrates that NoCaveats replacement works for non-optional fields too.
type testFetchCaveats struct {
	Url string
}

func (tfc testFetchCaveats) ToIPLD() (ipld.Node, error) {
	return ipld.WrapWithRecovery(&tfc, testFetchTyp.TypeByName("TestFetchCaveats"))
}

var testFetchTyp = helpers.Must(goipld.LoadSchemaBytes([]byte(`
	type TestFetchCaveats struct {
		url String
	}
`)))

var testFetch = NewCapability(
	"test/fetch",
	schema.DIDString(),
	schema.Struct[testFetchCaveats](testFetchTyp.TypeByName("TestFetchCaveats"), nil),
	DefaultDerives[testFetchCaveats],
)

// TestNewProofPrunerNoCaveats verifies that NewProofPruner handles capabilities
// whose Nb is ucan.NoCaveats. NoCaveats encodes as a schema-less empty map {}
// in IPLD, which the capability parser's Nb schema reader cannot rebind to the
// specific Caveats type. The pruner must replace NoCaveats with the zero value
// of Caveats before building the draft delegation used for proof chain
// validation.
func TestNewProofPrunerNoCaveats(t *testing.T) {
	// Service delegates test/fetch to Alice with NoCaveats – the common
	// pattern callers use when they have no specific caveats to attach.
	proof, err := delegation.Delegate(
		fixtures.Service,
		fixtures.Alice,
		[]ucan.Capability[ucan.NoCaveats]{
			ucan.NewCapability(testFetch.Can(), fixtures.Service.DID().String(), ucan.NoCaveats{}),
		},
		delegation.WithNoExpiration(),
	)
	require.NoError(t, err)

	pruner := NewProofPruner(fixtures.Service.Verifier(), testFetch)

	// Alice re-delegates to Bob with NoCaveats and proof pruning enabled.
	// Before the fix the pruner would error: it built a draft delegation with
	// NoCaveats, which encodes as {} in IPLD. When the validation logic then
	// tried to parse that capability it called the TestFetchCaveats schema
	// reader on {}, which failed because "url" is a required field.
	dlg, err := delegation.Delegate(
		fixtures.Alice,
		fixtures.Bob,
		[]ucan.Capability[ucan.NoCaveats]{
			ucan.NewCapability(testFetch.Can(), fixtures.Service.DID().String(), ucan.NoCaveats{}),
		},
		delegation.WithNoExpiration(),
		delegation.WithProof(delegation.FromDelegation(proof)),
		delegation.WithProofPruning(pruner),
	)
	require.NoError(t, err)
	require.NotNil(t, dlg)
	// The pruner should retain exactly the one proof that authorises Alice.
	require.Len(t, dlg.Proofs(), 1)
}

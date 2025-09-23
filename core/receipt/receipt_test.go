package receipt

import (
	"fmt"
	"io"
	"slices"
	"testing"

	ipldprime "github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/ipld/block"
	"github.com/storacha/go-ucanto/core/receipt/fx"
	"github.com/storacha/go-ucanto/core/receipt/ran"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/core/result/ok"
	"github.com/storacha/go-ucanto/testing/fixtures"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/stretchr/testify/require"
)

func TestEffects(t *testing.T) {
	ran := ran.FromLink(helpers.RandomCID())
	out := result.Ok[ok.Unit, ipld.Builder](ok.Unit{})

	t.Run("as links", func(t *testing.T) {
		f0 := fx.FromLink(helpers.RandomCID())
		f1 := fx.FromLink(helpers.RandomCID())
		j := fx.FromLink(helpers.RandomCID())

		receipt, err := Issue(fixtures.Alice, out, ran, WithFork(f0, f1), WithJoin(j))
		require.NoError(t, err)

		effects := receipt.Fx()
		require.True(t, slices.ContainsFunc(effects.Fork(), func(f fx.Effect) bool {
			return f.Link().String() == f0.Link().String()
		}))
		require.True(t, slices.ContainsFunc(effects.Fork(), func(f fx.Effect) bool {
			return f.Link().String() == f1.Link().String()
		}))
		require.Equal(t, effects.Join().Link(), j.Link())
	})

	t.Run("as invocations", func(t *testing.T) {
		i0, err := invocation.Invoke(
			fixtures.Alice,
			fixtures.Bob,
			ucan.NewCapability("fx/0", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		)
		require.NoError(t, err)
		i1, err := invocation.Invoke(
			fixtures.Alice,
			fixtures.Mallory,
			ucan.NewCapability("fx/1", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		)
		require.NoError(t, err)
		i2, err := invocation.Invoke(
			fixtures.Mallory,
			fixtures.Bob,
			ucan.NewCapability("fx/2", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		)
		require.NoError(t, err)

		f0 := fx.FromInvocation(i0)
		f1 := fx.FromInvocation(i1)
		j := fx.FromInvocation(i2)

		receipt, err := Issue(fixtures.Alice, out, ran, WithFork(f0, f1), WithJoin(j))
		require.NoError(t, err)

		effects := receipt.Fx()
		require.True(t, slices.ContainsFunc(effects.Fork(), func(f fx.Effect) bool {
			return f.Link().String() == f0.Link().String()
		}))
		require.True(t, slices.ContainsFunc(effects.Fork(), func(f fx.Effect) bool {
			return f.Link().String() == f1.Link().String()
		}))
		require.Equal(t, effects.Join().Link(), j.Link())

		for _, effect := range effects.Fork() {
			_, ok := effect.Invocation()
			require.True(t, ok)
		}

		_, ok := effects.Join().Invocation()
		require.True(t, ok)
	})
}

var someTS = mustLoadTS()

func mustLoadTS() *schema.TypeSystem {
	someSchema := []byte(`
		type someOkType struct {
			someOkProperty String
		}

		type someErrorType struct {
			someErrorProperty String
		}
	`)
	ts, err := ipldprime.LoadSchemaBytes(someSchema)
	if err != nil {
		panic(fmt.Errorf("loading some schema: %w", err))
	}

	return ts
}

type someOkType struct {
	SomeOkProperty string
}

func (s someOkType) ToIPLD() (ipld.Node, error) {
	return ipld.WrapWithRecovery(&s, someTS.TypeByName("someOkType"))
}

type someErrorType struct {
	SomeErrorProperty string
}

func (s someErrorType) ToIPLD() (ipld.Node, error) {
	return ipld.WrapWithRecovery(&s, someTS.TypeByName("someErrorType"))
}

func TestIssue(t *testing.T) {
	t.Run("ran as invocation", func(t *testing.T) {
		inv, err := invocation.Invoke(
			fixtures.Alice,
			fixtures.Bob,
			ucan.NewCapability("ran/invoke", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		)
		require.NoError(t, err)
		ran := ran.FromInvocation(inv)

		out := result.Ok[someOkType, someErrorType](someOkType{SomeOkProperty: "some ok value"})

		issuedRcpt, err := Issue(fixtures.Alice, out, ran)
		require.NoError(t, err)

		ranInv, ok := issuedRcpt.Ran().Invocation()
		require.True(t, ok)
		require.Equal(t, inv.Link().String(), ranInv.Link().String())
	})

	t.Run("ran as link", func(t *testing.T) {
		inv, err := invocation.Invoke(
			fixtures.Alice,
			fixtures.Bob,
			ucan.NewCapability("ran/invoke", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		)
		require.NoError(t, err)
		ran := ran.FromLink(inv.Link())

		out := result.Ok[someOkType, someErrorType](someOkType{SomeOkProperty: "some ok value"})

		issuedRcpt, err := Issue(fixtures.Alice, out, ran)
		require.NoError(t, err)

		ranInv, ok := issuedRcpt.Ran().Invocation()
		require.False(t, ok)
		require.Nil(t, ranInv)

		ranInvLink := issuedRcpt.Ran().Link()
		require.NotNil(t, ranInvLink)
		require.Equal(t, inv.Link().String(), ranInvLink.String())
	})
}

func TestArchiveExtract(t *testing.T) {
	prf, err := delegation.Delegate(
		fixtures.Alice,
		fixtures.Bob,
		[]ucan.Capability[ucan.NoCaveats]{
			ucan.NewCapability("test/proof", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		},
	)
	require.NoError(t, err)

	inv, err := invocation.Invoke(
		fixtures.Alice,
		fixtures.Bob,
		ucan.NewCapability("test/attach", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
	)
	require.NoError(t, err)

	ran := ran.FromInvocation(inv)
	ok := someOkType{SomeOkProperty: "some ok value"}
	rcpt, err := Issue(
		fixtures.Alice,
		result.Ok[someOkType, someErrorType](ok),
		ran,
		WithProofs(delegation.Proofs{
			delegation.FromDelegation(prf),
			// include an absent proof to prove things don't break - PUN INTENDED
			delegation.FromLink(helpers.RandomCID()),
		}),
	)
	require.NoError(t, err)

	archive := rcpt.Archive()

	archiveBytes, err := io.ReadAll(archive)
	require.NoError(t, err)
	extracted, err := Extract(archiveBytes)
	require.NoError(t, err)

	var rcptBlks []ipld.Block
	for b, err := range rcpt.Export() {
		require.NoError(t, err)
		rcptBlks = append(rcptBlks, b)
	}

	var extractedBlks []ipld.Block
	for b, err := range extracted.Export() {
		require.NoError(t, err)
		extractedBlks = append(extractedBlks, b)
	}

	require.Equal(t, len(rcptBlks), len(extractedBlks))
	for i, b := range rcptBlks {
		require.Equal(t, b.Link().String(), extractedBlks[i].Link().String())
	}
}

func TestExport(t *testing.T) {
	prf, err := delegation.Delegate(
		fixtures.Alice,
		fixtures.Bob,
		[]ucan.Capability[ucan.NoCaveats]{
			ucan.NewCapability("test/proof", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		},
	)
	require.NoError(t, err)

	inv, err := invocation.Invoke(
		fixtures.Alice,
		fixtures.Bob,
		ucan.NewCapability("test/export", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
	)
	require.NoError(t, err)

	ran := ran.FromInvocation(inv)
	ok := someOkType{SomeOkProperty: "some ok value"}
	rcpt, err := Issue(
		fixtures.Alice,
		result.Ok[someOkType, someErrorType](ok),
		ran,
		WithProofs(delegation.Proofs{
			delegation.FromDelegation(prf),
			// include an absent proof to prove things don't break - PUN INTENDED
			delegation.FromLink(helpers.RandomCID()),
		}),
	)
	require.NoError(t, err)

	bs, err := blockstore.NewBlockStore()
	require.NoError(t, err)

	var blks []ipld.Block
	for b, err := range rcpt.Blocks() {
		require.NoError(t, err)
		require.NoError(t, bs.Put(b))
		blks = append(blks, b)
	}
	require.Len(t, blks, 3)
	require.True(t, slices.ContainsFunc(blks, func(b ipld.Block) bool {
		return b.Link().String() == prf.Link().String()
	}))
	require.True(t, slices.ContainsFunc(blks, func(b ipld.Block) bool {
		return b.Link().String() == inv.Link().String()
	}))
	require.True(t, slices.ContainsFunc(blks, func(b ipld.Block) bool {
		return b.Link().String() == rcpt.Root().Link().String()
	}))

	// add an additional block to the blockstore that is not linked to by the receipt
	otherblk := block.NewBlock(helpers.RandomCID(), helpers.RandomBytes(32))
	err = bs.Put(otherblk)
	require.NoError(t, err)

	// reinstantiate receipt with our new blockstore
	rcpt, err = NewAnyReceipt(rcpt.Root().Link(), bs)
	require.NoError(t, err)

	var exblks []ipld.Block
	// export the receipt from the blockstore
	for b, err := range rcpt.Export() {
		require.NoError(t, err)
		exblks = append(exblks, b)
	}

	// expect exblks to have the same blocks in the same order and it should not
	// include otherblk
	require.Len(t, exblks, len(blks))
	for i, b := range blks {
		require.Equal(t, b.Link().String(), exblks[i].Link().String())
	}

	// expect rcpt.Blocks() to include otherblk though...
	var blklnks []string
	for b, err := range rcpt.Blocks() {
		require.NoError(t, err)
		blklnks = append(blklnks, b.Link().String())
	}
	require.Contains(t, blklnks, otherblk.Link().String())
}

func TestWithInvocation(t *testing.T) {
	inv, err := invocation.Invoke(
		fixtures.Alice,
		fixtures.Bob,
		ucan.NewCapability("ran/invoke", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
	)
	require.NoError(t, err)

	out := result.Ok[someOkType, someErrorType](someOkType{SomeOkProperty: "some ok value"})

	t.Run("adds invocation to receipt without one", func(t *testing.T) {
		issuedRcpt, err := Issue(fixtures.Alice, out, ran.FromLink(inv.Link()))
		require.NoError(t, err)

		ranInv, ok := issuedRcpt.Ran().Invocation()
		require.False(t, ok)
		require.Nil(t, ranInv)

		fullRcpt, err := issuedRcpt.WithInvocation(inv)
		require.NoError(t, err)

		fullRanInv, ok := fullRcpt.Ran().Invocation()
		require.True(t, ok)
		require.Equal(t, inv.Link().String(), fullRanInv.Link().String())

		// the original receipt's blockstore should be unchanged
		issuedRcptNumBlocks := 0
		for range issuedRcpt.Blocks() {
			issuedRcptNumBlocks++
		}
		fullRcptNumBlocks := 0
		for range fullRcpt.Blocks() {
			fullRcptNumBlocks++
		}
		require.True(t, fullRcptNumBlocks > issuedRcptNumBlocks)
	})

	t.Run("doesn't fail if receipt already has invocation and invocations match", func(t *testing.T) {
		issuedRcpt, err := Issue(fixtures.Alice, out, ran.FromInvocation(inv))
		require.NoError(t, err)

		ranInv, ok := issuedRcpt.Ran().Invocation()
		require.True(t, ok)
		require.Equal(t, inv.Link().String(), ranInv.Link().String())

		_, err = issuedRcpt.WithInvocation(inv)
		require.NoError(t, err)
	})

	t.Run("fails if receipt invocations don't match", func(t *testing.T) {
		issuedRcpt, err := Issue(fixtures.Alice, out, ran.FromLink(inv.Link()))
		require.NoError(t, err)

		inv2, err := invocation.Invoke(
			fixtures.Alice,
			fixtures.Service, // previous invocation's audience is Bob
			ucan.NewCapability("ran/invoke", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		)
		require.NoError(t, err)

		_, err = issuedRcpt.WithInvocation(inv2)
		require.Error(t, err)
	})
}

func TestAnyReceiptReader(t *testing.T) {
	ranInv, err := invocation.Invoke(
		fixtures.Alice,
		fixtures.Bob,
		ucan.NewCapability("ran/invoke", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
	)
	require.NoError(t, err)
	ran := ran.FromInvocation(ranInv)

	out := result.Ok[someOkType, someErrorType](someOkType{SomeOkProperty: "some ok value"})

	issuedRcpt, err := Issue(fixtures.Alice, out, ran)
	require.NoError(t, err)

	reader := NewAnyReceiptReader()
	var anyRcpt AnyReceipt
	anyRcpt, err = reader.Read(issuedRcpt.Root().Link(), issuedRcpt.Blocks())
	require.NoError(t, err)

	concreteRcpt, err := Rebind[*someOkType, *someErrorType](anyRcpt, someTS.TypeByName("someOkType"), someTS.TypeByName("someErrorType"))
	require.NoError(t, err)

	someOk, someErr := result.Unwrap(concreteRcpt.Out())
	require.Equal(t, "some ok value", someOk.SomeOkProperty)
	require.Nil(t, someErr)
}

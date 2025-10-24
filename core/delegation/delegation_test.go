package delegation

import (
	_ "embed"
	"fmt"
	"slices"
	"testing"

	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/ipld/block"
	"github.com/storacha/go-ucanto/testing/fixtures"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/stretchr/testify/require"
)

func TestExport(t *testing.T) {
	prf, err := Delegate(
		fixtures.Alice,
		fixtures.Bob,
		[]ucan.Capability[ucan.NoCaveats]{
			ucan.NewCapability("test/proof", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		},
	)
	require.NoError(t, err)
	dlg, err := Delegate(
		fixtures.Bob,
		fixtures.Mallory,
		[]ucan.Capability[ucan.NoCaveats]{
			ucan.NewCapability("test/proof", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		},
		WithProof(
			FromDelegation(prf),
			// include an absent proof to prove things don't break - PUN INTENDED
			FromLink(helpers.RandomCID()),
		),
	)
	require.NoError(t, err)

	bs, err := blockstore.NewBlockStore()
	require.NoError(t, err)

	var blks []ipld.Block
	for b, err := range dlg.Blocks() {
		require.NoError(t, err)
		require.NoError(t, bs.Put(b))
		blks = append(blks, b)
	}
	require.Len(t, blks, 2)
	require.True(t, slices.ContainsFunc(blks, func(b ipld.Block) bool {
		return b.Link().String() == prf.Link().String()
	}))
	require.True(t, slices.ContainsFunc(blks, func(b ipld.Block) bool {
		return b.Link().String() == dlg.Link().String()
	}))

	// add an additional block to the blockstore that is not linked to by the
	// delegation
	otherblk := block.NewBlock(helpers.RandomCID(), helpers.RandomBytes(32))
	err = bs.Put(otherblk)
	require.NoError(t, err)

	// reinstantiate delegation with our new blockstore
	dlg, err = NewDelegationView(dlg.Link(), bs)
	require.NoError(t, err)

	var exblks []ipld.Block
	// export the delegation from the blockstore
	for b, err := range dlg.Export() {
		require.NoError(t, err)
		exblks = append(exblks, b)
	}

	// expect exblks to have the same blocks in the same order and it should not
	// include otherblk
	require.Len(t, exblks, len(blks))
	for i, b := range blks {
		require.Equal(t, b.Link().String(), exblks[i].Link().String())
	}

	// expect dlg.Blocks() to include otherblk though...
	var blklnks []string
	for b, err := range dlg.Blocks() {
		require.NoError(t, err)
		blklnks = append(blklnks, b.Link().String())
	}
	require.Contains(t, blklnks, otherblk.Link().String())
}

func TestExportOmitsProofs(t *testing.T) {
	prf, err := Delegate(
		fixtures.Alice,
		fixtures.Bob,
		[]ucan.Capability[ucan.NoCaveats]{
			ucan.NewCapability("test/proof", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		},
	)
	require.NoError(t, err)
	dlg, err := Delegate(
		fixtures.Bob,
		fixtures.Mallory,
		[]ucan.Capability[ucan.NoCaveats]{
			ucan.NewCapability("test/proof", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		},
		WithProof(
			FromDelegation(prf),
			// include an absent proof to prove things don't break - PUN INTENDED
			FromLink(helpers.RandomCID()),
		),
	)
	require.NoError(t, err)

	blks := map[string]struct{}{}
	for b, err := range dlg.Export() {
		require.NoError(t, err)
		blks[b.Link().String()] = struct{}{}
	}

	exblks := map[string]struct{}{}
	// export the delegation from the blockstore, excluding the proof
	for b, err := range dlg.Export(WithOmitProof(prf.Link())) {
		require.NoError(t, err)
		exblks[b.Link().String()] = struct{}{}
	}

	require.Contains(t, blks, prf.Link().String())
	require.NotContains(t, exblks, prf.Link().String())
}

func TestAttach(t *testing.T) {
	dlg, err := Delegate(
		fixtures.Alice,
		fixtures.Bob,
		[]ucan.Capability[ucan.NoCaveats]{
			ucan.NewCapability("test/attach", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		},
	)
	require.NoError(t, err)

	blk := block.NewBlock(helpers.RandomCID(), helpers.RandomBytes(32))
	err = dlg.Attach(blk)
	require.NoError(t, err)

	var blklnks []string
	for b, err := range dlg.Blocks() {
		require.NoError(t, err)
		blklnks = append(blklnks, b.Link().String())
	}
	require.Contains(t, blklnks, blk.Link().String())
}

func TestFormatParse(t *testing.T) {
	dlg, err := Delegate(
		fixtures.Alice,
		fixtures.Bob,
		[]ucan.Capability[ucan.NoCaveats]{
			ucan.NewCapability("test/proof", fixtures.Alice.DID().String(), ucan.NoCaveats{}),
		},
	)
	require.NoError(t, err)

	formatted, err := Format(dlg)
	require.NoError(t, err)

	fmt.Println(formatted)

	parsed, err := Parse(formatted)
	require.NoError(t, err)

	require.Equal(t, dlg.Link(), parsed.Link())
}

//go:embed delegationnonb.car
var delegationnonb []byte

// delegationnonb.car is an archived delegation with a capability with no `nb`.
// This is (currently) difficult to accomplish in Go, but perfectly common in
// JS, leading to delegations from a server which a Go client chokes on (hence
// this test). The delegation was generated with JS like the following:

// const client = await getClient()
//
// const delegation = await delegate({
//   issuer: client.agent.issuer,
//   audience: DID.parse('did:example:alice'),
//   capabilities: [
//     {
//       with: 'did:key:123456789',
//       can: 'do/something',
//     },
//   ],
// })
//
// const res = await delegation.archive()
// fs.writeFileSync('delegationnonb.car', res.ok)

func TestParseNoNb(t *testing.T) {
	// An archived delegation with a capability with no `nb`
	dlg, err := Extract(delegationnonb)
	require.NoError(t, err)
	require.Equal(t, "did:key:z6MkpveRpPySqSVXyhAmWbyQLdY9w5noKr1Ff2MX8P9htje9", dlg.Issuer().DID().String())
	require.Equal(t, "did:example:alice", dlg.Audience().DID().String())
	require.Len(t, dlg.Capabilities(), 1)
	require.Equal(t, "do/something", dlg.Capabilities()[0].Can())
	require.Equal(t, "did:key:123456789", dlg.Capabilities()[0].With())
	require.Equal(t, nil, dlg.Capabilities()[0].Nb())
}

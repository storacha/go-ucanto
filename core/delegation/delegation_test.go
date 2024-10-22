package delegation

import (
	"fmt"
	"testing"

	"github.com/storacha/go-ucanto/core/ipld/block"
	"github.com/storacha/go-ucanto/testing/fixtures"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/stretchr/testify/require"
)

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

	found := false
	for b, err := range dlg.Blocks() {
		require.NoError(t, err)
		if b.Link().String() == blk.Link().String() {
			found = true
			break
		}
	}
	require.True(t, found)
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

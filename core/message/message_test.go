package message

import (
	"testing"

	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/receipt"
	"github.com/storacha/go-ucanto/core/receipt/ran"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/core/result/ok"
	"github.com/storacha/go-ucanto/testing/fixtures"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/stretchr/testify/require"
)

func TestMessageReceipts(t *testing.T) {
	t.Run("no receipts", func(t *testing.T) {
		msg, err := Build([]invocation.Invocation{}, []receipt.AnyReceipt{})
		require.NoError(t, err)

		rcpts := msg.Receipts()
		require.Len(t, rcpts, 0)
	})

	t.Run("one receipt", func(t *testing.T) {
		rcpt, err := receipt.Issue(
			fixtures.Alice,
			result.Ok[ok.Unit, ipld.Builder](ok.Unit{}),
			ran.FromLink(helpers.RandomCID()),
		)
		require.NoError(t, err)

		msg, err := Build([]invocation.Invocation{}, []receipt.AnyReceipt{rcpt})
		require.NoError(t, err)

		rcpts := msg.Receipts()
		require.Len(t, rcpts, 1)

		r, ok, err := msg.Receipt(rcpts[0])
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, rcpt.Root().Link().String(), r.Root().Link().String())
	})
}

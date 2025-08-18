package message

import (
	"fmt"
	"testing"

	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/testing/fixtures"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	inv, err := invocation.Invoke(
		fixtures.Alice,
		fixtures.Service,
		ucan.NewCapability(
			"test/invoke",
			fixtures.Alice.DID().String(),
			ucan.NoCaveats{},
		),
	)
	require.NoError(t, err)

	msg, err := message.Build([]invocation.Invocation{inv}, nil)
	require.NoError(t, err)

	s, err := EncodeHeader(msg)
	require.NoError(t, err)

	fmt.Printf("X-Agent-Message: %s (%d bytes)\n", s, len(s))

	_, err = DecodeHeader(s)
	require.NoError(t, err)
}

func TestEncodeHeaderTooLarge(t *testing.T) {
	inv, err := invocation.Invoke(
		fixtures.Alice,
		fixtures.Service,
		ucan.NewCapability(
			"test/invoke",
			fixtures.Alice.DID().String(),
			ucan.NoCaveats{},
		),
	)
	require.NoError(t, err)

	msg, err := message.Build([]invocation.Invocation{inv}, nil)
	require.NoError(t, err)

	s, err := EncodeHeader(msg, WithMaxSize(1))
	require.Empty(t, s)
	require.ErrorIs(t, err, ErrHeaderTooLarge)
}

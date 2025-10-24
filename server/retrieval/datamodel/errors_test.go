package datamodel

import (
	"testing"

	"github.com/ipld/go-ipld-prime"
	"github.com/storacha/go-ucanto/core/ipld/codec/json"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/stretchr/testify/require"
)

func TestAgentMessageInvocationError(t *testing.T) {
	t.Run("encode decode", func(t *testing.T) {
		mdl := AgentMessageInvocationErrorModel{
			Name:    "AgentMessageInvocationError",
			Message: "boom",
		}
		data, err := json.Encode(&mdl, AgentMessageInvocationErrorType())
		require.NoError(t, err)
		var decoded AgentMessageInvocationErrorModel
		err = json.Decode(data, &decoded, AgentMessageInvocationErrorType())
		require.NoError(t, err)
		require.Equal(t, mdl.Name, decoded.Name)
		require.Equal(t, mdl.Message, decoded.Message)
	})
}

func TestMissingProofs(t *testing.T) {
	t.Run("encode decode", func(t *testing.T) {
		prf := helpers.RandomCID()
		mdl := MissingProofsModel{
			Name:    "MissingProofs",
			Message: "boom",
			Proofs:  []ipld.Link{prf},
		}
		data, err := json.Encode(&mdl, MissingProofsType())
		require.NoError(t, err)
		var decoded MissingProofsModel
		err = json.Decode(data, &decoded, MissingProofsType())
		require.NoError(t, err)
		require.Equal(t, mdl.Name, decoded.Name)
		require.Equal(t, mdl.Message, decoded.Message)
		require.Len(t, decoded.Proofs, 1)
		require.Equal(t, prf.String(), decoded.Proofs[0].String())
	})
}

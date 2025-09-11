package retrieval

import (
	"github.com/storacha/go-ucanto/core/ipld"
	rdm "github.com/storacha/go-ucanto/server/retrieval/datamodel"
)

type AgentMessageInvocationCountError struct{}

func (amie AgentMessageInvocationCountError) Error() string {
	return "Agent Message is required to have a single invocation."
}

func (amie AgentMessageInvocationCountError) Name() string {
	return "AgentMessageInvocationError"
}

func (amie AgentMessageInvocationCountError) ToIPLD() (ipld.Node, error) {
	mdl := rdm.AgentMessageInvocationErrorModel{
		Name:    amie.Name(),
		Message: amie.Error(),
	}
	return ipld.WrapWithRecovery(&mdl, rdm.AgentMessageInvocationErrorType())
}

func NewAgentMessageInvocationCountError() AgentMessageInvocationCountError {
	return AgentMessageInvocationCountError{}
}

type MissingProofs struct {
	proofs []ipld.Link
}

func (mpe MissingProofs) Error() string {
	return "proofs were missing, resubmit the invocation with the requested proofs"
}

func (mpe MissingProofs) Name() string {
	return "MissingProofs"
}

func (mpe MissingProofs) Proofs() []ipld.Link {
	return mpe.proofs
}

func (mpe MissingProofs) ToIPLD() (ipld.Node, error) {
	mdl := rdm.MissingProofsModel{
		Name:    mpe.Name(),
		Message: mpe.Error(),
		Proofs:  mpe.Proofs(),
	}
	return ipld.WrapWithRecovery(&mdl, rdm.MissingProofsType())
}

func NewMissingProofsError(proofs []ipld.Link) MissingProofs {
	return MissingProofs{proofs}
}

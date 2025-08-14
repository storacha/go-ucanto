package retrieval

import (
	"fmt"
	"strings"

	"github.com/storacha/go-ucanto/core/ipld"
	rdm "github.com/storacha/go-ucanto/server/retrieval/datamodel"
)

type AgentMessageInvocationError struct{}

func (amie AgentMessageInvocationError) Error() string {
	return "Agent Message is required to have a single invocation."
}

func (amie AgentMessageInvocationError) Name() string {
	return "AgentMessageInvocationError"
}

func (amie AgentMessageInvocationError) ToIPLD() (ipld.Node, error) {
	mdl := rdm.AgentMessageInvocationErrorModel{
		Name:    amie.Name(),
		Message: amie.Error(),
	}
	return ipld.WrapWithRecovery(&mdl, rdm.AgentMessageInvocationErrorType())
}

func NewAgentMessageInvocationError() AgentMessageInvocationError {
	return AgentMessageInvocationError{}
}

type MissingProofs struct {
	proofs []ipld.Link
}

func (mpe MissingProofs) Error() string {
	var links []string
	for _, p := range mpe.proofs {
		links = append(links, p.String())
	}
	return fmt.Sprintf("Missing proofs: %s", strings.Join(links, ", "))
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

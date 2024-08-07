package validator

import (
	"github.com/storacha-network/go-ucanto/did"
	"github.com/storacha-network/go-ucanto/ucan"
)

func IsSelfIssued[Caveats any](capability ucan.Capability[Caveats], issuer did.DID) bool {
	return capability.With() == issuer.DID().String()
}

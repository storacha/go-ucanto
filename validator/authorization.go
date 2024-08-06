package validator

import (
	"github.com/web3-storage/go-ucanto/core/result"
	"github.com/web3-storage/go-ucanto/did"
	"github.com/web3-storage/go-ucanto/ucan"
)

type Revoked interface {
	result.Failure
}

type Authorization interface{}

func IsSelfIssued[Caveats any](capability ucan.Capability[Caveats], issuer did.DID) bool {
	return capability.With() == issuer.DID().String()
}

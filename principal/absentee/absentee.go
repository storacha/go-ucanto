package absentee

import (
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/storacha/go-ucanto/ucan/crypto/signature"
)

type absentee struct {
	id did.DID
}

func (a absentee) DID() did.DID {
	return a.id
}

func (a absentee) Sign(msg []byte) signature.SignatureView {
	return signature.NewSignatureView(signature.NewNonStandard(a.SignatureAlgorithm(), []byte{}))
}

func (a absentee) SignatureAlgorithm() string {
	return ""
}

func (a absentee) SignatureCode() uint64 {
	return signature.NON_STANDARD
}

// From creates a special type of signer that produces an absent signature,
// which signals that verifier needs to verify authorization interactively.
func From(id did.DID) ucan.Signer {
	return absentee{id}
}

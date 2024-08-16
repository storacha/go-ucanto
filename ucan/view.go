package ucan

import (
	"fmt"

	"github.com/storacha-network/go-ucanto/did"
	"github.com/storacha-network/go-ucanto/ucan/crypto/signature"
	udm "github.com/storacha-network/go-ucanto/ucan/datamodel/ucan"
)

type UCAN interface {
	// Issuer is the signer of the UCAN.
	Issuer() Principal
	// Audience is the principal delegated to.
	Audience() Principal
	// Version is the spec version the UCAN conforms to.
	Version() Version
	// Capabilities are claimed abilities that can be performed on a resource.
	Capabilities() []Capability[any]
	// Expiration is the time in seconds since the Unix epoch that the UCAN
	// becomes invalid.
	Expiration() UTCUnixTimestamp
	// NotBefore is the time in seconds since the Unix epoch that the UCAN
	// becomes valid.
	NotBefore() UTCUnixTimestamp
	// Nonce is a randomly generated string to provide a unique
	Nonce() Nonce
	// Facts are arbitrary facts and proofs of knowledge.
	Facts() []Fact
	// Proofs of delegation.
	Proofs() []Link
	// Signature of the UCAN issuer.
	Signature() signature.SignatureView
}

// View represents a decoded "view" of a UCAN that can be used in your
// domain logic, etc.
type View interface {
	UCAN
	// Model references the underlying IPLD datamodel instance.
	Model() *udm.UCANModel
}

type ucanView struct {
	model *udm.UCANModel
}

var _ View = (*ucanView)(nil)

func (v *ucanView) Audience() Principal {
	did, err := did.Decode(v.model.Aud)
	if err != nil {
		fmt.Printf("Error: decoding audience DID: %s\n", err)
	}
	return did
}

func (v *ucanView) Capabilities() []Capability[any] {
	caps := []Capability[any]{}
	for _, c := range v.model.Att {
		caps = append(caps, NewCapability[any](c.Can, c.With, c.Nb))
	}
	return caps
}

func (v *ucanView) Expiration() uint64 {
	return v.model.Exp
}

func (v *ucanView) Facts() []map[string]any {
	facts := []map[string]any{}
	for _, f := range v.model.Fct {
		fact := map[string]any{}
		for k, v := range f.Values {
			fact[k] = v
		}
		facts = append(facts, fact)
	}
	return facts
}

func (v *ucanView) Issuer() Principal {
	did, err := did.Decode(v.model.Iss)
	if err != nil {
		fmt.Printf("decoding issuer DID: %s\n", err)
	}
	return did
}

func (v *ucanView) Model() *udm.UCANModel {
	return v.model
}

func (v *ucanView) Nonce() string {
	if v.model.Nnc == nil {
		return ""
	}
	return *v.model.Nnc
}

func (v *ucanView) NotBefore() uint64 {
	if v.model.Nbf == nil {
		return 0
	}
	return *v.model.Nbf
}

func (v *ucanView) Proofs() []Link {
	return v.model.Prf
}

func (v *ucanView) Signature() signature.SignatureView {
	s := signature.Decode(v.model.S)
	return signature.NewSignatureView(s)
}

func (v *ucanView) Version() string {
	return v.model.V
}

// NewUCAN creates a UCAN view from the underlying data model. Please note
// that this function does no verification of the model and it is callers
// responsibility to ensure that:
//
//  1. Data model is correct contains all the field etc.
//  2. Payload of the signature will match paylodad when model is serialized
//     with DAG-JSON.
//
// In other words you should never use this function unless you've parsed or
// decoded a valid UCAN and want to wrap it into a view.
func NewUCAN(model *udm.UCANModel) (View, error) {
	return &ucanView{model}, nil
}

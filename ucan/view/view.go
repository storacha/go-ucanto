package view

import (
	"github.com/alanshaw/go-ucanto/ucan"
	"github.com/alanshaw/go-ucanto/ucan/crypto/signature"
	"github.com/alanshaw/go-ucanto/ucan/datamodel"
)

// UCANView represents a decoded "view" of a UCAN as a JS object that can be
// used in your domain logic, etc.
type UCANView interface {
	Code() ucan.Code
	Model() datamodel.UCANModel

	Issuer() ucan.Principal
	Audience() ucan.Principal

	Version() ucan.Version

	Capabilities() []ucan.Capability[any]

	Expiration() ucan.UTCUnixTimestamp
	NotBefore() ucan.UTCUnixTimestamp
	Nonce() ucan.Nonce
	Facts() []ucan.Fact
	Proofs() []ucan.Link

	Signature() signature.SignatureView

	Encode() []byte
}

type ucanView struct {
}

var _ UCANView = (*ucanView)(nil)

// Audience implements UCANView.
func (*ucanView) Audience() ucan.Principal {
	panic("unimplemented")
}

// Capabilities implements UCANView.
func (*ucanView) Capabilities() []ucan.Capability[any] {
	panic("unimplemented")
}

// Code implements UCANView.
func (*ucanView) Code() uint64 {
	panic("unimplemented")
}

// Encode implements UCANView.
func (*ucanView) Encode() []byte {
	panic("unimplemented")
}

// Expiration implements UCANView.
func (*ucanView) Expiration() uint64 {
	panic("unimplemented")
}

// Facts implements UCANView.
func (*ucanView) Facts() []map[string]any {
	panic("unimplemented")
}

// Issuer implements UCANView.
func (*ucanView) Issuer() ucan.Principal {
	panic("unimplemented")
}

// Model implements UCANView.
func (*ucanView) Model() datamodel.UCANModel {
	panic("unimplemented")
}

// Nonce implements UCANView.
func (*ucanView) Nonce() string {
	panic("unimplemented")
}

// NotBefore implements UCANView.
func (*ucanView) NotBefore() uint64 {
	panic("unimplemented")
}

// Proofs implements UCANView.
func (*ucanView) Proofs() []ucan.Link {
	panic("unimplemented")
}

// Signature implements UCANView.
func (*ucanView) Signature() signature.SignatureView {
	panic("unimplemented")
}

// Version implements UCANView.
func (*ucanView) Version() string {
	panic("unimplemented")
}

// NewUCANView creates a UCAN view from the underlying data model. Please note
// that this function does no verification of the model and it is callers
// responsibility to ensure that:
//
//  1. Data model is correct contains all the field etc.
//  2. Payload of the signature will match paylodad when model is serialized
//     with DAG-JSON.
//
// In other words you should never use this function unless you've parsed or
// decoded a valid UCAN and want to wrap it into a view.
func NewUCANView(model *datamodel.UCANModel) (UCANView, error) {
	panic("TODO")
	return &ucanView{}, nil
}

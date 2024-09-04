package datamodel

import (
	// for go:embed
	_ "embed"
	"fmt"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed errors.ipldsch
var errorsch []byte

var (
	errorTypeSystem *schema.TypeSystem
)

func init() {
	ts, err := ipld.LoadSchemaBytes(errorsch)
	if err != nil {
		panic(fmt.Errorf("failed to load IPLD schema: %s", err))
	}
	errorTypeSystem = ts
}

func InvalidAudienceType() schema.Type {
	return errorTypeSystem.TypeByName("InvalidAudience")
}

type Delegation struct {
	Audience string
}

type InvalidAudienceModel struct {
	Name       *string
	Audience   string
	Delegation Delegation
	Message    string
	Stack      *string
}

type ExpiredModel struct {
	Name      *string
	Message   string
	ExpiredAt int64
	Stack     *string
}

func ExpiredType() schema.Type {
	return errorTypeSystem.TypeByName("Expired")
}

type NotValidBeforeModel struct {
	Name    *string
	Message string
	ValidAt int64
	Stack   *string
}

func NotValidBeforeType() schema.Type {
	return errorTypeSystem.TypeByName("NotValidBefore")
}

type FailureModel struct {
	Name    *string
	Message string
}

type CapabilityModel struct {
	Can  string
	With string
}

type EscalatedCapabilityModel struct {
	Name      *string
	Message   string
	Claimed   CapabilityModel
	Delegated CapabilityModel
	Cause     FailureModel
	Stack     *string
}

func EscalatedCapabilityType() schema.Type {
	return errorTypeSystem.TypeByName("EscalatedCapability")
}

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
		panic(fmt.Errorf("failed to load IPLD schema: %w", err))
	}
	errorTypeSystem = ts
}

func Schema() []byte {
	return errorsch
}

func HandlerExecutionErrorType() schema.Type {
	return errorTypeSystem.TypeByName("HandlerExecutionError")
}

type FailureModel struct {
	Name    *string
	Message string
	Stack   *string
}

type CapabilityModel struct {
	Can  string
	With string
}

type HandlerExecutionErrorModel struct {
	Error      bool
	Name       *string
	Message    string
	Stack      *string
	Cause      FailureModel
	Capability CapabilityModel
}

func InvocationCapabilityErrorType() schema.Type {
	return errorTypeSystem.TypeByName("InvocationCapabilityError")
}

type InvocationCapabilityErrorModel struct {
	Error        bool
	Name         *string
	Message      string
	Capabilities []CapabilityModel
}

func HandlerNotFoundErrorType() schema.Type {
	return errorTypeSystem.TypeByName("HandlerNotFoundError")
}

type HandlerNotFoundErrorModel struct {
	Error      bool
	Name       *string
	Message    string
	Capability CapabilityModel
}

func InvalidAudienceErrorType() schema.Type {
	return errorTypeSystem.TypeByName("InvalidAudienceError")
}

type InvalidAudienceErrorModel struct {
	Error   bool
	Name    *string
	Message string
}

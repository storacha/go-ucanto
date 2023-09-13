package ucan

import (
	did "github.com/alanshaw/go-ucanto/did"
	"github.com/ipld/go-ipld-prime"
)

// Resorce is a string that represents resource a UCAN holder can act upon.
// It MUST have format `${string}:${string}`
type Resource = string

// Ability is a string that represents some action that a UCAN holder can do.
// It MUST have format `${string}/${string}` | "*"
type Ability = string

// Capability represents an ability that a UCAN holder can perform with some
// resource.
type Capability interface {
	can() Ability
	with() Resource
	nb() any
}

type Principal interface {
	DID() did.DID
}

// Link is an IPLD link to UCAN data.
type Link interface {
	ipld.Link
}

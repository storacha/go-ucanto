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
type Capability[Caveats any] interface {
	can() Ability
	with() Resource
	nb() Caveats
}

// Principal is a DID object representation with a `did` accessor for the DID.
type Principal interface {
	DID() did.DID
}

// Link is an IPLD link to UCAN data.
type Link interface {
	ipld.Link
}

// Version of the UCAN spec used to produce a specific UCAN.
// It MUST have format `${number}.${number}.${number}`
type Version = string

// UTCUnixTimestamp is a timestamp in seconds since the Unix epoch.
type UTCUnixTimestamp = uint64

// https://github.com/ucan-wg/spec/#324-nonce
type Nonce = string

// A map of arbitrary facts and proofs of knowledge. The enclosed data MUST
// be self-evident and externally verifiable. It MAY include information such
// as hash preimages, server challenges, a Merkle proof, dictionary data, etc.
// See https://github.com/ucan-wg/spec/#325-facts
type Fact = map[string]any

type Code = uint64

package ucan

import (
	"encoding/json"

	"github.com/ipld/go-ipld-prime"
	"github.com/web3-storage/go-ucanto/did"
	"github.com/web3-storage/go-ucanto/ucan/crypto"
	"github.com/web3-storage/go-ucanto/ucan/crypto/signature"
)

// Resorce is a string that represents resource a UCAN holder can act upon.
// It MUST have format `${string}:${string}`
type Resource = string

// Ability is a string that represents some action that a UCAN holder can do.
// It MUST have format `${string}/${string}` | "*"
type Ability = string

// UnknownCapability is a capability whose Nb type is unknown
type UnknownCapability interface {
	json.Marshaler
	Can() Ability
	With() Resource
}

// Capability represents an ability that a UCAN holder can perform with some
// resource.
type Capability[Caveats any] interface {
	UnknownCapability
	Nb() Caveats
}

// Principal is a DID object representation with a `did` accessor for the DID.
type Principal interface {
	DID() did.DID
}

// Link is an IPLD link to UCAN data.
type Link = ipld.Link

// Version of the UCAN spec used to produce a specific UCAN.
// It MUST have format `${number}.${number}.${number}`
type Version = string

// UTCUnixTimestamp is a timestamp in milliseconds since the Unix epoch.
type UTCUnixTimestamp = uint64

// https://github.com/ucan-wg/spec/#324-nonce
type Nonce = string

// A map of arbitrary facts and proofs of knowledge. The enclosed data MUST
// be self-evident and externally verifiable. It MAY include information such
// as hash preimages, server challenges, a Merkle proof, dictionary data, etc.
// See https://github.com/ucan-wg/spec/#325-facts
type Fact = map[string]any

// Signer is an entity that can sign UCANs with keys from a `Principal`.
type Signer interface {
	Principal
	crypto.Signer

	// SignatureCode is an integer corresponding to the byteprefix of the
	// signature algorithm. It is used to tag the [signature] so it can self
	// describe what algorithm was used.
	//
	// [signature]: https://github.com/ucan-wg/ucan-ipld/#25-signature
	SignatureCode() uint64

	// SignatureAlgorithm is the name of the signature algorithm. It is a human
	// readable equivalent of the `SignatureCode`, however it is also used as the
	// last segment in [Nonstandard Signatures], which is used as an `alg` field
	// of the JWT header.
	//
	// [Nonstandard Signatures]: https://github.com/ucan-wg/ucan-ipld/#251-nonstandard-signatures
	SignatureAlgorithm() string
}

// Verifier is an entity that can verify UCAN signatures against a `Principal`.
type Verifier interface {
	Principal
	signature.Verifier
}

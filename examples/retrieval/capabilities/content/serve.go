package content

import (
	ipldprime "github.com/ipld/go-ipld-prime"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/schema"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/storacha/go-ucanto/validator"
)

type ServeCaveats struct {
	Digest []byte
	Range  []int
}

var ServeTypeSystem = helpers.Must(ipldprime.LoadSchemaBytes([]byte(`
	type ServeCaveats struct {
		digest Bytes
		range [Int]
	}
	type ServeOk struct {
		digest Bytes
		range [Int]
	}
`)))

func (sc ServeCaveats) ToIPLD() (ipld.Node, error) {
	return ipld.WrapWithRecovery(&sc, ServeTypeSystem.TypeByName("ServeCaveats"))
}

type ServeOk struct {
	Digest []byte
	Range  []int
}

func (so ServeOk) ToIPLD() (ipld.Node, error) {
	return ipld.WrapWithRecovery(&so, ServeTypeSystem.TypeByName("ServeOk"))
}

var ServeCaveatsReader = schema.Struct[ServeCaveats](ServeTypeSystem.TypeByName("ServeCaveats"), nil)

var Serve = validator.NewCapability(
	"content/serve",
	schema.DIDString(),
	ServeCaveatsReader,
	validator.DefaultDerives,
)

package schema

import (
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/did"
)

var didreader = reader[string, did.DID]{
	readFunc: func(input string) result.Result[did.DID, result.Failure] {
		d, err := did.Parse(input)
		if err != nil {
			return result.Error[did.DID](NewSchemaError(err.Error()))
		}
		return result.Ok[did.DID, result.Failure](d)
	},
}

func DID() Reader[string, did.DID] {
	return &didreader
}

// DIDString read a string that is in DID format.
func DIDString() Reader[string, string] {
	return &didstrreader
}

var didstrreader = reader[string, string]{
	readFunc: func(input string) result.Result[string, result.Failure] {
		return result.MapOk(DID().Read(input), func(id did.DID) string {
			return id.String()
		})
	},
}

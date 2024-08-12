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

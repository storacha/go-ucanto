package schema

import (
	"fmt"
	"strings"

	"github.com/storacha-network/go-ucanto/core/result/failure"
	"github.com/storacha-network/go-ucanto/did"
)

var didreader = reader[string, did.DID]{
	readFunc: func(input string) (did.DID, failure.Failure) {
		pfx := "did:"
		if !strings.HasPrefix(input, pfx) {
			return did.Undef, NewSchemaError(fmt.Sprintf(`Expected a "%s" but got "%s" instead`, pfx, input))
		}
		d, err := did.Parse(input)
		if err != nil {
			return did.Undef, NewSchemaError(err.Error())
		}
		return d, nil
	},
}

func DID() Reader[string, did.DID] {
	return didreader
}

// DIDString read a string that is in DID format.
func DIDString() Reader[string, string] {
	return didstrreader
}

var didstrreader = reader[string, string]{
	readFunc: func(input string) (string, failure.Failure) {
		d, err := DID().Read(input)
		if err != nil {
			return "", err
		}
		return d.String(), nil
	},
}

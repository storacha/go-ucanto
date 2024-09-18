package schema

import (
	"fmt"
	"strings"

	"github.com/storacha/go-ucanto/core/result/failure"
	"github.com/storacha/go-ucanto/did"
)

type didConfig struct {
	method string
}

type DIDOption func(*didConfig)

func WithMethod(method string) DIDOption {
	return func(c *didConfig) {
		c.method = method
	}
}

func DID(opts ...DIDOption) Reader[string, did.DID] {
	c := &didConfig{}
	for _, opt := range opts {
		opt(c)
	}
	return reader[string, did.DID]{
		readFunc: func(input string) (did.DID, failure.Failure) {
			pfx := "did:"
			if c.method != "" {
				pfx = fmt.Sprintf("%s%s:", pfx, c.method)
			}
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
}

// DIDString read a string that is in DID format.
func DIDString(opts ...DIDOption) Reader[string, string] {
	rdr := DID(opts...)
	return reader[string, string]{
		readFunc: func(input string) (string, failure.Failure) {
			d, err := rdr.Read(input)
			if err != nil {
				return "", err
			}
			return d.String(), nil
		},
	}
}

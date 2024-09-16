package schema

import (
	"fmt"
	"net/url"

	"github.com/storacha-network/go-ucanto/core/result/failure"
)

type uriConfig struct {
	protocol *string
}

type uriReader struct {
	uc *uriConfig
}

type URIOption func(*uriConfig)

func WithProtocol(protocol string) URIOption {
	return func(uc *uriConfig) {
		uc.protocol = &protocol
	}
}

func (ur uriReader) Read(input any) (url.URL, failure.Failure) {
	asString, stringOk := input.(string)
	asUrl, urlOk := input.(url.URL)
	if !stringOk && !urlOk {
		return url.URL{}, NewSchemaError(fmt.Sprintf("Expected URI but got %T", input))
	}
	if !urlOk {
		u, err := url.ParseRequestURI(asString)
		if err != nil {
			return url.URL{}, NewSchemaError("Invalid URI")
		}
		asUrl = *u
	}
	if ur.uc.protocol != nil && *ur.uc.protocol != asUrl.Scheme+":" {
		return url.URL{}, NewSchemaError(fmt.Sprintf("Expected %s URI instead got %s", *ur.uc.protocol, asUrl.String()))
	}
	return asUrl, nil
}

func URI(opts ...URIOption) Reader[any, url.URL] {
	uc := &uriConfig{}
	for _, opt := range opts {
		opt(uc)
	}
	return uriReader{uc}
}

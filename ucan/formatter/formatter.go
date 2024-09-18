package formatter

import (
	"encoding/base64"
	"fmt"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagjson"
	"github.com/storacha/go-ucanto/ucan/crypto/signature"
	hdm "github.com/storacha/go-ucanto/ucan/datamodel/header"
	pdm "github.com/storacha/go-ucanto/ucan/datamodel/payload"
)

func FormatSignPayload(payload pdm.PayloadModel, version string, algorithm string) (string, error) {
	hdr, err := FormatHeader(version, algorithm)
	if err != nil {
		return "", fmt.Errorf("formatting header: %s", hdr)
	}
	pld, err := FormatPayload(payload)
	if err != nil {
		return "", fmt.Errorf("formatting payload: %s", err)
	}
	return fmt.Sprintf("%s.%s", hdr, pld), nil
}

func FormatHeader(version string, algorithm string) (string, error) {
	header := hdm.HeaderModel{
		Alg: algorithm,
		Ucv: version,
		Typ: "JWT",
	}
	bytes, err := ipld.Marshal(dagjson.Encode, &header, hdm.Type())
	if err != nil {
		return "", fmt.Errorf("dag-json encoding header: %s", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func FormatPayload(payload pdm.PayloadModel) (string, error) {
	bytes, err := ipld.Marshal(dagjson.Encode, &payload, pdm.Type())
	if err != nil {
		return "", fmt.Errorf("dag-json encoding payload: %s", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func FormatSignature(s signature.Signature) (string, error) {
	return base64.RawURLEncoding.EncodeToString(s.Raw()), nil
}

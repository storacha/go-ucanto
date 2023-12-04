package formatter

import (
	"encoding/base64"
	"fmt"

	"github.com/alanshaw/go-ucanto/ucan/crypto/signature"
	hdm "github.com/alanshaw/go-ucanto/ucan/datamodel/header"
	pdm "github.com/alanshaw/go-ucanto/ucan/datamodel/payload"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagjson"
)

func FormatSignPayload(header *hdm.HeaderModel, payload *pdm.PayloadModel) (string, error) {
	hdr, err := FormatHeader(header)
	if err != nil {
		return "", fmt.Errorf("formatting header: %s", hdr)
	}
	pld, err := FormatPayload(payload)
	if err != nil {
		return "", fmt.Errorf("formatting payload: %s", err)
	}
	return fmt.Sprintf("%s.%s", hdr, pld), nil
}

func FormatHeader(header *hdm.HeaderModel) (string, error) {
	bytes, err := ipld.Marshal(dagjson.Encode, header, hdm.Type())
	if err != nil {
		return "", fmt.Errorf("dag-json encoding header: %s", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func FormatPayload(payload *pdm.PayloadModel) (string, error) {
	bytes, err := ipld.Marshal(dagjson.Encode, payload, pdm.Type())
	if err != nil {
		return "", fmt.Errorf("dag-json encoding payload: %s", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func FormatSignature(s signature.Signature) (string, error) {
	return base64.RawURLEncoding.EncodeToString(s.Raw()), nil
}

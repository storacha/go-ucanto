package multiformat

import (
	"bytes"
	"fmt"

	"github.com/multiformats/go-varint"
)

func TagWith(code uint64, bytes []byte) []byte {
	offset := varint.UvarintSize(code)
	tagged := make([]byte, len(bytes)+offset)
	varint.PutUvarint(tagged, code)
	copy(tagged[offset:], bytes)
	return tagged
}

func UntagWith(code uint64, source []byte, offset int) ([]byte, error) {
	b := source
	if offset != 0 {
		b = source[offset:]
	}

	tag, err := varint.ReadUvarint(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	if tag != code {
		return nil, fmt.Errorf("expected multiformat with 0x%x tag instead got 0x%x", code, tag)
	}

	size := varint.UvarintSize(code)
	return b[size:], nil
}

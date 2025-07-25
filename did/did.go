package did

import (
	"encoding/json"
	"fmt"
	"strings"

	mbase "github.com/multiformats/go-multibase"
	varint "github.com/multiformats/go-varint"
)

const Prefix = "did:"
const KeyPrefix = "did:key:"

const DIDCore = 0x0d1d
const Ed25519 = 0xed
const RSA = 0x1205

var MethodOffset = varint.UvarintSize(uint64(DIDCore))

type DID struct {
	key bool
	str string
}

// Undef can be used to represent a nil or undefined DID, using DID{}
// directly is also acceptable.
var Undef = DID{}

func (d DID) Defined() bool {
	return d.str != ""
}

func (d DID) Bytes() []byte {
	if !d.Defined() {
		return nil
	}
	return []byte(d.str)
}

func (d DID) DID() DID {
	return d
}

// String formats the decentralized identity document (DID) as a string.
func (d DID) String() string {
	if d.key {
		key, _ := mbase.Encode(mbase.Base58BTC, []byte(d.str))
		return "did:key:" + key
	}
	return "did:" + d.str[MethodOffset:]
}

// GoString formats the decentralized identity document (DID) as a string.
func (d DID) GoString() string {
	return d.String()
}

func (d DID) MarshalJSON() ([]byte, error) {
	if d == Undef {
		return json.Marshal(nil)
	}
	return json.Marshal(d.String())
}

func (d *DID) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return fmt.Errorf("parsing string: %w", err)
	}
	if str == "" {
		return nil
	}
	parsed, err := Parse(str)
	if err != nil {
		return fmt.Errorf("parsing DID: %w", err)
	}
	*d = parsed
	return nil
}

func Decode(bytes []byte) (DID, error) {
	code, _, err := varint.FromUvarint(bytes)
	if err != nil {
		return Undef, err
	}
	if code == Ed25519 || code == RSA {
		return DID{str: string(bytes), key: true}, nil
	} else if code == DIDCore {
		return DID{str: string(bytes)}, nil
	}
	return Undef, fmt.Errorf("unsupported DID encoding: 0x%x", code)
}

func Parse(str string) (DID, error) {
	if !strings.HasPrefix(str, Prefix) {
		return Undef, fmt.Errorf("must start with 'did:'")
	}

	if strings.HasPrefix(str, KeyPrefix) {
		code, bytes, err := mbase.Decode(str[len(KeyPrefix):])
		if err != nil {
			return Undef, err
		}
		if code != mbase.Base58BTC {
			return Undef, fmt.Errorf("not Base58BTC encoded")
		}
		return Decode(bytes)
	}

	buf := make([]byte, MethodOffset)
	varint.PutUvarint(buf, DIDCore)
	suffix, _ := strings.CutPrefix(str, Prefix)
	buf = append(buf, suffix...)
	return DID{str: string(buf)}, nil
}

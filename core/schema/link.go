package schema

import (
	"bytes"
	"fmt"

	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-base32"
	mh "github.com/multiformats/go-multihash"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/result/failure"
)

type linkReader struct {
	lc *linkCfg
}

func (lr linkReader) Read(input any) (ipld.Link, failure.Failure) {
	link, asLink := input.(ipld.Link)
	if !asLink {
		node, asNode := input.(ipld.Node)
		if !asNode {
			// If input is not an IPLD node, can it be converted to one?
			if builder, ok := input.(ipld.Builder); ok {
				n, err := builder.ToIPLD()
				if err != nil {
					return nil, NewSchemaError(err.Error())
				}
				node = n
			} else {
				return nil, NewSchemaError("unexpected input: not an IPLD node or link")
			}
		}
		var err error
		link, err = node.AsLink()
		if err != nil {
			return nil, NewSchemaError(err.Error())
		}
	}

	cidLink, ok := link.(cidlink.Link)
	if !ok {
		return nil, NewSchemaError("Unsupported Link Type")
	}
	cid := cidLink.Cid
	if lr.lc.codec != nil && cid.Prefix().Codec != *lr.lc.codec {
		return nil, NewSchemaError(fmt.Sprintf("Expected link to be CID with %X codec", *lr.lc.codec))
	}

	if lr.lc.version != nil && cid.Prefix().Version != *lr.lc.version {
		return nil, NewSchemaError(fmt.Sprintf(
			"Expected link to be CID version %d instead of %d", *lr.lc.version, cid.Prefix().Version))
	}

	if lr.lc.multihash != nil {
		multihash := lr.lc.multihash
		if multihash.code != nil && cid.Prefix().MhType != *multihash.code {
			return nil, NewSchemaError(fmt.Sprintf("Expected link to be CID with %X hashing algorithm", *&multihash.code))
		}
		if multihash.digest != nil {
			decoded, err := mh.Decode(cid.Hash())
			if err != nil {
				return nil, NewSchemaError(err.Error())
			}

			if bytes.Compare(decoded.Digest, *multihash.digest) != 0 {
				return nil, NewSchemaError(fmt.Sprintf("Expected link with %s hash digest instead of %s", base32.StdEncoding.EncodeToString(*multihash.digest), base32.StdEncoding.EncodeToString(decoded.Digest)))
			}
		}
	}
	return link, nil
}

type multihashConfig struct {
	code   *uint64
	digest *[]byte
}

type MultihashOption func(*multihashConfig)

func WithAlg(code uint64) MultihashOption {
	return func(mc *multihashConfig) {
		mc.code = &code
	}
}

func WithDigest(digest []byte) MultihashOption {
	return func(mc *multihashConfig) {
		mc.digest = &digest
	}
}

type linkCfg struct {
	version   *uint64
	codec     *uint64
	multihash *multihashConfig
}

type LinkOption func(*linkCfg)

func WithVersion(version uint64) LinkOption {
	return func(lc *linkCfg) {
		lc.version = &version
	}
}

func WithCodec(codec uint64) LinkOption {
	return func(lc *linkCfg) {
		lc.codec = &codec
	}
}

func WithMultihashConfig(opts ...MultihashOption) LinkOption {
	return func(lc *linkCfg) {
		mc := &multihashConfig{}
		for _, opt := range opts {
			opt(mc)
		}
		lc.multihash = mc
	}
}

func Link(opts ...LinkOption) Reader[any, ipld.Link] {
	lc := &linkCfg{}
	for _, opt := range opts {
		opt(lc)
	}
	return linkReader{lc}
}

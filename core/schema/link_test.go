package schema_test

import (
	"fmt"
	"iter"
	"maps"
	"regexp"
	"testing"

	"github.com/ipfs/go-cid"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
	"github.com/multiformats/go-base32"
	mh "github.com/multiformats/go-multihash"
	"github.com/storacha/go-ucanto/core/iterable"
	"github.com/storacha/go-ucanto/core/schema"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/stretchr/testify/require"
)

func NewSet[T comparable](list iter.Seq[T]) iter.Seq[T] {
	set := make(map[T]struct{})
	for elem := range list {
		set[elem] = struct{}{}
	}
	return maps.Keys(set)
}
func TestReadLink(t *testing.T) {
	fixtures := map[string]cid.Cid{
		"pb":          cid.MustParse("QmTgnQBKj7eTV7ohraBCmh1DLwerUd2X9Rxzgf3gyMJbC8"),
		"cbor":        cid.MustParse("bafyreieuo63r3y2nuycaq4b3q2xvco3nprlxiwzcfp4cuupgaywat3z6mq"),
		"rawIdentity": cid.MustParse("bafkqaaa"),
		"ipns":        cid.MustParse("k2k4r8kuj2bs2l996lhjx8rc727xlvthtak8o6eia3qm5adxvs5k84gf"),
		"sha512":      cid.MustParse("kgbuwaen1jrbjip6iwe9mqg54spvuucyz7f5jho2tkc2o0c7xzqwpxtogbyrwck57s9is6zqlwt9rsxbuvszym10nbaxt9jn7sf4eksqd"),
	}

	links := maps.Values(fixtures)
	versions := NewSet(iterable.Map(func(c cid.Cid) uint64 { return c.Version() }, maps.Values(fixtures)))
	codecs := NewSet(iterable.Map(func(c cid.Cid) uint64 { return c.Prefix().Codec }, maps.Values(fixtures)))
	algs := NewSet(iterable.Map(func(c cid.Cid) uint64 { return c.Prefix().MhType }, maps.Values(fixtures)))
	digests := NewSet(iterable.Map(func(c cid.Cid) string {
		h := c.Hash()
		dh := helpers.Must(mh.Decode(h))
		return string(dh.Digest)
	}, maps.Values(fixtures)))

	for link := range links {
		t.Run(fmt.Sprintf("%s ➡ schema.Link()", link), func(t *testing.T) {
			output, err := schema.Link().Read(basicnode.NewLink(cidlink.Link{Cid: link}))
			require.NoError(t, err)
			require.Equal(t, output, cidlink.Link{Cid: link}, link.String())
		})

		for version := range versions {
			t.Run(fmt.Sprintf("%s ➡ schema.Link(WithVersion(%d))", link, version), func(t *testing.T) {
				reader := schema.Link(schema.WithVersion(version))
				output, err := reader.Read(basicnode.NewLink(cidlink.Link{Cid: link}))
				if link.Version() == version {
					require.NoError(t, err)
					require.Equal(t, output, cidlink.Link{Cid: link})
				} else {
					require.Error(t, err)
					require.Contains(t, err.Error(), "Expected link to be CID version")
				}
			})
		}

		for codec := range codecs {
			t.Run(fmt.Sprintf("%s ➡ schema.Link(WithCodec(%d))", link, codec), func(t *testing.T) {
				reader := schema.Link(schema.WithCodec(codec))
				output, err := reader.Read(basicnode.NewLink(cidlink.Link{Cid: link}))
				if link.Prefix().Codec == codec {
					require.NoError(t, err)
					require.Equal(t, output, cidlink.Link{Cid: link})
				} else {
					require.Error(t, err)
					require.Regexp(t, helpers.Must(regexp.Compile("Expected link to be CID with .* code")), err.Error())
				}
			})
		}

		for alg := range algs {
			t.Run(fmt.Sprintf("%s ➡ schema.Link(WithMultihashConfig(WithAlg(%d)))", link, alg), func(t *testing.T) {
				reader := schema.Link(schema.WithMultihashConfig(schema.WithAlg(alg)))
				output, err := reader.Read(basicnode.NewLink(cidlink.Link{Cid: link}))
				if link.Prefix().MhType == alg {
					require.NoError(t, err)
					require.Equal(t, output, cidlink.Link{Cid: link})
				} else {
					require.Error(t, err)
					require.Regexp(t, helpers.Must(regexp.Compile("Expected link to be CID with .* hashing algorithm")), err.Error())
				}
			})
		}

		for digest := range digests {
			t.Run(fmt.Sprintf("%s ➡ schema.Link(WithMultihashConfig(WithDigest(%s)))", link, base32.StdEncoding.EncodeToString([]byte(digest))), func(t *testing.T) {
				reader := schema.Link(schema.WithMultihashConfig(schema.WithDigest([]byte(digest))))
				output, err := reader.Read(basicnode.NewLink(cidlink.Link{Cid: link}))
				if string(helpers.Must(mh.Decode(link.Hash())).Digest) == digest {
					require.NoError(t, err)
					require.Equal(t, output, cidlink.Link{Cid: link})
				} else {
					require.Error(t, err)
					require.Regexp(t, helpers.Must(regexp.Compile("Expected link with .* hash digest")), err.Error())
				}
			})
		}
	}
}

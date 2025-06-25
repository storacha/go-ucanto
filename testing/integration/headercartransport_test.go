package testing

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	prime "github.com/ipld/go-ipld-prime"
	"github.com/multiformats/go-multihash"
	"github.com/storacha/go-ucanto/client"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/core/receipt/fx"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/core/schema"
	"github.com/storacha/go-ucanto/server"
	"github.com/storacha/go-ucanto/testing/fixtures"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/storacha/go-ucanto/transport/headercar"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/storacha/go-ucanto/validator"
	"github.com/stretchr/testify/require"
)

type serveCaveats struct {
	Digest []byte
	Range  []int
}

var serveTS = helpers.Must(prime.LoadSchemaBytes([]byte(`
	type ServeCaveats struct {
		digest Bytes
		range [Int]
	}
	type ServeOk struct {
		digest Bytes
		range [Int]
	}
`)))

func (sc serveCaveats) ToIPLD() (ipld.Node, error) {
	return ipld.WrapWithRecovery(&sc, serveTS.TypeByName("ServeCaveats"))
}

type serveOk struct {
	Digest []byte
	Range  []int
}

func (so serveOk) ToIPLD() (ipld.Node, error) {
	return ipld.WrapWithRecovery(&so, serveTS.TypeByName("ServeOk"))
}

var serveCaveatsReader = schema.Struct[serveCaveats](serveTS.TypeByName("ServeCaveats"), nil)

var serve = validator.NewCapability(
	"space/content/serve",
	schema.DIDString(),
	serveCaveatsReader,
	validator.DefaultDerives,
)

type mockStreamer struct {
	t    *testing.T
	data map[string][]byte
}

func (ms *mockStreamer) Stream(msg message.AgentMessage) (io.Reader, http.Header, error) {
	t := ms.t
	require.Len(t, msg.Receipts(), 1)
	rcpt, ok, err := msg.Receipt(msg.Receipts()[0])
	require.NoError(t, err)
	require.True(t, ok)

	o, x := result.Unwrap(rcpt.Out())
	require.Nil(t, x)

	serveOk, err := ipld.Rebind[serveOk](o, serveTS.TypeByName("ServeOk"))
	require.NoError(t, err)

	data := ms.data[multihash.Multihash(serveOk.Digest).B58String()]
	offset, length := serveOk.Range[0], serveOk.Range[1]
	return bytes.NewReader(data[offset : offset+length]), http.Header{}, nil
}

func TestHeaderCARTransport(t *testing.T) {
	blobDigest := helpers.RandomDigest()
	data := helpers.RandomBytes(1024)
	dataStreamer := mockStreamer{
		t:    t,
		data: map[string][]byte{blobDigest.B58String(): data},
	}

	server, err := server.NewServer(
		fixtures.Service,
		// Handler definitions
		server.WithServiceMethod(
			serve.Can(),
			server.Provide(
				serve,
				func(cap ucan.Capability[serveCaveats], inv invocation.Invocation, ctx server.InvocationContext) (serveOk, fx.Effects, error) {
					return serveOk{Digest: cap.Nb().Digest, Range: cap.Nb().Range}, nil, nil
				},
			),
		),
		server.WithInboundCodec(headercar.NewInboundCodec(headercar.WithDataStreamer(&dataStreamer))),
	)
	require.NoError(t, err)

	conn, err := client.NewConnection(
		fixtures.Service,
		server,
		client.WithOutboundCodec(headercar.NewOutboundCodec()),
	)
	require.NoError(t, err)

	inv, err := invocation.Invoke(
		fixtures.Alice,
		fixtures.Service,
		serve.New(
			fixtures.Alice.DID().String(),
			serveCaveats{
				Digest: []byte(blobDigest),
				Range:  []int{5, 900},
			},
		),
	)
	require.NoError(t, err)

	input, err := message.Build([]invocation.Invocation{inv}, nil)
	require.NoError(t, err)

	req, err := conn.Codec().Encode(input)
	require.NoError(t, err)

	res, err := conn.Channel().Request(req)
	require.NoError(t, err)

	output, err := conn.Codec().Decode(res)
	require.NoError(t, err)

	require.Len(t, output.Receipts(), 1)
	rcpt, ok, err := output.Receipt(output.Receipts()[0])
	require.NoError(t, err)
	require.True(t, ok)

	o, x := result.Unwrap(rcpt.Out())
	require.Nil(t, x)
	serveOk, err := ipld.Rebind[serveOk](o, serveTS.TypeByName("ServeOk"))
	require.NoError(t, err)
	require.Equal(t, serveOk.Digest, []byte(blobDigest))

	// stream the bytes
	outBytes, err := io.ReadAll(res.Body())
	require.NoError(t, err)
	require.Equal(t, data[5:905], outBytes)
}

package testing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	prime "github.com/ipld/go-ipld-prime"
	"github.com/multiformats/go-multihash"
	"github.com/storacha/go-ucanto/client"
	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/core/receipt"
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

type mockBodyProvider struct {
	t    *testing.T
	data map[string][]byte
}

func (ms *mockBodyProvider) Stream(msg message.AgentMessage) (io.Reader, int, http.Header, error) {
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
	return bytes.NewReader(data[offset : offset+length]), 206, http.Header{}, nil
}

type mockDelegationCache struct {
	data map[string]delegation.Delegation
}

func (m *mockDelegationCache) Get(ctx context.Context, root ipld.Link) (delegation.Delegation, bool, error) {
	d, ok := m.data[root.String()]
	return d, ok, nil
}

func (m *mockDelegationCache) Put(ctx context.Context, d delegation.Delegation) error {
	m.data[d.Link().String()] = d
	return nil
}

func TestHeaderCARTransport(t *testing.T) {
	blobDigest := helpers.RandomDigest()
	data := helpers.RandomBytes(1024)
	provider := mockBodyProvider{
		t:    t,
		data: map[string][]byte{blobDigest.B58String(): data},
	}
	dlgCache := mockDelegationCache{data: map[string]delegation.Delegation{}}

	server, err := server.NewServer(
		fixtures.Service,
		// Handler definitions
		server.WithServiceMethod(
			serve.Can(),
			server.Provide(
				serve,
				func(ctx context.Context, cap ucan.Capability[serveCaveats], inv invocation.Invocation, ictx server.InvocationContext) (serveOk, fx.Effects, error) {
					printDelegation(t, inv)
					return serveOk{Digest: cap.Nb().Digest, Range: cap.Nb().Range}, nil, nil
				},
			),
		),
		server.WithInboundCodec(headercar.NewInboundCodec(&dlgCache, headercar.WithResponseBodyProvider(&provider))),
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

	res, err := conn.Channel().Request(t.Context(), req)
	require.NoError(t, err)

	output, err := conn.Codec().Decode(res)
	require.NoError(t, err)

	require.Len(t, output.Receipts(), 1)
	rcpt, ok, err := output.Receipt(output.Receipts()[0])
	require.NoError(t, err)
	require.True(t, ok)

	fmt.Println("---")
	printReceipt(t, rcpt)

	o, x := result.Unwrap(rcpt.Out())
	require.Nil(t, x)
	serveOk, err := ipld.Rebind[serveOk](o, serveTS.TypeByName("ServeOk"))
	require.NoError(t, err)
	require.Equal(t, serveOk.Digest, []byte(blobDigest))

	// stream the bytes
	outBytes, err := io.ReadAll(res.Body())
	require.NoError(t, err)
	require.Equal(t, data[5:905], outBytes)

	fmt.Println("---")
	fmt.Println("Response Body")
	fmt.Printf("\tRange: %d-%d\n", serveOk.Range[0], serveOk.Range[0]+serveOk.Range[1])
	fmt.Printf("\tBytes: 0x%x\n", outBytes)
}

func printDelegation(t *testing.T, d delegation.Delegation) {
	t.Helper()
	fmt.Printf("Delegation (%s)\n", d.Link())
	fmt.Printf("\tIssuer: %s\n", d.Issuer().DID())
	fmt.Printf("\tAudience: %s\n", d.Audience().DID())

	fmt.Println("\tCapabilities:")
	for _, c := range d.Capabilities() {
		fmt.Printf("\t\tCan: %s\n", c.Can())
		fmt.Printf("\t\tWith: %s\n", c.With())
		fmt.Printf("\t\tNb: %v\n", c.Nb())
	}

	if d.Expiration() != nil {
		fmt.Printf("\tExpiration: %s\n", time.Unix(int64(*d.Expiration()), 0).String())
	}

	bs, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(d.Blocks()))
	require.NoError(t, err)

	if len(d.Proofs()) > 0 {
		fmt.Println("\tProofs:")
		for _, p := range d.Proofs() {
			fmt.Printf("\t\t%s\n", p)
		}
		for _, p := range d.Proofs() {
			fmt.Println("---")
			pd, err := delegation.NewDelegationView(p, bs)
			if err == nil {
				printDelegation(t, pd)
			}
		}
	}
}

func printReceipt(t *testing.T, r receipt.AnyReceipt) {
	t.Helper()
	fmt.Printf("Receipt (%s)\n", r.Root().Link())
	fmt.Printf("\tIssuer: %s\n", r.Issuer().DID())
	fmt.Printf("\tRan: %s\n", r.Ran().Link())
	fmt.Println("\tOut:")
	o, x := result.Unwrap(r.Out())
	if x != nil {
		fmt.Printf("\t\tError: %+v\n", x)
	} else {
		fmt.Printf("\t\tOK: %+v\n", o)
	}
}

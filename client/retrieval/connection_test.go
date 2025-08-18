package retrieval

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	prime "github.com/ipld/go-ipld-prime"
	"github.com/multiformats/go-multihash"
	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/receipt"
	"github.com/storacha/go-ucanto/core/receipt/fx"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/core/result/failure"
	"github.com/storacha/go-ucanto/core/schema"
	ed25519 "github.com/storacha/go-ucanto/principal/ed25519/signer"
	"github.com/storacha/go-ucanto/server"
	"github.com/storacha/go-ucanto/server/retrieval"
	"github.com/storacha/go-ucanto/testing/fixtures"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/storacha/go-ucanto/testing/helpers/printer"
	thttp "github.com/storacha/go-ucanto/transport/http"
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
	"content/serve",
	schema.DIDString(),
	serveCaveatsReader,
	validator.DefaultDerives,
)

func mkDelegationChain(t *testing.T, rootIssuer ucan.Signer, endAudience ucan.Principal, can ucan.Ability, len int) delegation.Delegation {
	require.GreaterOrEqual(t, len, 1)

	var dlg delegation.Delegation
	var proof delegation.Delegation

	iss := rootIssuer
	aud, err := ed25519.Generate()
	require.NoError(t, err)

	for range len - 1 {
		var opts []delegation.Option
		if proof != nil {
			opts = append(opts, delegation.WithProof(delegation.FromDelegation(proof)))
		}
		dlg, err = delegation.Delegate(
			iss,
			aud,
			[]ucan.Capability[ucan.NoCaveats]{
				ucan.NewCapability(can, rootIssuer.DID().String(), ucan.NoCaveats{}),
			},
			opts...,
		)
		require.NoError(t, err)
		iss = aud
		aud, err = ed25519.Generate()
		require.NoError(t, err)
		proof = dlg
	}

	var opts []delegation.Option
	if proof != nil {
		opts = append(opts, delegation.WithProof(delegation.FromDelegation(proof)))
	}
	dlg, err = delegation.Delegate(
		iss,
		endAudience,
		[]ucan.Capability[ucan.NoCaveats]{
			ucan.NewCapability(can, rootIssuer.DID().String(), ucan.NoCaveats{}),
		},
		opts...,
	)
	require.NoError(t, err)

	return dlg
}

func calcHeadersSize(h http.Header) int {
	var buf bytes.Buffer
	h.Write(&buf)
	return buf.Len()
}

var kb = 1024

// newRetrievalHTTPServer creates a HTTP server that will send a 431 response
// when HTTP headers exceed 2KiB, but otherwise calls the UCAN server as usual
func newRetrievalHTTPServer(t *testing.T, server server.ServerView[retrieval.Service]) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("-> %s %s", r.Method, r.URL)
		printer.PrintHeaders(t, r.Header)
		size := calcHeadersSize(r.Header)
		t.Logf("Total size of headers: %s", printer.SprintBytes(t, size))

		if size > 2*kb {
			t.Logf("<- %d %s", http.StatusRequestHeaderFieldsTooLarge, http.StatusText(http.StatusRequestHeaderFieldsTooLarge))
			w.WriteHeader(http.StatusRequestHeaderFieldsTooLarge)
			return
		}

		resp, err := server.Request(r.Context(), thttp.NewInboundRequest(r.URL, r.Body, r.Header))
		require.NoError(t, err)

		t.Logf("<- %d %s", resp.Status(), http.StatusText(resp.Status()))
		printer.PrintHeaders(t, resp.Headers())
		t.Logf("Total size of headers: %s", printer.SprintBytes(t, calcHeadersSize(resp.Headers())))

		for name, values := range resp.Headers() {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}
		w.WriteHeader(resp.Status())
		body := resp.Body()
		if body != nil {
			// log out the "not extended" dag-json response for debugging purposes
			if resp.Status() == http.StatusNotExtended {
				bodyBytes, err := io.ReadAll(body)
				require.NoError(t, err)
				t.Logf("Body: %s", string(bodyBytes))
				body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
			_, err := io.Copy(w, body)
			require.NoError(t, err)
		}
	}))
}

type testDelegationCache struct {
	t    *testing.T
	data map[string]delegation.Delegation
}

func (c *testDelegationCache) Get(ctx context.Context, root ipld.Link) (delegation.Delegation, bool, error) {
	d, ok := c.data[root.String()]
	if ok {
		c.t.Logf("CACHE HIT: %s", root.String())
	} else {
		c.t.Logf("CACHE MISS: %s", root.String())
	}
	return d, ok, nil
}

func (c *testDelegationCache) Put(ctx context.Context, d delegation.Delegation) error {
	c.data[d.Link().String()] = d
	c.t.Logf("CACHE PUT: %s", d.Link().String())
	return nil
}

func newTestDelegationCache(t *testing.T) *testDelegationCache {
	return &testDelegationCache{t: t, data: map[string]delegation.Delegation{}}
}

func TestExecute(t *testing.T) {
	chainLengths := []int{1, 5, 10}
	for _, length := range chainLengths {
		t.Run(fmt.Sprintf("retrieval via partitioned request (proof chain of %d delegations)", length), func(t *testing.T) {
			dlg := mkDelegationChain(t, fixtures.Service, fixtures.Alice, serve.Can(), length)
			data := helpers.RandomBytes(512)

			// create a retrieval server that will send bytes back for an authorized
			// UCAN invocation sent in HTTP headers of the GET request
			server, err := retrieval.NewServer(
				fixtures.Service,
				retrieval.WithServiceMethod(
					serve.Can(),
					retrieval.Provide(
						serve,
						func(ctx context.Context, cap ucan.Capability[serveCaveats], inv invocation.Invocation, ictx server.InvocationContext, req retrieval.Request) (result.Result[serveOk, failure.IPLDBuilderFailure], fx.Effects, retrieval.Response, error) {
							t.Logf("Handling %s: %s", serve.Can(), req.URL.String())
							t.Log("Invocation:")
							printer.PrintDelegation(t, inv, 0)
							nb := cap.Nb()
							result := result.Ok[serveOk, failure.IPLDBuilderFailure](serveOk(nb))
							start, end := nb.Range[0], nb.Range[1]
							length := end - start + 1
							headers := http.Header{}
							headers.Set("Content-Length", fmt.Sprintf("%d", length))
							headers.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(data)))
							response := retrieval.Response{
								Status:  http.StatusPartialContent,
								Headers: headers,
								Body:    io.NopCloser(bytes.NewReader(data[start : end+1])),
							}
							return result, nil, response, nil
						},
					),
				),
				retrieval.WithDelegationCache(newTestDelegationCache(t)),
			)
			require.NoError(t, err)

			httpServer := newRetrievalHTTPServer(t, server)
			defer httpServer.Close()

			// make a UCAN authorized retrieval request for some bytes from the data

			// identify the data
			digest, err := multihash.Sum(data, multihash.SHA2_256, -1)
			require.NoError(t, err)

			// specify the byte range we want to receive (inclusive)
			contentRange := []int{100, 200}

			url, err := url.Parse(httpServer.URL)
			require.NoError(t, err)

			// the URL doesn't really have a consequence on this test, but it can be
			// used to idenitfy the data if not done so in the invocation caveats
			conn, err := NewConnection(fixtures.Service, url.JoinPath("blob", "z"+digest.B58String()))
			require.NoError(t, err)

			inv, err := serve.Invoke(
				fixtures.Alice,
				fixtures.Service,
				fixtures.Service.DID().String(),
				serveCaveats{Digest: digest, Range: contentRange},
				delegation.WithProof(delegation.FromDelegation(dlg)),
			)
			require.NoError(t, err)

			// send the invocation, and receive the execution response _as well as_ the
			// HTTP response!
			xRes, hRes, err := Execute(t.Context(), inv, conn)
			require.NoError(t, err)
			require.NotNil(t, xRes)
			require.NotNil(t, hRes)

			rcptLink, ok := xRes.Get(inv.Link())
			require.True(t, ok)

			bs, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(xRes.Blocks()))
			require.NoError(t, err)

			rcpt, err := receipt.NewAnyReceipt(rcptLink, bs)
			require.NoError(t, err)

			// verify the receipt is not an error, and that the info matches the
			// invocation caveats
			o, x := result.Unwrap(rcpt.Out())
			require.Nil(t, x)

			sok, err := ipld.Rebind[serveOk](o, serveTS.TypeByName("ServeOk"))
			require.NoError(t, err)
			require.Equal(t, digest, multihash.Multihash(sok.Digest))
			require.Equal(t, []int{100, 200}, sok.Range)

			// verify the data in the HTTP body is what we asked for
			body, err := io.ReadAll(hRes.Body())
			require.NoError(t, err)
			require.Equal(t, data[100:200+1], body)
		})
	}
}

package server

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/ipfs/go-cid"
	ipldprime "github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	ipldschema "github.com/ipld/go-ipld-prime/schema"
	"github.com/storacha/go-ucanto/client"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/receipt"
	"github.com/storacha/go-ucanto/core/receipt/fx"
	"github.com/storacha/go-ucanto/core/result"
	fdm "github.com/storacha/go-ucanto/core/result/failure/datamodel"
	"github.com/storacha/go-ucanto/core/schema"
	"github.com/storacha/go-ucanto/testing/fixtures"
	"github.com/storacha/go-ucanto/testing/helpers"
	"github.com/storacha/go-ucanto/transport/car/request"
	"github.com/storacha/go-ucanto/transport/car/response"
	thttp "github.com/storacha/go-ucanto/transport/http"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/storacha/go-ucanto/validator"
	"github.com/stretchr/testify/require"
)

type uploadAddCaveats struct {
	Root ipld.Link
}

func (c uploadAddCaveats) ToIPLD() (ipld.Node, error) {
	np := basicnode.Prototype.Any
	nb := np.NewBuilder()
	ma, _ := nb.BeginMap(1)
	if c != (uploadAddCaveats{}) {
		ma.AssembleKey().AssignString("root")
		ma.AssembleValue().AssignLink(c.Root)
	}
	ma.Finish()
	return nb.Build(), nil
}

func uploadAddCaveatsType() ipldschema.Type {
	ts := helpers.Must(ipldprime.LoadSchemaBytes([]byte(`
	  type UploadAddCaveats struct {
		  root Link
		}
	`)))
	return ts.TypeByName("UploadAddCaveats")
}

type uploadAddSuccess struct {
	Root   ipldprime.Link
	Status string
}

func (ok uploadAddSuccess) ToIPLD() (ipld.Node, error) {
	np := basicnode.Prototype.Any
	nb := np.NewBuilder()
	ma, _ := nb.BeginMap(2)
	ma.AssembleKey().AssignString("root")
	ma.AssembleValue().AssignLink(ok.Root)
	ma.AssembleKey().AssignString("status")
	ma.AssembleValue().AssignString(ok.Status)
	ma.Finish()
	return nb.Build(), nil
}

var rcptsch = []byte(`
	type Result union {
		| UploadAddSuccess "ok"
		| Any "error"
	} representation keyed

	type UploadAddSuccess struct {
		root Link
		status String
	}
`)

// asFailure binds the IPLD node to a FailureModel if possible. This works
// around IPLD requiring data to match the schema exactly
func asFailure(t testing.TB, n ipld.Node) fdm.FailureModel {
	t.Helper()
	require.Equal(t, n.Kind(), datamodel.Kind_Map)
	f := fdm.FailureModel{}

	nn, err := n.LookupByString("name")
	if err == nil {
		name, err := nn.AsString()
		require.NoError(t, err)
		f.Name = &name
	}

	mn, err := n.LookupByString("message")
	require.NoError(t, err)
	msg, err := mn.AsString()
	require.NoError(t, err)
	f.Message = msg

	sn, err := n.LookupByString("stack")
	if err == nil {
		stack, err := sn.AsString()
		require.NoError(t, err)
		f.Stack = &stack
	}

	return f
}

func TestExecute(t *testing.T) {
	t.Run("self-signed", func(t *testing.T) {
		uploadadd := validator.NewCapability(
			"upload/add",
			schema.DIDString(),
			schema.Struct[uploadAddCaveats](uploadAddCaveatsType(), nil),
			nil,
		)

		server := helpers.Must(NewServer(
			fixtures.Service,
			WithServiceMethod(
				uploadadd.Can(),
				Provide(uploadadd, func(cap ucan.Capability[uploadAddCaveats], inv invocation.Invocation, ctx InvocationContext) (uploadAddSuccess, fx.Effects, error) {
					return uploadAddSuccess{Root: cap.Nb().Root, Status: "done"}, nil, nil
				}),
			),
		))

		conn := helpers.Must(client.NewConnection(fixtures.Service, server))
		rt := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
		cap := uploadadd.New(fixtures.Service.DID().String(), uploadAddCaveats{Root: rt})
		inv, err := invocation.Invoke(fixtures.Service, fixtures.Service, cap)
		require.NoError(t, err)

		resp, err := client.Execute([]invocation.Invocation{inv}, conn)
		require.NoError(t, err)

		// get the receipt link for the invocation from the response
		rcptlnk, ok := resp.Get(inv.Link())
		require.True(t, ok, "missing receipt for invocation: %s", inv.Link())

		reader := helpers.Must(receipt.NewReceiptReader[uploadAddSuccess, ipld.Node](rcptsch))
		rcpt := helpers.Must(reader.Read(rcptlnk, resp.Blocks()))

		result.MatchResultR0(rcpt.Out(), func(ok uploadAddSuccess) {
			fmt.Printf("%+v\n", ok)
			require.Equal(t, ok.Root, rt)
			require.Equal(t, ok.Status, "done")
		}, func(x ipld.Node) {
			f := asFailure(t, x)
			fmt.Println(f.Message)
			fmt.Println(*f.Stack)
			require.Nil(t, f)
		})
	})

	t.Run("delegated", func(t *testing.T) {
		uploadadd := validator.NewCapability(
			"upload/add",
			schema.DIDString(),
			schema.Struct[uploadAddCaveats](uploadAddCaveatsType(), nil),
			nil,
		)

		server := helpers.Must(NewServer(
			fixtures.Service,
			WithServiceMethod(
				uploadadd.Can(),
				Provide(uploadadd, func(cap ucan.Capability[uploadAddCaveats], inv invocation.Invocation, ctx InvocationContext) (uploadAddSuccess, fx.Effects, error) {
					return uploadAddSuccess{Root: cap.Nb().Root, Status: "done"}, nil, nil
				}),
			),
		))

		conn := helpers.Must(client.NewConnection(fixtures.Service, server))
		rt := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
		cap := uploadadd.New(fixtures.Service.DID().String(), uploadAddCaveats{Root: rt})
		dgl, err := delegation.Delegate(
			fixtures.Service,
			fixtures.Alice,
			[]ucan.Capability[uploadAddCaveats]{
				ucan.NewCapability(uploadadd.Can(), fixtures.Service.DID().String(), uploadAddCaveats{}),
			},
		)
		require.NoError(t, err)

		prfs := []delegation.Proof{delegation.FromDelegation(dgl)}
		inv, err := invocation.Invoke(fixtures.Alice, fixtures.Service, cap, delegation.WithProof(prfs...))
		require.NoError(t, err)

		resp, err := client.Execute([]invocation.Invocation{inv}, conn)
		require.NoError(t, err)

		// get the receipt link for the invocation from the response
		rcptlnk, ok := resp.Get(inv.Link())
		require.True(t, ok, "missing receipt for invocation: %s", inv.Link())

		reader := helpers.Must(receipt.NewReceiptReader[uploadAddSuccess, ipld.Node](rcptsch))
		rcpt := helpers.Must(reader.Read(rcptlnk, resp.Blocks()))

		result.MatchResultR0(rcpt.Out(), func(ok uploadAddSuccess) {
			fmt.Printf("%+v\n", ok)
			require.Equal(t, ok.Root, rt)
			require.Equal(t, ok.Status, "done")
		}, func(x ipld.Node) {
			f := asFailure(t, x)
			fmt.Println(f.Message)
			fmt.Println(*f.Stack)
			require.Nil(t, f)
		})
	})

	t.Run("not found", func(t *testing.T) {
		server := helpers.Must(NewServer(fixtures.Service))
		conn := helpers.Must(client.NewConnection(fixtures.Service, server))

		rt := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
		capability := ucan.NewCapability(
			"upload/add",
			fixtures.Alice.DID().String(),
			uploadAddCaveats{Root: rt},
		)

		invs := []invocation.Invocation{helpers.Must(invocation.Invoke(fixtures.Alice, fixtures.Service, capability))}
		resp := helpers.Must(client.Execute(invs, conn))
		rcptlnk, ok := resp.Get(invs[0].Link())
		require.True(t, ok, "missing receipt for invocation: %s", invs[0].Link())

		reader := helpers.Must(receipt.NewReceiptReader[uploadAddSuccess, ipld.Node](rcptsch))
		rcpt := helpers.Must(reader.Read(rcptlnk, resp.Blocks()))

		result.MatchResultR0(rcpt.Out(), func(uploadAddSuccess) {
			t.Fatalf("expected error: %s", invs[0].Link())
		}, func(x ipld.Node) {
			f := asFailure(t, x)
			fmt.Printf("%s %+v\n", *f.Name, f)
			require.Equal(t, *f.Name, "HandlerNotFoundError")
		})
	})

	t.Run("execution error", func(t *testing.T) {
		uploadadd := validator.NewCapability(
			"upload/add",
			schema.DIDString(),
			schema.Struct[uploadAddCaveats](uploadAddCaveatsType(), nil),
			nil,
		)

		server := helpers.Must(NewServer(
			fixtures.Service,
			WithServiceMethod(
				uploadadd.Can(),
				Provide(uploadadd, func(cap ucan.Capability[uploadAddCaveats], inv invocation.Invocation, ctx InvocationContext) (uploadAddSuccess, fx.Effects, error) {
					return uploadAddSuccess{}, nil, fmt.Errorf("test error")
				}),
			),
		))

		conn := helpers.Must(client.NewConnection(fixtures.Service, server))
		rt := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
		cap := uploadadd.New(fixtures.Alice.DID().String(), uploadAddCaveats{Root: rt})
		invs := []invocation.Invocation{helpers.Must(invocation.Invoke(fixtures.Alice, fixtures.Service, cap))}
		resp := helpers.Must(client.Execute(invs, conn))
		rcptlnk, ok := resp.Get(invs[0].Link())
		require.True(t, ok, "missing receipt for invocation: %s", invs[0].Link())

		reader := helpers.Must(receipt.NewReceiptReader[uploadAddSuccess, ipld.Node](rcptsch))
		rcpt := helpers.Must(reader.Read(rcptlnk, resp.Blocks()))

		result.MatchResultR0(rcpt.Out(), func(uploadAddSuccess) {
			t.Fatalf("expected error: %s", invs[0].Link())
		}, func(x ipld.Node) {
			f := asFailure(t, x)
			fmt.Printf("%s %+v\n", *f.Name, f)
			require.Equal(t, *f.Name, "HandlerExecutionError")
		})
	})
}

func TestHandle(t *testing.T) {
	t.Run("content type error", func(t *testing.T) {
		server := helpers.Must(NewServer(fixtures.Service))

		hd := http.Header{}
		hd.Set("Content-Type", "unsupported/media")
		hd.Set("Accept", response.ContentType)

		req := thttp.NewHTTPRequest(bytes.NewReader([]byte{}), hd)
		res := helpers.Must(Handle(server, req))
		require.Equal(t, res.Status(), http.StatusUnsupportedMediaType)
	})

	t.Run("accept error", func(t *testing.T) {
		server := helpers.Must(NewServer(fixtures.Service))

		hd := http.Header{}
		hd.Set("Content-Type", request.ContentType)
		hd.Set("Accept", "not/acceptable")

		req := thttp.NewHTTPRequest(bytes.NewReader([]byte{}), hd)
		res := helpers.Must(Handle(server, req))
		require.Equal(t, res.Status(), http.StatusNotAcceptable)
	})

	t.Run("decode error", func(t *testing.T) {
		server := helpers.Must(NewServer(fixtures.Service))

		hd := http.Header{}
		hd.Set("Content-Type", request.ContentType)
		hd.Set("Accept", request.ContentType)

		// request with invalid payload
		req := thttp.NewHTTPRequest(bytes.NewReader([]byte{}), hd)
		res := helpers.Must(Handle(server, req))
		require.Equal(t, res.Status(), http.StatusBadRequest)
	})
}

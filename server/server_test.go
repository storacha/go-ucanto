package server

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/ipfs/go-cid"
	ipldprime "github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	ipldschema "github.com/ipld/go-ipld-prime/schema"
	"github.com/storacha-network/go-ucanto/client"
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/core/receipt"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/core/schema"
	"github.com/storacha-network/go-ucanto/principal/ed25519/signer"
	sdm "github.com/storacha-network/go-ucanto/server/datamodel"
	"github.com/storacha-network/go-ucanto/testing/helpers"
	"github.com/storacha-network/go-ucanto/transport/car/request"
	"github.com/storacha-network/go-ucanto/transport/car/response"
	thttp "github.com/storacha-network/go-ucanto/transport/http"
	"github.com/storacha-network/go-ucanto/ucan"
	"github.com/storacha-network/go-ucanto/validator"
	"github.com/stretchr/testify/require"
)

type uploadAddCaveats struct {
	Root ipld.Link
}

func (c uploadAddCaveats) Build() (ipld.Node, error) {
	np := basicnode.Prototype.Any
	nb := np.NewBuilder()
	ma, _ := nb.BeginMap(2)
	ma.AssembleKey().AssignString("root")
	ma.AssembleValue().AssignLink(c.Root)
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

func (ok uploadAddSuccess) Build() (ipld.Node, error) {
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

func TestHandlerNotFound(t *testing.T) {
	service := helpers.Must(signer.Generate())
	alice := helpers.Must(signer.Generate())
	space := helpers.Must(signer.Generate())

	server := helpers.Must(NewServer(service))
	conn := helpers.Must(client.NewConnection(service, server))

	rt := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
	capability := ucan.NewCapability(
		"upload/add",
		space.DID().String(),
		uploadAddCaveats{Root: rt},
	)

	invs := []invocation.Invocation{helpers.Must(invocation.Invoke(alice, service, capability))}
	resp := helpers.Must(client.Execute(invs, conn))
	rcptlnk, ok := resp.Get(invs[0].Link())
	require.True(t, ok, "missing receipt for invocation: %s", invs[0].Link())

	rcptsch := bytes.Join([][]byte{sdm.Schema(), []byte(`
		type Result union {
			| Any "ok"
			| HandlerNotFoundError "error"
		} representation keyed
	`)}, []byte("\n"))

	reader := helpers.Must(receipt.NewReceiptReader[ipld.Node, sdm.HandlerNotFoundErrorModel](rcptsch))
	rcpt := helpers.Must(reader.Read(rcptlnk, resp.Blocks()))

	result.MatchResultR0(rcpt.Out(), func(ipld.Node) {
		t.Fatalf("expected error: %s", invs[0].Link())
	}, func(rerr sdm.HandlerNotFoundErrorModel) {
		fmt.Printf("%s %+v\n", *rerr.Name, rerr)
		require.Equal(t, *rerr.Name, "HandlerNotFoundError")
	})
}

func TestSimpleHandler(t *testing.T) {
	service := helpers.Must(signer.Generate())
	alice := helpers.Must(signer.Generate())
	space := helpers.Must(signer.Generate())

	uploadadd := validator.NewCapability(
		"upload/add",
		schema.DIDString(),
		schema.Struct[uploadAddCaveats](uploadAddCaveatsType(), nil),
		nil,
	)

	server := helpers.Must(NewServer(
		service,
		WithServiceMethod(
			uploadadd.Can(),
			Provide(uploadadd, func(cap ucan.Capability[uploadAddCaveats], inv invocation.Invocation, ctx InvocationContext) (uploadAddSuccess, receipt.Effects, error) {
				return uploadAddSuccess{Root: cap.Nb().Root, Status: "done"}, nil, nil
			}),
		),
	))

	conn := helpers.Must(client.NewConnection(service, server))
	rt := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
	cap := uploadadd.New(space.DID().String(), uploadAddCaveats{Root: rt})
	invs := []invocation.Invocation{helpers.Must(invocation.Invoke(alice, service, cap))}
	resp := helpers.Must(client.Execute(invs, conn))

	// get the receipt link for the invocation from the response
	rcptlnk, ok := resp.Get(invs[0].Link())
	require.True(t, ok, "missing receipt for invocation: %s", invs[0].Link())

	rcptsch := bytes.Join([][]byte{sdm.Schema(), []byte(`
		type Result union {
			| UploadAddSuccess "ok"
			| Any "error"
		} representation keyed

		type UploadAddSuccess struct {
		  root Link
		  status String
		}
	`)}, []byte("\n"))

	reader := helpers.Must(receipt.NewReceiptReader[uploadAddSuccess, ipld.Node](rcptsch))
	rcpt := helpers.Must(reader.Read(rcptlnk, resp.Blocks()))

	result.MatchResultR0(rcpt.Out(), func(ok uploadAddSuccess) {
		fmt.Printf("%+v\n", ok)
		require.Equal(t, ok.Root, rt)
		require.Equal(t, ok.Status, "done")
	}, func(rerr ipld.Node) {
		t.Fatalf("unexpected error: %+v", rerr)
	})
}

func TestHandlerExecutionError(t *testing.T) {
	service := helpers.Must(signer.Generate())
	alice := helpers.Must(signer.Generate())
	space := helpers.Must(signer.Generate())

	uploadadd := validator.NewCapability(
		"upload/add",
		schema.DIDString(),
		schema.Struct[uploadAddCaveats](uploadAddCaveatsType(), nil),
		nil,
	)

	server := helpers.Must(NewServer(
		service,
		WithServiceMethod(
			uploadadd.Can(),
			Provide(uploadadd, func(cap ucan.Capability[uploadAddCaveats], inv invocation.Invocation, ctx InvocationContext) (uploadAddSuccess, receipt.Effects, error) {
				return uploadAddSuccess{}, nil, fmt.Errorf("test error")
			}),
		),
	))

	conn := helpers.Must(client.NewConnection(service, server))
	rt := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
	cap := uploadadd.New(space.DID().String(), uploadAddCaveats{Root: rt})
	invs := []invocation.Invocation{helpers.Must(invocation.Invoke(alice, service, cap))}
	resp := helpers.Must(client.Execute(invs, conn))
	rcptlnk, ok := resp.Get(invs[0].Link())
	require.True(t, ok, "missing receipt for invocation: %s", invs[0].Link())

	rcptsch := bytes.Join([][]byte{sdm.Schema(), []byte(`
		type Result union {
			| Any "ok"
			| HandlerExecutionError "error"
		} representation keyed
	`)}, []byte("\n"))

	reader := helpers.Must(receipt.NewReceiptReader[ipld.Node, sdm.HandlerExecutionErrorModel](rcptsch))
	rcpt := helpers.Must(reader.Read(rcptlnk, resp.Blocks()))

	result.MatchResultR0(rcpt.Out(), func(ipld.Node) {
		t.Fatalf("expected error: %s", invs[0].Link())
	}, func(rerr sdm.HandlerExecutionErrorModel) {
		fmt.Printf("%s %+v\n", *rerr.Name, rerr)
		require.Equal(t, *rerr.Name, "HandlerExecutionError")
	})
}

func TestHandleContentTypeError(t *testing.T) {
	service := helpers.Must(signer.Generate())
	server := helpers.Must(NewServer(service))

	hd := http.Header{}
	hd.Set("Content-Type", "unsupported/media")
	hd.Set("Accept", response.ContentType)

	req := thttp.NewHTTPRequest(bytes.NewReader([]byte{}), hd)
	res := helpers.Must(Handle(server, req))
	require.Equal(t, res.Status(), http.StatusUnsupportedMediaType)
}

func TestHandleAcceptError(t *testing.T) {
	service := helpers.Must(signer.Generate())
	server := helpers.Must(NewServer(service))

	hd := http.Header{}
	hd.Set("Content-Type", request.ContentType)
	hd.Set("Accept", "not/acceptable")

	req := thttp.NewHTTPRequest(bytes.NewReader([]byte{}), hd)
	res := helpers.Must(Handle(server, req))
	require.Equal(t, res.Status(), http.StatusNotAcceptable)
}

func TestHandleDecodeError(t *testing.T) {
	service := helpers.Must(signer.Generate())
	server := helpers.Must(NewServer(service))

	hd := http.Header{}
	hd.Set("Content-Type", request.ContentType)
	hd.Set("Accept", request.ContentType)

	// request with invalid payload
	req := thttp.NewHTTPRequest(bytes.NewReader([]byte{}), hd)
	res := helpers.Must(Handle(server, req))
	require.Equal(t, res.Status(), http.StatusBadRequest)
}

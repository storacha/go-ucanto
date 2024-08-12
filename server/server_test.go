package server

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ipfs/go-cid"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/storacha-network/go-ucanto/client"
	"github.com/storacha-network/go-ucanto/core/invocation"
	"github.com/storacha-network/go-ucanto/core/ipld"
	"github.com/storacha-network/go-ucanto/core/receipt"
	"github.com/storacha-network/go-ucanto/core/result"
	"github.com/storacha-network/go-ucanto/core/schema"
	"github.com/storacha-network/go-ucanto/principal/ed25519/signer"
	sdm "github.com/storacha-network/go-ucanto/server/datamodel"
	"github.com/storacha-network/go-ucanto/server/transaction"
	"github.com/storacha-network/go-ucanto/testing/helpers"
	"github.com/storacha-network/go-ucanto/ucan"
	"github.com/storacha-network/go-ucanto/validator"
)

type uploadAddCaveats struct {
	Root ipld.Link
}

func (c *uploadAddCaveats) Build() (map[string]ipld.Node, error) {
	data := map[string]ipld.Node{}
	b := basicnode.Prototype.Link.NewBuilder()
	err := b.AssignLink(c.Root)
	if err != nil {
		return nil, err
	}
	data["root"] = b.Build()
	return data, nil
}

type uploadAddSuccess struct {
	Status string
}

func (ok *uploadAddSuccess) Build() (ipld.Node, error) {
	np := basicnode.Prototype.Any
	nb := np.NewBuilder()
	ma, _ := nb.BeginMap(1)
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
		ucan.CaveatBuilder(&uploadAddCaveats{Root: rt}),
	)

	invs := []invocation.Invocation{helpers.Must(invocation.Invoke(alice, service, capability))}
	resp := helpers.Must(client.Execute(invs, conn))
	rcptlnk, ok := resp.Get(invs[0].Link())
	if !ok {
		t.Fatalf("missing receipt for invocation: %s", invs[0].Link())
	}

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
		if *rerr.Name != "HandlerNotFoundError" {
			t.Fatalf("unexpected error name: %s", *rerr.Name)
		}
	})
}

func TestSimpleHandler(t *testing.T) {
	service := helpers.Must(signer.Generate())
	alice := helpers.Must(signer.Generate())
	space := helpers.Must(signer.Generate())

	uploadadd := validator.NewCapability(
		"upload/add",
		schema.DID(),
		schema.Struct(&uploadAddCaveats{}, typ),
	)

	server := helpers.Must(NewServer(
		service,
		WithServiceMethod("upload/add", Provide(uploadadd, func(cap ucan.Capability[*uploadAddCaveats], inv invocation.Invocation, ctx InvocationContext) (transaction.Transaction[*uploadAddSuccess, ipld.Builder], error) {
			r := result.Ok[*uploadAddSuccess, ipld.Builder](&uploadAddSuccess{Status: "done"})
			return transaction.NewTransaction(r), nil
		})),
	))

	conn := helpers.Must(client.NewConnection(service, server))
	rt := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
	cap := uploadadd.New(space.DID().String(), &uploadAddCaveats{Root: rt})
	invs := []invocation.Invocation{helpers.Must(invocation.Invoke(alice, service, cap))}
	resp := helpers.Must(client.Execute(invs, conn))

	// get the receipt link for the invocation from the response
	rcptlnk, ok := resp.Get(invs[0].Link())
	if !ok {
		t.Fatalf("missing receipt for invocation: %s", invs[0].Link())
	}

	rcptsch := bytes.Join([][]byte{sdm.Schema(), []byte(`
		type Result union {
			| UploadAddSuccess "ok"
			| Any "error"
		} representation keyed

		type UploadAddSuccess struct {
		  status String
		}
	`)}, []byte("\n"))

	reader := helpers.Must(receipt.NewReceiptReader[*uploadAddSuccess, ipld.Node](rcptsch))
	rcpt := helpers.Must(reader.Read(rcptlnk, resp.Blocks()))

	result.MatchResultR0(rcpt.Out(), func(ok *uploadAddSuccess) {
		fmt.Printf("%+v\n", ok)
		if ok.Status != "done" {
			t.Fatalf("unexpected status: %s", ok.Status)
		}
	}, func(rerr ipld.Node) {
		t.Fatalf("unexpected error: %+v", rerr)
	})
}

func TestHandlerExecutionError(t *testing.T) {
	service := helpers.Must(signer.Generate())
	alice := helpers.Must(signer.Generate())
	space := helpers.Must(signer.Generate())

	rt := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
	nb := uploadAddCaveats{Root: rt}
	// TODO: this should be a descriptor not an instance
	cap := ucan.NewCapability("upload/add", space.DID().String(), &nb)

	server := helpers.Must(NewServer(
		service,
		WithServiceMethod("upload/add", Provide(cap, func(cap ucan.Capability[*uploadAddCaveats], inv invocation.Invocation, ctx InvocationContext) (transaction.Transaction[*uploadAddSuccess, ipld.Builder], error) {
			return nil, fmt.Errorf("test error")
		})),
	))
	conn := helpers.Must(client.NewConnection(service, server))

	invs := []invocation.Invocation{helpers.Must(invocation.Invoke(alice, service, cap))}
	resp := helpers.Must(client.Execute(invs, conn))
	rcptlnk, ok := resp.Get(invs[0].Link())
	if !ok {
		t.Fatalf("missing receipt for invocation: %s", invs[0].Link())
	}

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
		if *rerr.Name != "HandlerExecutionError" {
			t.Fatalf("unexpected error name: %s", *rerr.Name)
		}
	})
}

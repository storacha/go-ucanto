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
	"github.com/storacha-network/go-ucanto/principal"
	"github.com/storacha-network/go-ucanto/principal/ed25519/signer"
	sdm "github.com/storacha-network/go-ucanto/server/datamodel"
	"github.com/storacha-network/go-ucanto/server/transaction"
	"github.com/storacha-network/go-ucanto/transport/car"
	"github.com/storacha-network/go-ucanto/ucan"
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

func generateSigner(t testing.TB) principal.Signer {
	t.Helper()
	signer, err := signer.Generate()
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}
	return signer
}

func TestHandlerNotFound(t *testing.T) {
	service := generateSigner(t)
	alice := generateSigner(t)
	space := generateSigner(t)

	server, err := NewServer(service)
	if err != nil {
		t.Fatalf("creating service: %v", err)
	}

	codec := car.NewCAROutboundCodec()
	conn, err := client.NewConnection(service, codec, server)
	if err != nil {
		t.Fatalf("creating connection: %v", err)
	}

	rt := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
	capability := ucan.NewCapability(
		"upload/add",
		space.DID().String(),
		ucan.CaveatBuilder(&uploadAddCaveats{Root: rt}),
	)

	// create invocation(s) to perform a task with granted capabilities
	inv, _ := invocation.Invoke(alice, service, capability)
	invs := []invocation.Invocation{inv}

	// send the invocation(s) to the service
	resp, err := client.Execute(invs, conn)
	if err != nil {
		t.Fatalf("requesting service: %v", err)
	}

	// get the receipt link for the invocation from the response
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

	reader, err := receipt.NewReceiptReader[ipld.Node, sdm.HandlerNotFoundErrorModel](rcptsch)
	if err != nil {
		t.Fatalf("creating reader: %v", err)
	}

	// read the receipt for the invocation from the response
	rcpt, err := reader.Read(rcptlnk, resp.Blocks())
	if err != nil {
		t.Fatalf("reading receipt: %v", err)
	}

	result.MatchResultR0(rcpt.Out(), func(ipld.Node) {
		t.Fatalf("expected error: %s", invs[0].Link())
	}, func(rerr sdm.HandlerNotFoundErrorModel) {
		fmt.Printf("%+v\n", rerr)
	})
}

func TestSimpleHandler(t *testing.T) {
	service := generateSigner(t)
	alice := generateSigner(t)
	space := generateSigner(t)

	rt := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
	nb := uploadAddCaveats{Root: rt}
	// TODO: this should be a definition not an instance
	cap := ucan.NewCapability("upload/add", space.DID().String(), ucan.CaveatBuilder(&nb))

	server, err := NewServer(
		service,
		WithServiceMethod("upload/add", Provide(cap, func(cap ucan.Capability[ucan.CaveatBuilder], inv invocation.Invocation, ctx InvocationContext) (transaction.Transaction[*uploadAddSuccess, ipld.Builder], error) {
			return transaction.NewTransaction(result.Ok[*uploadAddSuccess, ipld.Builder](&uploadAddSuccess{Status: "done"})), nil
		})),
	)
	if err != nil {
		t.Fatalf("creating service: %v", err)
	}

	codec := car.NewCAROutboundCodec()
	conn, err := client.NewConnection(service, codec, server)
	if err != nil {
		t.Fatalf("creating connection: %v", err)
	}

	// create invocation(s) to perform a task with granted capabilities
	inv, _ := invocation.Invoke(alice, service, cap)
	invs := []invocation.Invocation{inv}

	// send the invocation(s) to the service
	resp, err := client.Execute(invs, conn)
	if err != nil {
		t.Fatalf("requesting service: %v", err)
	}

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

	reader, err := receipt.NewReceiptReader[*uploadAddSuccess, ipld.Node](rcptsch)
	if err != nil {
		t.Fatalf("creating reader: %v", err)
	}

	// read the receipt for the invocation from the response
	rcpt, err := reader.Read(rcptlnk, resp.Blocks())
	if err != nil {
		t.Fatalf("reading receipt: %v", err)
	}

	result.MatchResultR0(rcpt.Out(), func(ok *uploadAddSuccess) {
		fmt.Printf("%+v\n", ok)
		if ok.Status != "done" {
			t.Fatalf("unexpected status: %s", ok.Status)
		}
	}, func(rerr ipld.Node) {
		t.Fatalf("unexpected error: %+v", rerr)
	})
}

// func TestHandlerExecutionError(t *testing.T) {
// 	service, err := signer.Generate()
// 	if err != nil {
// 		t.Fatalf("generating service key: %v", err)
// 	}

// 	space, err := signer.Generate()
// 	if err != nil {
// 		t.Fatalf("generating space key: %v", err)
// 	}

// 	rt := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
// 	nb := uploadAddCaveats{root: rt}
// 	cap := ucan.NewCapability("upload/add", space.DID().String(), nb)
// 	hdlr := func(capability ucan.Capability[uploadAddCaveats], invocation invocation.Invocation, context InvocationContext) () {

// 	}

// 	definition := map[string]ServiceMethod[invocation.Invocation, ipld.Datamodeler, ipld.Datamodeler]{
// 		"upload/add": Provide(cap, hdlr),
// 	}

// 	svr, err := NewServer(service)
// 	if err != nil {
// 		t.Fatalf("creating service: %v", err)
// 	}
// }

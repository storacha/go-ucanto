package server

import (
	"log"
	"testing"

	"github.com/ipfs/go-cid"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/web3-storage/go-ucanto/client"
	"github.com/web3-storage/go-ucanto/core/invocation"
	"github.com/web3-storage/go-ucanto/core/ipld"
	"github.com/web3-storage/go-ucanto/principal/ed25519/signer"
	"github.com/web3-storage/go-ucanto/transport/car"
	"github.com/web3-storage/go-ucanto/ucan"
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

func TestHandlerNotFound(t *testing.T) {
	service, err := signer.Generate()
	if err != nil {
		t.Fatalf("generating service key: %v", err)
	}

	alice, err := signer.Generate()
	if err != nil {
		t.Fatalf("generating alice key: %v", err)
	}

	space, err := signer.Generate()
	if err != nil {
		t.Fatalf("generating space key: %v", err)
	}

	definition := map[string]ServiceMethod[invocation.Invocation, ipld.Datamodeler, ipld.Datamodeler]{}

	server, err := NewServer(service, definition)
	if err != nil {
		t.Fatalf("creating service: %v", err)
	}

	codec := car.NewCAROutboundCodec()
	conn, _ := client.NewConnection(service, codec, server)

	rt := cidlink.Link{Cid: cid.MustParse("bafkreiem4twkqzsq2aj4shbycd4yvoj2cx72vezicletlhi7dijjciqpui")}
	capability := ucan.NewCapability(
		"upload/add",
		space.DID().String(),
		ucan.CaveatBuilder(&uploadAddCaveats{Root: rt}),
	)

	// create invocation(s) to perform a task with granted capabilities
	inv, err := invocation.Invoke(alice, service, capability)
	invs := []invocation.Invocation{inv}

	// send the invocation(s) to the service
	resp, err := client.Execute(invs, conn)
	if err != nil {
		t.Fatalf("requesting service: %v", err)
	}

	log.Printf("%+v", resp)
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

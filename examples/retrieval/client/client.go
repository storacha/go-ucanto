package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"

	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-multihash"
	"github.com/storacha/go-ucanto/client/retrieval"
	"github.com/storacha/go-ucanto/core/dag/blockstore"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/ipld"
	"github.com/storacha/go-ucanto/core/receipt"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/examples/retrieval/capabilities/content"
	"github.com/storacha/go-ucanto/testing/fixtures"
)

func main() {
	// first obtain a delegation from service -> agent (Alice)
	dlg, err := content.Serve.Delegate(
		fixtures.Service,
		fixtures.Alice,
		fixtures.Service.DID().String(),
		content.ServeCaveats{},
	)
	if err != nil {
		panic(fmt.Errorf("delegating %s to %s: %w", content.Serve.Can(), fixtures.Alice.DID(), err))
	}

	if len(os.Args) < 2 {
		fmt.Println("missing hash argument")
		os.Exit(1)
	}

	digestStr := os.Args[1]
	_, digestBytes, err := multibase.Decode(digestStr)
	if err != nil {
		panic(fmt.Errorf("decoding multibase string: %w", err))
	}

	digest, err := multihash.Cast(digestBytes)
	if err != nil {
		panic(fmt.Errorf("decoding digest: %w", err))
	}

	url, err := url.Parse("http://localhost:3000/" + digestStr)
	if err != nil {
		panic(fmt.Errorf("parsing retrieval URL: %w", err))
	}

	conn, err := retrieval.NewConnection(fixtures.Service, url)
	if err != nil {
		panic(fmt.Errorf("creating connection: %w", err))
	}

	var byteRange []int
	if len(os.Args) == 4 {
		start, err := strconv.Atoi(os.Args[2])
		if err != nil {
			panic(fmt.Errorf("parsing byte range start value: %w", err))
		}
		end, err := strconv.Atoi(os.Args[3])
		if err != nil {
			panic(fmt.Errorf("parsing byte range end value: %w", err))
		}
		byteRange = []int{start, end}
	}

	inv, err := content.Serve.Invoke(
		fixtures.Alice,
		fixtures.Service,
		fixtures.Service.DID().String(),
		content.ServeCaveats{Digest: digest, Range: byteRange},
		delegation.WithProof(delegation.FromDelegation(dlg)),
	)
	if err != nil {
		panic(fmt.Errorf("creating invocation: %w", err))
	}

	xres, hres, err := retrieval.Execute(context.Background(), inv, conn)
	if err != nil {
		panic(fmt.Errorf("executing invocation: %w", err))
	}

	rcptLink, ok := xres.Get(inv.Link())
	if !ok {
		panic(errors.New("receipt for invocation not found"))
	}

	bs, err := blockstore.NewBlockReader(blockstore.WithBlocksIterator(xres.Blocks()))
	if err != nil {
		panic(fmt.Errorf("creating block reader: %w", err))
	}

	rcpt, err := receipt.NewAnyReceipt(rcptLink, bs)
	if err != nil {
		panic(fmt.Errorf("creating receipt: %w", err))
	}

	o, x := result.Unwrap(rcpt.Out())
	if x != nil {
		panic(fmt.Errorf("invocation failed: %+v", x))
	}

	_, err = ipld.Rebind[content.ServeOk](o, content.ServeTypeSystem.TypeByName("ServeOk"))
	if err != nil {
		panic(fmt.Errorf("decoding response: %w", err))
	}

	_, err = io.Copy(os.Stdout, hres.Body())
	if err != nil {
		panic(fmt.Errorf("printing body to stdout: %w", err))
	}
	fmt.Println()
}

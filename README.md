# go-ucanto

Ucanto UCAN RPC in Golang.

## Install

```console
go get github.com/storacha-network/go-ucanto
```

## Usage

### Client

```go
package main

import (
  "..."
)

// service URL & DID
serviceURL, _ := url.Parse("https://up.web3.storage")
servicePrincipal, _ := did.Parse("did:web:web3.storage")

// HTTP transport and CAR encoding
channel := http.NewHTTPChannel(serviceURL)
conn, _ := client.NewConnection(servicePrincipal, channel)

// private key to sign UCANs with
priv, _ := ioutil.ReadFile("path/to/private.key")
signer, _ := ed25519.Parse(priv)

audience := servicePrincipal

type StoreAddCaveats struct {
	Link ipld.Link
	Size uint64
}

func (c *StoreAddCaveats) Build() (datamodel.Node, error) {
	n := bindnode.Wrap(c, typ)
	return n.Representation(), nil
}

capability := ucan.NewCapability(
	"store/add",
	did.Parse("did:key:z6MkwDuRThQcyWjqNsK54yKAmzfsiH6BTkASyiucThMtHt1T").String(),
	&StoreAddCaveats{
		// TODO
	},
)

// create invocation(s) to perform a task with granted capabilities
inv, _ := invocation.Invoke(signer, audience, capability, delegation.WithProofs(...))
invocations := []invocation.Invocation{inv}

// send the invocation(s) to the service
resp, _ := client.Execute(invocations, conn)

// define datamodels for ok and error outcome
type OkModel struct {
  Status string
}
type ErrModel struct {
	Message string
}

// create new receipt reader, passing the IPLD schema for the result and the
// ok and error types
reader, _ := receipt.NewReceiptReader[OkModel, ErrModel]([]byte(`
	type Result union {
		| Ok "ok"
		| Err "error"
	} representation keyed

	type Ok struct {
		status String
	}

	type Err struct {
		message String
	}
`))

// get the receipt link for the invocation from the response
rcptlnk, _ := resp.Get(invocations[0].Link())
// read the receipt for the invocation from the response
rcpt, _ := reader.Read(rcptlnk, res.Blocks())

fmt.Println(rcpt.Out().Ok())
```

### Server

```go
package main

import (
	"..."
)

type TestEcho struct {
	Echo string
}

func (c *TestEcho) Build() (ipld.Node, error) {
	np := basicnode.Prototype.Any
	nb := np.NewBuilder()
	ma, _ := nb.BeginMap(1)
	ma.AssembleKey().AssignString("echo")
	ma.AssembleValue().AssignString(c.Echo)
	ma.Finish()
	return nb.Build(), nil
}

func EchoType() ipldschema.Type {
	ts, _ := ipldprime.LoadSchemaBytes([]byte(`
		type TestEcho struct {
			echo String
		}
	`))
	return ts.TypeByName("TestEcho")
}

func createServer(signer principal.Signer) (server.ServerView, error) {
	// Capability definition(s)
	testecho := validator.NewCapability(
		"test/echo",
		schema.DIDString(),
		schema.Struct[*TestEcho](EchoType()),
	)

	return server.NewServer(
		signer,
		// Handler definitions
		server.WithServiceMethod(testecho.Can(), server.Provide(testecho, func(cap ucan.Capability[*TestEcho], inv invocation.Invocation, ctx server.InvocationContext) (transaction.Transaction[*TestEcho, ipld.Builder], error) {
			r := result.Ok[*TestEcho, ipld.Builder](&TestEcho{Echo: cap.Nb().Echo})
			return transaction.NewTransaction(r), nil
		})),
	)
}

func main() {
	signer, _ := ed25519.Generate()
	server, _ := createServer(signer)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		res, _ := server.Request(uhttp.NewHTTPRequest(r.Body, r.Header))

		for key, vals := range res.Headers() {
			for _, v := range vals {
				w.Header().Add(key, v)
			}
		}

		if res.Status() != 0 {
			w.WriteHeader(res.Status())
		}

		io.Copy(w, res.Body())
	})

	listener, _ := net.Listen("tcp", ":0")

	port := listener.Addr().(*net.TCPAddr).Port
	fmt.Printf("{\"id\":\"%s\",\"url\":\"http://127.0.0.1:%d\"}\n", signer.DID().String(), port)

	http.Serve(listener, nil)
}
```

## API

[pkg.go.dev Reference](https://pkg.go.dev/github.com/storacha-network/go-ucanto)

## Related

* [Ucanto in Javascript](https://github.com/storacha-network/ucanto)

## Contributing

Feel free to join in. All welcome. Please [open an issue](https://github.com/storacha-network/go-ucanto/issues)!

## License

Dual-licensed under [MIT + Apache 2.0](LICENSE.md)

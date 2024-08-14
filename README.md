# go-ucanto

Ucanto UCAN RPC in Golang.

## Install

```console
go get github.com/storacha-network/go-ucanto
```

## Usage

```go
package main

import (
  "net/url"
  "ioutil"

  "github.com/storacha-network/go-ucanto/client"
  "github.com/storacha-network/go-ucanto/did"
  ed25519 "github.com/storacha-network/go-ucanto/principal/ed25519/signer"
  "github.com/storacha-network/go-ucanto/transport/car"
  "github.com/storacha-network/go-ucanto/transport/http"
  "github.com/storacha-network/go-ucanto/core/delegation"
  "github.com/storacha-network/go-ucanto/core/invocation"
  "github.com/storacha-network/go-ucanto/core/receipt"
)

// service URL & DID
serviceURL, _ := url.Parse("https://up.web3.storage")
servicePrincipal, _ := did.Parse("did:web:web3.storage")

// HTTP transport and CAR encoding
channel := http.NewHTTPChannel(serviceURL)
codec := car.NewCAROutboundCodec()

conn, _ := client.NewConnection(servicePrincipal, codec, channel)

// private key to sign UCANs with
priv, _ := ioutil.ReadFile("path/to/private.key")
signer, _ := ed25519.Parse(priv)

audience := servicePrincipal

type StoreAddCaveats struct {
  Link ipld.Link
  Size uint64
}

func (c *StoreAddCaveats) Build() (map[string]datamodel.Node, error) {
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

## API

[pkg.go.dev Reference](https://pkg.go.dev/github.com/storacha-network/go-ucanto)

## Related

* [Ucanto in Javascript](https://github.com/storacha-network/ucanto)

## Contributing

Feel free to join in. All welcome. Please [open an issue](https://github.com/storacha-network/go-ucanto/issues)!

## License

Dual-licensed under [MIT + Apache 2.0](LICENSE.md)

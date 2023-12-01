# go-ucanto

Ucanto UCAN RPC in Golang.

```go
package main

import (
  "net/url"
  "ioutil"

  "github.com/alanshaw/go-ucanto/client"
  "github.com/alanshaw/go-ucanto/did"
  ed25519 "github.com/alanshaw/go-ucanto/principal/ed25519/signer"
  "github.com/alanshaw/go-ucanto/transport/car"
  "github.com/alanshaw/go-ucanto/transport/http"
  "github.com/alanshaw/go-ucanto/core/delegation"
  "github.com/alanshaw/go-ucanto/core/invocation"
  "github.com/alanshaw/go-ucanto/core/receipt"
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
invocations := []invocation.Invocation{
  invocation.Invoke(signer, audience, capability, delegation.WithProofs(...))
}

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
rcpt := reader.Read(rcptlnk, res.Blocks())

fmt.Println(rcpt.Out.Ok)
```

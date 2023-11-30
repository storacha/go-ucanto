# go-ucanto

Ucanto UCAN RPC in Golang.

```go
package main

import (
  "net/url"
  "ioutil"

  "github.com/alanshaw/go-ucanto/client"
  "github.com/alanshaw/go-ucanto/did"
  "github.com/alanshaw/go-ucanto/principal/ed25519/signer"
  "github.com/alanshaw/go-ucanto/transport/car"
  "github.com/alanshaw/go-ucanto/transport/http"
  "github.com/alanshaw/go-ucanto/core/invocation"
  "github.com/alanshaw/go-ucanto/core/receipt"
)

// service URL & DID
u, _ := url.Parse("https://up.web3.storage")
p, _ := did.Parse("did:web:web3.storage")

// HTTP transport and CAR encoding
ch := http.NewHTTPChannel(u)
co := car.NewCAROutboundCodec()

cn, _ := client.NewConnection(p, co, ch)

// private key to sign UCANs with
ss, _ := ioutil.ReadFile("path/to/private.key")
snr, _ := signer.Parse(ss)

// create an invocation to perform a task with granted capabilities
inv := invocation.Invoke(snr, p, ...TODO)

typ := []byte(`
  type Result union {
    | Ok "ok"
    | Err "error"
  } representation keyed

  type Ok struct {
    status String (rename "Status")
  }

  type Err struct {
    message String (rename "Message")
  }
`)
rr := receipt.NewReceiptReader[O, X](typ)

// send the invocation to the service
rcpt, _ := client.Execute[O, X](inv, rr, cn)

fmt.Println(rcpt.Out.Ok)
```

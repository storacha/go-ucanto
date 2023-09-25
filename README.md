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
)

u, _ := url.Parse("https://up.web3.storage")
p, _ := did.Parse("did:web:web3.storage")

ch := http.NewHTTPChannel(u)
co := car.NewCAROutboundCodec()

cn, _ := client.NewConnection(p, co, ch)

ss, _ := ioutil.ReadFile("path/to/private.key")
snr, _ := signer.Parse(ss)

// TODO: define inv
// TODO TODO
client.Execute(inv, cn)
```

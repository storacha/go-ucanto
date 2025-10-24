package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/receipt/fx"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/core/result/failure"
	"github.com/storacha/go-ucanto/examples/retrieval/capabilities/content"
	"github.com/storacha/go-ucanto/server"
	"github.com/storacha/go-ucanto/server/retrieval"
	"github.com/storacha/go-ucanto/testing/fixtures"
	thttp "github.com/storacha/go-ucanto/transport/http"
	"github.com/storacha/go-ucanto/ucan"
)

func main() {
	server, err := retrieval.NewServer(
		fixtures.Service,
		retrieval.WithServiceMethod(
			content.Serve.Can(),
			retrieval.Provide(
				content.Serve,
				func(ctx context.Context, cap ucan.Capability[content.ServeCaveats], inv invocation.Invocation, ictx server.InvocationContext, req retrieval.Request) (result.Result[content.ServeOk, failure.IPLDBuilderFailure], fx.Effects, retrieval.Response, error) {
					filepath := path.Join(".", "data", req.URL.String()+".blob")
					file, err := os.Open(filepath)
					if err != nil {
						return nil, nil, retrieval.Response{}, err
					}
					info, err := file.Stat()
					if err != nil {
						return nil, nil, retrieval.Response{}, err
					}
					nb := cap.Nb()
					response := retrieval.Response{Status: http.StatusOK, Headers: http.Header{}, Body: file}
					response.Headers.Set("Content-Length", fmt.Sprintf("%d", info.Size()))
					if len(nb.Range) > 0 { // handle byte range request
						start, end := nb.Range[0], nb.Range[1]
						length := end - start + 1
						response.Headers.Set("Content-Length", fmt.Sprintf("%d", length))
						response.Headers.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, info.Size()))
						response.Status = http.StatusPartialContent
						response.Body = newFileSectionReader(file, start, length)
					}
					result := result.Ok[content.ServeOk, failure.IPLDBuilderFailure](content.ServeOk(nb))
					return result, nil, response, nil
				},
			),
		),
	)
	if err != nil {
		panic(fmt.Errorf("creating UCAN server: %w", err))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/{digest}", func(w http.ResponseWriter, r *http.Request) {
		resp, err := server.Request(r.Context(), thttp.NewInboundRequest(r.URL, r.Body, r.Header))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		for name, values := range resp.Headers() {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}
		w.WriteHeader(resp.Status())
		body := resp.Body()
		io.Copy(w, body)
		body.Close()
	})

	httpServer := &http.Server{
		Addr:           ":3000",
		Handler:        mux,
		MaxHeaderBytes: 2 * 1024,
	}

	fmt.Printf("ID: %s\n", fixtures.Service.DID())
	fmt.Println("Listening on: http://localhost:3000")
	err = httpServer.ListenAndServe()
	if err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}
}

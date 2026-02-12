package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestChannelPropagatesTraceContext(t *testing.T) {
	const (
		requestTraceIDHex  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		requestSpanIDHex   = "bbbbbbbbbbbbbbbb"
		responseTraceIDHex = "cccccccccccccccccccccccccccccccc"
		responseSpanIDHex  = "dddddddddddddddd"
		responseTrace      = "00-" + responseTraceIDHex + "-" + responseSpanIDHex + "-01"
		expectedRequest    = "00-" + requestTraceIDHex + "-" + requestSpanIDHex + "-01"
	)

	var seenRequestTrace string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenRequestTrace = r.Header.Get("traceparent")
		w.Header().Set("traceparent", responseTrace)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parsing server URL: %v", err)
	}

	channel := NewChannel(endpoint, WithClient(server.Client()))

	restoreProp := setTraceContextPropagator()
	t.Cleanup(restoreProp)

	ctx := context.Background()
	ctx = trace.ContextWithSpanContext(ctx, newSpanContext(t, requestTraceIDHex, requestSpanIDHex))

	res, err := channel.Request(ctx, NewRequest(http.NoBody, nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	t.Cleanup(func() { res.Body().Close() })

	if seenRequestTrace != expectedRequest {
		t.Fatalf("expected traceparent %q, got %q", expectedRequest, seenRequestTrace)
	}

	responseCtx, ok := res.(*Response)
	if !ok {
		t.Fatalf("expected *Response, got %T", res)
	}
	sc := trace.SpanContextFromContext(responseCtx.Context())
	expectedTraceID := mustTraceIDFromHex(t, responseTraceIDHex)
	if sc.TraceID() != expectedTraceID {
		t.Fatalf("expected response trace ID %s, got %s", expectedTraceID, sc.TraceID())
	}
	expectedSpanID := mustSpanIDFromHex(t, responseSpanIDHex)
	if sc.SpanID() != expectedSpanID {
		t.Fatalf("expected response span ID %s, got %s", expectedSpanID, sc.SpanID())
	}
}

func newSpanContext(t *testing.T, traceIDHex, spanIDHex string) trace.SpanContext {
	t.Helper()
	traceID := mustTraceIDFromHex(t, traceIDHex)
	spanID := mustSpanIDFromHex(t, spanIDHex)
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
}

func mustTraceIDFromHex(t *testing.T, hex string) trace.TraceID {
	t.Helper()
	traceID, err := trace.TraceIDFromHex(hex)
	if err != nil {
		t.Fatalf("parsing trace ID: %v", err)
	}
	return traceID
}

func mustSpanIDFromHex(t *testing.T, hex string) trace.SpanID {
	t.Helper()
	spanID, err := trace.SpanIDFromHex(hex)
	if err != nil {
		t.Fatalf("parsing span ID: %v", err)
	}
	return spanID
}

func TestChannelErrorIncludesResponseBody(t *testing.T) {
	const errorBody = `{"error":"InternalServerError","message":"something went wrong"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorBody))
	}))
	t.Cleanup(server.Close)

	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parsing server URL: %v", err)
	}

	channel := NewChannel(endpoint, WithClient(server.Client()))

	_, err = channel.Request(context.Background(), NewRequest(http.NoBody, nil))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), errorBody) {
		t.Fatalf("expected error to contain response body %q, got: %s", errorBody, err.Error())
	}
}

func TestChannelErrorHandlesEmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parsing server URL: %v", err)
	}

	channel := NewChannel(endpoint, WithClient(server.Client()))

	_, err = channel.Request(context.Background(), NewRequest(http.NoBody, nil))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expected := fmt.Sprintf("HTTP Request failed. POST %s → 500", server.URL)
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestChannelErrorTruncatesLongBody(t *testing.T) {
	longBody := strings.Repeat("x", 5000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(longBody))
	}))
	t.Cleanup(server.Close)

	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parsing server URL: %v", err)
	}

	channel := NewChannel(endpoint, WithClient(server.Client()))

	_, err = channel.Request(context.Background(), NewRequest(http.NoBody, nil))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "... (truncated)") {
		t.Fatalf("expected error to contain truncation marker, got: %s", err.Error())
	}
	if strings.Contains(err.Error(), longBody) {
		t.Fatal("expected error to NOT contain the full body")
	}
}

func setTraceContextPropagator() func() {
	prev := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return func() {
		otel.SetTextMapPropagator(prev)
	}
}

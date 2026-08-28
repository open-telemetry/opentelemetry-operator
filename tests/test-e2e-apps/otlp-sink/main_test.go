// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	// Register the gzip compressor for the test's gRPC client.
	grpcgzip "google.golang.org/grpc/encoding/gzip"
)

// testStack starts a full sink (real otlpreceiver) on loopback ports and
// returns it with its ingest endpoints.
func testStack(t *testing.T) (s *sinkStack, grpcAddr, httpAddr string) {
	t.Helper()
	s = newSinkStack()
	grpcAddr, httpAddr = freeAddr(t), freeAddr(t)
	require.NoError(t, s.startReceiver(t.Context(), grpcAddr, httpAddr), "start receiver")
	return s, grpcAddr, httpAddr
}

// freeAddr reserves a loopback port and returns host:port for the receiver to
// bind. The listener is closed first, so a tiny race exists, which is
// acceptable in tests.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().String()
}

func testTraces(spanName string) ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "test")
	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName(spanName)
	span.SetKind(ptrace.SpanKindServer)
	return td
}

// post sends body to url and returns the response with its body still open,
// so callers can assert on it; closing is deferred to test cleanup.
func post(t *testing.T, url string, body []byte, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// querySpanNames runs the query handler and returns the span names it served.
// It takes require.TestingT so it can be called both directly and from inside
// a require.EventuallyWithT condition.
func querySpanNames(t require.TestingT, s *sinkStack) []string {
	rec := httptest.NewRecorder()
	s.queryMux().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/received/traces", nil))
	require.Equalf(t, http.StatusOK, rec.Code, "query body: %s", rec.Body)
	var envelope struct {
		Requests []json.RawMessage `json:"requests"`
		Dropped  int               `json:"dropped"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope), "decode envelope")
	unmarshaler := &ptrace.JSONUnmarshaler{}
	var names []string
	for _, raw := range envelope.Requests {
		td, err := unmarshaler.UnmarshalTraces(raw)
		require.NoError(t, err, "decode request")
		for i := range td.ResourceSpans().Len() {
			rs := td.ResourceSpans().At(i)
			for j := range rs.ScopeSpans().Len() {
				ss := rs.ScopeSpans().At(j)
				for k := range ss.Spans().Len() {
					names = append(names, ss.Spans().At(k).Name())
				}
			}
		}
	}
	return names
}

// waitForSpan polls the query API until the span name shows up (ingest is
// asynchronous relative to the HTTP response only in the gRPC case, but poll
// uniformly to be safe).
func waitForSpan(t *testing.T, s *sinkStack, name string) {
	t.Helper()
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Contains(c, querySpanNames(c, s), name)
	}, 10*time.Second, 50*time.Millisecond, "span %q not received", name)
}

func TestHTTPIngestProtobuf(t *testing.T) {
	s, _, httpAddr := testStack(t)
	body, err := (&ptrace.ProtoMarshaler{}).MarshalTraces(testTraces("proto-span"))
	require.NoError(t, err)
	resp := post(t, fmt.Sprintf("http://%s/v1/traces", httpAddr), body, map[string]string{"Content-Type": "application/x-protobuf"})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	waitForSpan(t, s, "proto-span")
}

// TestHTTPIngestGzip covers the collector's default behavior: the otlphttp
// exporter gzip-compresses request bodies.
func TestHTTPIngestGzip(t *testing.T) {
	s, _, httpAddr := testStack(t)
	body, err := (&ptrace.ProtoMarshaler{}).MarshalTraces(testTraces("gzip-span"))
	require.NoError(t, err)
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err = gz.Write(body)
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	resp := post(t, fmt.Sprintf("http://%s/v1/traces", httpAddr), buf.Bytes(), map[string]string{
		"Content-Type":     "application/x-protobuf",
		"Content-Encoding": "gzip",
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	waitForSpan(t, s, "gzip-span")
}

func TestHTTPIngestJSON(t *testing.T) {
	s, _, httpAddr := testStack(t)
	body, err := (&ptrace.JSONMarshaler{}).MarshalTraces(testTraces("json-span"))
	require.NoError(t, err)
	resp := post(t, fmt.Sprintf("http://%s/v1/traces", httpAddr), body, map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	waitForSpan(t, s, "json-span")
}

func TestHTTPIngestRejectsCorruptBody(t *testing.T) {
	_, _, httpAddr := testStack(t)
	resp := post(t, fmt.Sprintf("http://%s/v1/traces", httpAddr), []byte{0xff, 0xff, 0xff}, map[string]string{"Content-Type": "application/x-protobuf"})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestGRPCIngestGzip exports over gRPC with gzip compression, the collector's
// default for the otlp (gRPC) exporter.
func TestGRPCIngestGzip(t *testing.T) {
	s, grpcAddr, _ := testStack(t)

	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := ptraceotlp.NewGRPCClient(conn)
	req := ptraceotlp.NewExportRequestFromTraces(testTraces("grpc-span"))
	_, err = client.Export(t.Context(), req, grpc.UseCompressor(grpcgzip.Name))
	require.NoError(t, err, "gRPC export")
	waitForSpan(t, s, "grpc-span")
}

func TestHealthz(t *testing.T) {
	s := newSinkStack()
	rec := httptest.NewRecorder()
	s.queryMux().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, rec.Code)
}

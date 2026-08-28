// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// otlp-sink is a minimal OTLP backend for e2e tests. Ingest is the collector's
// own otlpreceiver (gRPC on :4317, HTTP protobuf/JSON on :4318, compression
// included) feeding consumertest sinks — the same building blocks the
// collector-contrib e2e tests use — so no OTLP protocol handling is
// reimplemented here. Everything received is kept in memory and served back as
// OTLP-JSON on the query port (:4319) so tests can assert on exactly what was
// received:
//
//	GET /received/traces  -> {"requests": [<OTLP-JSON traces request>...], "dropped": 0}
//	GET /received/metrics -> {"requests": [<OTLP-JSON metrics request>...], "dropped": 0}
//	GET /received/logs    -> {"requests": [<OTLP-JSON logs request>...], "dropped": 0}
//	GET /healthz          -> 200
//
// Every export request is also logged to stdout, so the sink's pod log doubles
// as a receive journal when a test fails.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

const (
	grpcEndpoint = "0.0.0.0:4317"
	httpEndpoint = "0.0.0.0:4318"
	queryAddr    = ":4319"

	// maxStoredRequests bounds each signal's stored requests; e2e tests are
	// short-lived so this is never hit in practice, it only guards the sink
	// against a runaway exporter. Overflow is counted and reported as "dropped".
	maxStoredRequests = 10000
)

// sink accumulates one signal's export requests and serves them as OTLP-JSON.
type sink[T any] struct {
	name    string
	dropped atomic.Int64

	all     func() []T
	marshal func(T) ([]byte, error)
}

// guard enforces maxStoredRequests and logs each accepted request; returning a
// non-nil error would make the receiver report a failure to the exporter, so
// overflow is silently counted instead.
func (s *sink[T]) guard(forward func() error) error {
	if len(s.all()) >= maxStoredRequests {
		s.dropped.Add(1)
		return nil
	}
	if err := forward(); err != nil {
		return err
	}
	log.Printf("received %s export request #%d", s.name, len(s.all()))
	return nil
}

// query serves the accumulated requests as {"requests": [...], "dropped": N}.
// The per-request payloads are already OTLP-JSON, so they go in as RawMessage
// rather than being re-encoded.
func (s *sink[T]) query() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		all := s.all()
		envelope := struct {
			// Non-nil even when empty, so the field marshals as [] and not null.
			Requests []json.RawMessage `json:"requests"`
			Dropped  int64             `json:"dropped"`
		}{
			Requests: make([]json.RawMessage, 0, len(all)),
			Dropped:  s.dropped.Load(),
		}
		for _, req := range all {
			data, err := s.marshal(req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			envelope.Requests = append(envelope.Requests, data)
		}
		body, err := json.Marshal(envelope)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// sinkStack is the assembled sink: consumertest sinks for each signal, fed by
// an otlpreceiver, exposed through the query API.
type sinkStack struct {
	traces  *sink[ptrace.Traces]
	metrics *sink[pmetric.Metrics]
	logs    *sink[plog.Logs]

	tracesSink  *consumertest.TracesSink
	metricsSink *consumertest.MetricsSink
	logsSink    *consumertest.LogsSink
}

func newSinkStack() *sinkStack {
	s := &sinkStack{
		tracesSink:  new(consumertest.TracesSink),
		metricsSink: new(consumertest.MetricsSink),
		logsSink:    new(consumertest.LogsSink),
	}
	s.traces = &sink[ptrace.Traces]{
		name:    "traces",
		all:     s.tracesSink.AllTraces,
		marshal: (&ptrace.JSONMarshaler{}).MarshalTraces,
	}
	s.metrics = &sink[pmetric.Metrics]{
		name:    "metrics",
		all:     s.metricsSink.AllMetrics,
		marshal: (&pmetric.JSONMarshaler{}).MarshalMetrics,
	}
	s.logs = &sink[plog.Logs]{
		name:    "logs",
		all:     s.logsSink.AllLogs,
		marshal: (&plog.JSONMarshaler{}).MarshalLogs,
	}
	return s
}

// startReceiver assembles and starts the otlpreceiver, wiring each signal
// through its guard into the consumertest sink. This mirrors how the
// collector-contrib e2e tests construct their sinks (see startUpSinks in
// processor/k8sattributesprocessor/e2e_test.go).
func (s *sinkStack) startReceiver(ctx context.Context, grpcEndpoint, httpEndpoint string) error {
	f := otlpreceiver.NewFactory()
	cfg := f.CreateDefaultConfig()
	conf := confmap.NewFromStringMap(map[string]any{
		"protocols": map[string]any{
			"grpc": map[string]any{"endpoint": grpcEndpoint},
			"http": map[string]any{"endpoint": httpEndpoint},
		},
	})
	if err := conf.Unmarshal(cfg); err != nil {
		return fmt.Errorf("configure receiver: %w", err)
	}
	set := receivertest.NewNopSettings(f.Type())

	tracesConsumer, err := consumer.NewTraces(func(ctx context.Context, td ptrace.Traces) error {
		return s.traces.guard(func() error { return s.tracesSink.ConsumeTraces(ctx, td) })
	})
	if err != nil {
		return err
	}
	metricsConsumer, err := consumer.NewMetrics(func(ctx context.Context, md pmetric.Metrics) error {
		return s.metrics.guard(func() error { return s.metricsSink.ConsumeMetrics(ctx, md) })
	})
	if err != nil {
		return err
	}
	logsConsumer, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
		return s.logs.guard(func() error { return s.logsSink.ConsumeLogs(ctx, ld) })
	})
	if err != nil {
		return err
	}

	// The three Create* calls return the same shared receiver instance; it only
	// needs to be started once.
	if _, err := f.CreateTraces(ctx, set, cfg, tracesConsumer); err != nil {
		return fmt.Errorf("create traces receiver: %w", err)
	}
	if _, err := f.CreateMetrics(ctx, set, cfg, metricsConsumer); err != nil {
		return fmt.Errorf("create metrics receiver: %w", err)
	}
	rcvr, err := f.CreateLogs(ctx, set, cfg, logsConsumer)
	if err != nil {
		return fmt.Errorf("create logs receiver: %w", err)
	}
	return rcvr.Start(ctx, componenttest.NewNopHost())
}

// queryMux builds the query API.
func (s *sinkStack) queryMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /received/traces", s.traces.query())
	mux.HandleFunc("GET /received/metrics", s.metrics.query())
	mux.HandleFunc("GET /received/logs", s.logs.query())
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

func main() {
	s := newSinkStack()
	if err := s.startReceiver(context.Background(), grpcEndpoint, httpEndpoint); err != nil {
		log.Fatalf("start otlp receiver: %v", err)
	}
	log.Printf("OTLP/gRPC listening on %s, OTLP/HTTP on %s", grpcEndpoint, httpEndpoint)

	log.Printf("query API listening on %s", queryAddr)
	srv := &http.Server{Addr: queryAddr, Handler: s.queryMux(), ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(srv.ListenAndServe())
}

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// The OTLP sink (tests/test-e2e-apps/otlp-sink) is an in-cluster OTLP backend for
// tests that need to assert on telemetry semantically: a collector under test exports
// OTLP to the sink, and the test reads back exactly what was received through the
// sink's query API, rather than inspecting a debug exporter's pod logs.

const (
	otlpSinkName      = "otlp-sink"
	otlpSinkGRPCPort  = 4317
	otlpSinkHTTPPort  = 4318
	otlpSinkQueryPort = 4319

	// defaultSinkTimeout bounds how long the retrying sink helpers wait for a check
	// to pass. It is longer than the Prometheus equivalent because an auto-instrumented
	// workload has to start its runtime and agent before the first export.
	defaultSinkTimeout = 3 * time.Minute
	// defaultSinkInterval is the delay between attempts of the retrying sink helpers.
	defaultSinkInterval = 2 * time.Second

	// OTLPSinkHTTPEndpoint is the sink's in-namespace OTLP/HTTP endpoint, for use in
	// collector exporter configs deployed to the sink's namespace.
	OTLPSinkHTTPEndpoint = "http://otlp-sink:4318"
	// OTLPSinkGRPCEndpoint is the sink's in-namespace OTLP/gRPC endpoint.
	OTLPSinkGRPCEndpoint = "http://otlp-sink:4317"
)

// OTLPSink is a handle to a deployed OTLP sink in one namespace.
type OTLPSink struct {
	Namespace string
}

// DeployOTLPSink deploys the OTLP sink (Deployment + Service, both named
// "otlp-sink") into ns and waits for it to become ready.
func DeployOTLPSink(ctx context.Context, t *testing.T, cfg *envconf.Config, ns, image string) *OTLPSink {
	t.Helper()
	c := CRClient(t, cfg)
	labels := map[string]string{"app": otlpSinkName}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: otlpSinkName, Namespace: ns},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  otlpSinkName,
						Image: image,
						Ports: []corev1.ContainerPort{
							{Name: "otlp-grpc", ContainerPort: otlpSinkGRPCPort},
							{Name: "otlp-http", ContainerPort: otlpSinkHTTPPort},
							{Name: "query", ContainerPort: otlpSinkQueryPort},
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{Path: "/healthz", Port: intstr.FromInt32(otlpSinkQueryPort)},
							},
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("10m"),
								corev1.ResourceMemory: resource.MustParse("32Mi"),
							},
						},
					}},
				},
			},
		},
	}
	require.NoError(t, c.Create(ctx, dep), "create otlp-sink deployment")

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: otlpSinkName, Namespace: ns},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{Name: "otlp-grpc", Port: otlpSinkGRPCPort, AppProtocol: new("grpc")},
				{Name: "otlp-http", Port: otlpSinkHTTPPort},
				{Name: "query", Port: otlpSinkQueryPort},
			},
		},
	}
	require.NoError(t, c.Create(ctx, svc), "create otlp-sink service")

	WaitForDeployment(ctx, t, cfg, ns, otlpSinkName, 2*time.Minute)
	return &OTLPSink{Namespace: ns}
}

// Span is one received span with its resource and scope flattened for assertions.
type Span struct {
	Resource map[string]string
	Scope    string
	Span     ptrace.Span
}

// Metric is one received metric with its resource and scope flattened for assertions.
type Metric struct {
	Resource map[string]string
	Scope    string
	Metric   pmetric.Metric
}

// received decodes the sink query API's {"requests": [...], "dropped": N} envelope;
// each request is protojson-encoded OTLP.
type received struct {
	Requests []json.RawMessage `json:"requests"`
	Dropped  int               `json:"dropped"`
}

func (s *OTLPSink) fetch(ctx context.Context, cfg *envconf.Config, signal string, each func(raw []byte) error) error {
	body, err := ServiceHTTPGet(ctx, cfg, s.Namespace, otlpSinkName, otlpSinkQueryPort, "/received/"+signal)
	if err != nil {
		return err
	}
	var envelope received
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("decode sink %s envelope: %w", signal, err)
	}
	if envelope.Dropped > 0 {
		return fmt.Errorf("sink dropped %d %s export requests; assertions would be unreliable", envelope.Dropped, signal)
	}
	for _, raw := range envelope.Requests {
		if err := each(raw); err != nil {
			return fmt.Errorf("decode sink %s request: %w", signal, err)
		}
	}
	return nil
}

// Spans fetches every span the sink has received so far.
func (s *OTLPSink) Spans(ctx context.Context, cfg *envconf.Config) ([]Span, error) {
	var spans []Span
	unmarshaler := &ptrace.JSONUnmarshaler{}
	err := s.fetch(ctx, cfg, "traces", func(raw []byte) error {
		td, err := unmarshaler.UnmarshalTraces(raw)
		if err != nil {
			return err
		}
		for i := range td.ResourceSpans().Len() {
			rs := td.ResourceSpans().At(i)
			res := attributesMap(rs.Resource().Attributes())
			for j := range rs.ScopeSpans().Len() {
				ss := rs.ScopeSpans().At(j)
				for k := range ss.Spans().Len() {
					spans = append(spans, Span{Resource: res, Scope: ss.Scope().Name(), Span: ss.Spans().At(k)})
				}
			}
		}
		return nil
	})
	return spans, err
}

// Metrics fetches every metric the sink has received so far.
func (s *OTLPSink) Metrics(ctx context.Context, cfg *envconf.Config) ([]Metric, error) {
	var metrics []Metric
	unmarshaler := &pmetric.JSONUnmarshaler{}
	err := s.fetch(ctx, cfg, "metrics", func(raw []byte) error {
		md, err := unmarshaler.UnmarshalMetrics(raw)
		if err != nil {
			return err
		}
		for i := range md.ResourceMetrics().Len() {
			rm := md.ResourceMetrics().At(i)
			res := attributesMap(rm.Resource().Attributes())
			for j := range rm.ScopeMetrics().Len() {
				sm := rm.ScopeMetrics().At(j)
				for k := range sm.Metrics().Len() {
					metrics = append(metrics, Metric{Resource: res, Scope: sm.Scope().Name(), Metric: sm.Metrics().At(k)})
				}
			}
		}
		return nil
	})
	return metrics, err
}

// attributesMap flattens OTLP attributes into strings for assertions.
func attributesMap(attrs pcommon.Map) map[string]string {
	m := make(map[string]string, attrs.Len())
	for k, v := range attrs.All() {
		m[k] = v.AsString()
	}
	return m
}

// EventuallySpans polls the sink until check accepts the received spans. On
// timeout the test fails with the last fetch or check error. Keeping the workload
// producing telemetry while this polls is the caller's business.
func (s *OTLPSink) EventuallySpans(ctx context.Context, t *testing.T, cfg *envconf.Config, check func([]Span) error) {
	t.Helper()
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		spans, err := s.Spans(ctx, cfg)
		if err != nil {
			c.Errorf("spans in otlp-sink: %v", err)
			return
		}
		if err := check(spans); err != nil {
			c.Errorf("spans in otlp-sink: %v", err)
		}
	}, defaultSinkTimeout, defaultSinkInterval)
}

// EventuallyMetrics polls the sink until check accepts the received metrics. See
// EventuallySpans.
func (s *OTLPSink) EventuallyMetrics(ctx context.Context, t *testing.T, cfg *envconf.Config, check func([]Metric) error) {
	t.Helper()
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		metrics, err := s.Metrics(ctx, cfg)
		if err != nil {
			c.Errorf("metrics in otlp-sink: %v", err)
			return
		}
		if err := check(metrics); err != nil {
			c.Errorf("metrics in otlp-sink: %v", err)
		}
	}, defaultSinkTimeout, defaultSinkInterval)
}

// AttrMatch is one expected attribute: at least one of Keys must be present
// (alternatives cover agents that straddle semconv versions, e.g. http.method vs
// http.request.method), and when Value is non-empty the attribute that is present
// must have that value (compared via pcommon.Value.AsString).
type AttrMatch struct {
	Keys  []string
	Value string
}

// matches reports whether attrs satisfies the AttrMatch.
func (am AttrMatch) matches(attrs map[string]string) bool {
	for _, k := range am.Keys {
		v, ok := attrs[k]
		if !ok {
			continue
		}
		if am.Value == "" || v == am.Value {
			return true
		}
	}
	return false
}

// SpanMatch describes an expected span for HasSpan.
type SpanMatch struct {
	// Resource attributes the span's resource must carry (exact values).
	Resource map[string]string
	// ResourcePresent are resource attribute keys that must exist with any value.
	ResourcePresent []string
	// Attrs are expected span attributes; see AttrMatch.
	Attrs []AttrMatch
	// Kind, when non-zero, is the required span kind.
	Kind ptrace.SpanKind
	// StatusNotError requires the span status code to not be Error.
	StatusNotError bool
}

// HasSpan returns a check (for EventuallySpans) that passes when some received span
// matches want. On failure it reports an inventory of what WAS received, so a test
// timeout is diagnosable from the failure message alone.
func HasSpan(want SpanMatch) func([]Span) error {
	return func(spans []Span) error {
		for _, s := range spans {
			if spanMatches(want, s) {
				return nil
			}
		}
		return fmt.Errorf("no span matched %+v\nreceived: %s", want, spanInventory(spans))
	}
}

func spanMatches(want SpanMatch, s Span) bool {
	for k, v := range want.Resource {
		if s.Resource[k] != v {
			return false
		}
	}
	for _, k := range want.ResourcePresent {
		if _, ok := s.Resource[k]; !ok {
			return false
		}
	}
	if want.Kind != ptrace.SpanKindUnspecified && s.Span.Kind() != want.Kind {
		return false
	}
	if want.StatusNotError && s.Span.Status().Code() == ptrace.StatusCodeError {
		return false
	}
	if len(want.Attrs) > 0 {
		attrs := attributesMap(s.Span.Attributes())
		for _, am := range want.Attrs {
			if !am.matches(attrs) {
				return false
			}
		}
	}
	return true
}

// MetricMatch describes an expected metric for HasMetric.
type MetricMatch struct {
	// NamesAnyOf requires the metric name to be one of these. Use multiple names
	// when the name depends on the instrumentation's semconv version (e.g.
	// process.runtime.jvm.memory.usage vs jvm.memory.used).
	NamesAnyOf []string
	// Resource attributes the metric's resource must carry (exact values).
	Resource map[string]string
	// Type, when not MetricTypeEmpty, is the required metric type.
	Type pmetric.MetricType
	// Unit, when non-empty, is the required unit. Leave empty when NamesAnyOf
	// spans semconv versions with different units (e.g. http.server.duration in
	// ms vs http.server.request.duration in s).
	Unit string
	// NonZero requires at least one data point with a non-zero value (for
	// histograms, a non-zero sample count).
	NonZero bool
}

// HasMetric returns a check (for EventuallyMetrics) that passes when some received
// metric matches want, reporting the received-metric inventory on failure.
func HasMetric(want MetricMatch) func([]Metric) error {
	return func(metrics []Metric) error {
		for _, m := range metrics {
			if metricMatches(want, m) {
				return nil
			}
		}
		return fmt.Errorf("no metric matched %+v\nreceived: %s", want, metricInventory(metrics))
	}
}

func metricMatches(want MetricMatch, m Metric) bool {
	if len(want.NamesAnyOf) > 0 && !slices.Contains(want.NamesAnyOf, m.Metric.Name()) {
		return false
	}
	for k, v := range want.Resource {
		if m.Resource[k] != v {
			return false
		}
	}
	if want.Type != pmetric.MetricTypeEmpty && m.Metric.Type() != want.Type {
		return false
	}
	if want.Unit != "" && m.Metric.Unit() != want.Unit {
		return false
	}
	if want.NonZero && !hasNonZeroDataPoint(m.Metric) {
		return false
	}
	return true
}

// hasNonZeroDataPoint reports whether some data point carries a non-zero value:
// the numeric value for gauges and sums, the sample count for histograms and
// summaries.
func hasNonZeroDataPoint(m pmetric.Metric) bool {
	nonZeroNumber := func(dps pmetric.NumberDataPointSlice) bool {
		for i := range dps.Len() {
			dp := dps.At(i)
			if dp.IntValue() != 0 || dp.DoubleValue() != 0 {
				return true
			}
		}
		return false
	}
	switch m.Type() {
	case pmetric.MetricTypeEmpty:
		return false
	case pmetric.MetricTypeGauge:
		return nonZeroNumber(m.Gauge().DataPoints())
	case pmetric.MetricTypeSum:
		return nonZeroNumber(m.Sum().DataPoints())
	case pmetric.MetricTypeHistogram:
		for i := range m.Histogram().DataPoints().Len() {
			if m.Histogram().DataPoints().At(i).Count() > 0 {
				return true
			}
		}
	case pmetric.MetricTypeExponentialHistogram:
		for i := range m.ExponentialHistogram().DataPoints().Len() {
			if m.ExponentialHistogram().DataPoints().At(i).Count() > 0 {
				return true
			}
		}
	case pmetric.MetricTypeSummary:
		for i := range m.Summary().DataPoints().Len() {
			if m.Summary().DataPoints().At(i).Count() > 0 {
				return true
			}
		}
	}
	return false
}

// spanInventory summarizes received spans for failure messages: totals, the
// service.name values seen, a bounded sample of span names, and one full
// resource, so identity mismatches are diagnosable from the message alone.
func spanInventory(spans []Span) string {
	if len(spans) == 0 {
		return "0 spans"
	}
	services := map[string]bool{}
	names := map[string]bool{}
	// Prefer a server span as the sample: it is what tests usually assert on,
	// and agents often emit internal spans alongside it.
	sample := spans[0]
	for _, s := range spans {
		services[s.Resource["service.name"]] = true
		names[s.Span.Name()] = true
		if s.Span.Kind() == ptrace.SpanKindServer && sample.Span.Kind() != ptrace.SpanKindServer {
			sample = s
		}
	}
	return fmt.Sprintf("%d spans; service.name values %v; span names %s\nsample %v span resource: %v\nsample span attributes: %v",
		len(spans), slices.Sorted(maps.Keys(services)), boundedList(names, 20),
		sample.Span.Kind(), sample.Resource, attributesMap(sample.Span.Attributes()))
}

// metricInventory summarizes received metrics for failure messages.
func metricInventory(metrics []Metric) string {
	if len(metrics) == 0 {
		return "0 metrics"
	}
	services := map[string]bool{}
	names := map[string]bool{}
	for _, m := range metrics {
		services[m.Resource["service.name"]] = true
		names[fmt.Sprintf("%s(%s,%q)", m.Metric.Name(), m.Metric.Type(), m.Metric.Unit())] = true
	}
	return fmt.Sprintf("%d metrics; service.name values %v; metric name(type,unit) %s\nsample resource: %v",
		len(metrics), slices.Sorted(maps.Keys(services)), boundedList(names, 50), metrics[0].Resource)
}

func boundedList(set map[string]bool, limit int) string {
	all := slices.Sorted(maps.Keys(set))
	if len(all) <= limit {
		return fmt.Sprintf("%v", all)
	}
	return fmt.Sprintf("%v (and %d more)", all[:limit], len(all)-limit)
}

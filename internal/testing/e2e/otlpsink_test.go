// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func sinkTestConfig(t *testing.T, signal, body string, status int) *envconf.Config {
	t.Helper()
	return proxyTestConfig(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/namespaces/test-ns/services/http:otlp-sink:4319/proxy/received/"+signal, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, err := w.Write([]byte(body))
		assert.NoError(t, err)
	})
}

func TestOTLPSinkSpans(t *testing.T) {
	cfg := sinkTestConfig(t, "traces", `{"requests":[
		{"resourceSpans":[
			{"resource":{"attributes":[
				{"key":"service.name","value":{"stringValue":"app"}},
				{"key":"attempt","value":{"intValue":"3"}},
				{"key":"active","value":{"boolValue":true}}
			]},"scopeSpans":[
				{"scope":{"name":"http"},"spans":[{"name":"GET /","kind":2,"attributes":[{"key":"http.response.status_code","value":{"intValue":"200"}}]},{"name":"GET /health"}]},
				{"scope":{"name":"db"},"spans":[{"name":"SELECT","kind":3}]}
			]},
			{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"other"}}]},"scopeSpans":[{"scope":{"name":"queue"},"spans":[{"name":"consume"}]}]}
		]},
		{"resourceSpans":[{"scopeSpans":[{"spans":[{"name":"last export"}]}]}]}
	],"dropped":0}`, http.StatusOK)
	sink := &OTLPSink{Namespace: "test-ns"}
	spans, err := sink.Spans(t.Context(), cfg)
	require.NoError(t, err)
	require.Len(t, spans, 5)
	assert.Equal(t, []string{"http", "http", "db", "queue", ""}, []string{spans[0].Scope, spans[1].Scope, spans[2].Scope, spans[3].Scope, spans[4].Scope})
	assert.Equal(t, map[string]string{"service.name": "app", "attempt": "3", "active": "true"}, spans[0].Resource)
	assert.Equal(t, spans[0].Resource, spans[1].Resource)
	assert.Equal(t, spans[0].Resource, spans[2].Resource)
	assert.Equal(t, map[string]string{"service.name": "other"}, spans[3].Resource)
	assert.Empty(t, spans[4].Resource)
	assert.Equal(t, "GET /health", spans[1].Span.Name())
	assert.Equal(t, "SELECT", spans[2].Span.Name())
	assert.Equal(t, "last export", spans[4].Span.Name())

	sink.EventuallySpans(t.Context(), t, cfg, HasSpan(SpanMatch{
		Resource:       map[string]string{"service.name": "app"},
		Attrs:          []AttrMatch{{Keys: []string{"http.response.status_code"}, Value: "200"}},
		Kind:           ptrace.SpanKindServer,
		StatusNotError: true,
	}), time.Second, 10*time.Millisecond)
}

func TestOTLPSinkMetrics(t *testing.T) {
	cfg := sinkTestConfig(t, "metrics", `{"requests":[
		{"resourceMetrics":[
			{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"app"}}]},"scopeMetrics":[
				{"scope":{"name":"runtime"},"metrics":[{"name":"memory","unit":"By","gauge":{"dataPoints":[{"asInt":"9"}]}},{"name":"threads","gauge":{"dataPoints":[{"asInt":"2"}]}}]},
				{"scope":{"name":"http"},"metrics":[{"name":"duration","unit":"s","histogram":{"dataPoints":[{"count":"3","sum":0.5}]}}]}
			]},
			{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"other"}}]},"scopeMetrics":[{"scope":{"name":"cpu"},"metrics":[{"name":"cpu.time","sum":{"dataPoints":[{"asDouble":1.5}]}}]}]}
		]},
		{"resourceMetrics":[{"scopeMetrics":[{"metrics":[{"name":"last export","gauge":{}}]}]}]}
	],"dropped":0}`, http.StatusOK)
	sink := &OTLPSink{Namespace: "test-ns"}
	metrics, err := sink.Metrics(t.Context(), cfg)
	require.NoError(t, err)
	require.Len(t, metrics, 5)
	assert.Equal(t, []string{"runtime", "runtime", "http", "cpu", ""}, []string{metrics[0].Scope, metrics[1].Scope, metrics[2].Scope, metrics[3].Scope, metrics[4].Scope})
	assert.Equal(t, map[string]string{"service.name": "app"}, metrics[0].Resource)
	assert.Equal(t, metrics[0].Resource, metrics[1].Resource)
	assert.Equal(t, metrics[0].Resource, metrics[2].Resource)
	assert.Equal(t, map[string]string{"service.name": "other"}, metrics[3].Resource)
	assert.Empty(t, metrics[4].Resource)
	assert.Equal(t, int64(9), metrics[0].Metric.Gauge().DataPoints().At(0).IntValue())
	assert.Equal(t, "threads", metrics[1].Metric.Name())
	assert.Equal(t, 1.5, metrics[3].Metric.Sum().DataPoints().At(0).DoubleValue())
	assert.Equal(t, "last export", metrics[4].Metric.Name())

	sink.EventuallyMetrics(t.Context(), t, cfg, HasMetric(MetricMatch{
		NamesAnyOf: []string{"duration"},
		Resource:   map[string]string{"service.name": "app"},
		Type:       pmetric.MetricTypeHistogram,
		Unit:       "s",
		NonZero:    true,
	}), time.Second, 10*time.Millisecond)
}

func TestOTLPSinkResponses(t *testing.T) {
	for _, signal := range []string{"traces", "metrics"} {
		t.Run(signal, func(t *testing.T) {
			for _, tt := range []struct {
				name      string
				body      string
				status    int
				wantError string
			}{
				{name: "empty", body: `{"requests":[],"dropped":0}`, status: http.StatusOK},
				{name: "invalid envelope", body: `{`, status: http.StatusOK, wantError: "decode sink " + signal + " envelope"},
				{name: "dropped exports", body: `{"requests":[],"dropped":3}`, status: http.StatusOK, wantError: "sink dropped 3 " + signal + " export requests"},
				{name: "invalid OTLP request", body: `{"requests":[{"resourceSpans":"invalid","resourceMetrics":"invalid"}]}`, status: http.StatusOK, wantError: "decode sink " + signal + " request"},
				{name: "proxy unavailable", body: `{"message":"unavailable"}`, status: http.StatusBadGateway, wantError: "GET service otlp-sink port 4319"},
			} {
				t.Run(tt.name, func(t *testing.T) {
					cfg := sinkTestConfig(t, signal, tt.body, tt.status)
					sink := &OTLPSink{Namespace: "test-ns"}
					var err error
					if signal == "traces" {
						var spans []Span
						spans, err = sink.Spans(t.Context(), cfg)
						assert.Empty(t, spans)
					} else {
						var metrics []Metric
						metrics, err = sink.Metrics(t.Context(), cfg)
						assert.Empty(t, metrics)
					}
					if tt.wantError != "" {
						require.ErrorContains(t, err, tt.wantError)
					} else {
						require.NoError(t, err)
					}
				})
			}
		})
	}
}

func TestOTLPSinkEventually(t *testing.T) {
	for _, tt := range []struct {
		signal string
		body   string
	}{
		{signal: "traces", body: `{"requests":[{"resourceSpans":[{"scopeSpans":[{"spans":[{"name":"GET /"}]}]}]}]}`},
		{signal: "metrics", body: `{"requests":[{"resourceMetrics":[{"scopeMetrics":[{"metrics":[{"name":"memory","gauge":{}}]}]}]}]}`},
	} {
		t.Run(tt.signal, func(t *testing.T) {
			t.Parallel()
			var attempts atomic.Int32
			cfg := proxyTestConfig(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/namespaces/test-ns/services/http:otlp-sink:4319/proxy/received/"+tt.signal, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				body := tt.body
				switch attempts.Add(1) {
				case 1:
					http.Error(w, "temporarily unavailable", http.StatusBadGateway)
					return
				case 2:
					body = "{"
				case 3:
					body = `{"requests":[]}`
				}
				_, err := w.Write([]byte(body))
				assert.NoError(t, err)
			})
			sink := &OTLPSink{Namespace: "test-ns"}
			if tt.signal == "traces" {
				sink.EventuallySpans(t.Context(), t, cfg, HasSpan(SpanMatch{}), 5*time.Second, 10*time.Millisecond)
			} else {
				sink.EventuallyMetrics(t.Context(), t, cfg, HasMetric(MetricMatch{}), 5*time.Second, 10*time.Millisecond)
			}
			assert.Equal(t, int32(4), attempts.Load())
		})
	}
}

func TestHasSpan(t *testing.T) {
	for _, tt := range []struct {
		name   string
		want   SpanMatch
		status ptrace.StatusCode
		match  bool
	}{
		{name: "empty matcher", match: true},
		{
			name: "all clauses and alternative attribute names",
			want: SpanMatch{
				Resource: map[string]string{"service.name": "app"}, ResourcePresent: []string{"empty"},
				Attrs: []AttrMatch{
					{Keys: []string{"http.request.method", "http.method"}, Value: "GET"},
					{Keys: []string{"http.status_code"}, Value: "200"},
				},
				Kind: ptrace.SpanKindServer, StatusNotError: true,
			},
			match: true,
		},
		{name: "wrong resource value", want: SpanMatch{Resource: map[string]string{"service.name": "other"}}},
		{name: "missing resource", want: SpanMatch{Resource: map[string]string{"missing": "value"}}},
		{name: "missing required resource key", want: SpanMatch{ResourcePresent: []string{"missing"}}},
		{name: "wrong span kind", want: SpanMatch{Kind: ptrace.SpanKindClient}},
		{name: "error status rejected", want: SpanMatch{StatusNotError: true}, status: ptrace.StatusCodeError},
		{name: "error status allowed by default", status: ptrace.StatusCodeError, match: true},
		{name: "OK status allowed", want: SpanMatch{StatusNotError: true}, status: ptrace.StatusCodeOk, match: true},
		{name: "attribute presence", want: SpanMatch{Attrs: []AttrMatch{{Keys: []string{"http.method"}}}}, match: true},
		{name: "missing attribute", want: SpanMatch{Attrs: []AttrMatch{{Keys: []string{"missing"}}}}},
		{name: "wrong attribute value", want: SpanMatch{Attrs: []AttrMatch{{Keys: []string{"http.method"}, Value: "POST"}}}},
		{name: "all attributes required", want: SpanMatch{Attrs: []AttrMatch{{Keys: []string{"http.method"}, Value: "GET"}, {Keys: []string{"missing"}}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			span := ptrace.NewSpan()
			span.SetName("GET /")
			span.SetKind(ptrace.SpanKindServer)
			span.Status().SetCode(tt.status)
			span.Attributes().PutStr("http.method", "GET")
			span.Attributes().PutInt("http.status_code", 200)
			err := HasSpan(tt.want)([]Span{{Resource: map[string]string{"service.name": "app", "empty": ""}, Span: span}})
			if tt.match {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "no span matched")
				assert.Contains(t, err.Error(), "GET /")
			}
		})
	}

	t.Run("clauses must match the same span", func(t *testing.T) {
		server := ptrace.NewSpan()
		server.SetKind(ptrace.SpanKindServer)
		client := ptrace.NewSpan()
		client.SetKind(ptrace.SpanKindClient)
		check := HasSpan(SpanMatch{Resource: map[string]string{"service.name": "app"}, Kind: ptrace.SpanKindServer})
		require.Error(t, check([]Span{
			{Resource: map[string]string{"service.name": "other"}, Span: server},
			{Resource: map[string]string{"service.name": "app"}, Span: client},
		}))
		require.NoError(t, check([]Span{{Resource: map[string]string{"service.name": "other"}, Span: client}, {Resource: map[string]string{"service.name": "app"}, Span: server}}))
	})

	t.Run("no spans", func(t *testing.T) {
		require.ErrorContains(t, HasSpan(SpanMatch{})(nil), "0 spans")
	})
}

func TestHasMetric(t *testing.T) {
	for _, tt := range []struct {
		name  string
		want  MetricMatch
		match bool
	}{
		{name: "empty matcher", match: true},
		{name: "all clauses", want: MetricMatch{NamesAnyOf: []string{"http.server.request.duration", "http.server.duration"}, Resource: map[string]string{"service.name": "app"}, Type: pmetric.MetricTypeHistogram, Unit: "ms", NonZero: true}, match: true},
		{name: "wrong name", want: MetricMatch{NamesAnyOf: []string{"memory"}}},
		{name: "wrong resource value", want: MetricMatch{Resource: map[string]string{"service.name": "other"}}},
		{name: "missing resource", want: MetricMatch{Resource: map[string]string{"missing": "value"}}},
		{name: "wrong type", want: MetricMatch{Type: pmetric.MetricTypeSum}},
		{name: "wrong unit", want: MetricMatch{Unit: "s"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			metric := pmetric.NewMetric()
			metric.SetName("http.server.duration")
			metric.SetUnit("ms")
			metric.SetEmptyHistogram().DataPoints().AppendEmpty().SetCount(2)
			err := HasMetric(tt.want)([]Metric{{Resource: map[string]string{"service.name": "app"}, Metric: metric}})
			if tt.match {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "no metric matched")
				assert.Contains(t, err.Error(), "http.server.duration")
			}
		})
	}

	t.Run("clauses must match the same metric", func(t *testing.T) {
		memory := pmetric.NewMetric()
		memory.SetName("memory")
		cpu := pmetric.NewMetric()
		cpu.SetName("cpu")
		err := HasMetric(MetricMatch{NamesAnyOf: []string{"memory"}, Resource: map[string]string{"service.name": "app"}})([]Metric{
			{Resource: map[string]string{"service.name": "other"}, Metric: memory},
			{Resource: map[string]string{"service.name": "app"}, Metric: cpu},
		})
		require.Error(t, err)
	})

	t.Run("no metrics", func(t *testing.T) {
		require.ErrorContains(t, HasMetric(MetricMatch{})(nil), "0 metrics")
	})
}

func TestMetricNonZero(t *testing.T) {
	for _, tt := range []struct {
		name  string
		setup func(pmetric.Metric)
		match bool
	}{
		{name: "empty metric", setup: func(_ pmetric.Metric) {}},
		{name: "empty gauge", setup: func(m pmetric.Metric) { m.SetEmptyGauge() }},
		{name: "zero gauge", setup: func(m pmetric.Metric) { m.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(0) }},
		{name: "negative integer gauge", setup: func(m pmetric.Metric) { m.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(-1) }, match: true},
		{name: "double gauge after zero", setup: func(m pmetric.Metric) {
			dps := m.SetEmptyGauge().DataPoints()
			dps.AppendEmpty().SetDoubleValue(0)
			dps.AppendEmpty().SetDoubleValue(0.5)
		}, match: true},
		{name: "zero sum", setup: func(m pmetric.Metric) { m.SetEmptySum().DataPoints().AppendEmpty().SetDoubleValue(0) }},
		{name: "double sum", setup: func(m pmetric.Metric) { m.SetEmptySum().DataPoints().AppendEmpty().SetDoubleValue(0.5) }, match: true},
		{name: "integer sum", setup: func(m pmetric.Metric) { m.SetEmptySum().DataPoints().AppendEmpty().SetIntValue(2) }, match: true},
		{name: "histogram without samples", setup: func(m pmetric.Metric) { m.SetEmptyHistogram().DataPoints().AppendEmpty().SetSum(2) }},
		{name: "histogram with zero sum and samples", setup: func(m pmetric.Metric) { m.SetEmptyHistogram().DataPoints().AppendEmpty().SetCount(1) }, match: true},
		{name: "exponential histogram without samples", setup: func(m pmetric.Metric) { m.SetEmptyExponentialHistogram().DataPoints().AppendEmpty() }},
		{name: "exponential histogram with samples", setup: func(m pmetric.Metric) { m.SetEmptyExponentialHistogram().DataPoints().AppendEmpty().SetCount(1) }, match: true},
		{name: "summary without samples", setup: func(m pmetric.Metric) { m.SetEmptySummary().DataPoints().AppendEmpty() }},
		{name: "summary with samples", setup: func(m pmetric.Metric) { m.SetEmptySummary().DataPoints().AppendEmpty().SetCount(1) }, match: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			metric := pmetric.NewMetric()
			tt.setup(metric)
			metrics := []Metric{{Metric: metric}}
			err := HasMetric(MetricMatch{NonZero: true})(metrics)
			if tt.match {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.NoError(t, HasMetric(MetricMatch{})(metrics))
		})
	}
}

func TestSpanInventory(t *testing.T) {
	client := ptrace.NewSpan()
	client.SetName("outbound")
	client.SetKind(ptrace.SpanKindClient)
	server := ptrace.NewSpan()
	server.SetName("GET /")
	server.SetKind(ptrace.SpanKindServer)
	server.Attributes().PutStr("http.method", "GET")
	inventory := spanInventory([]Span{
		{Resource: map[string]string{"service.name": "z-client"}, Span: client},
		{Resource: map[string]string{"service.name": "a-server", "pod": "server-pod"}, Span: server},
	})
	assert.Contains(t, inventory, "2 spans")
	assert.Contains(t, inventory, "[a-server z-client]")
	assert.Contains(t, inventory, "server-pod")
	assert.Contains(t, inventory, "http.method:GET")
	assert.Contains(t, inventory, "outbound")
}

func TestBoundedList(t *testing.T) {
	assert.Equal(t, "[]", boundedList(nil, 2))
	assert.Equal(t, "[a b]", boundedList(map[string]bool{"b": true, "a": true}, 2))
	assert.Equal(t, "[a b] (and 1 more)", boundedList(map[string]bool{"c": true, "a": true, "b": true}, 2))
}

func TestOTLPSinkObjects(t *testing.T) {
	deployment, service := otlpSinkObjects("test-ns", "test-sink:latest")
	require.NotNil(t, deployment.Spec.Selector)
	assert.Equal(t, map[string]string{"app": "otlp-sink"}, deployment.Spec.Selector.MatchLabels)
	assert.Equal(t, deployment.Spec.Selector.MatchLabels, deployment.Spec.Template.Labels)
	assert.Equal(t, deployment.Spec.Template.Labels, service.Spec.Selector)
	assert.Equal(t, "otlp-sink", deployment.Name)
	assert.Equal(t, "test-ns", deployment.Namespace)
	assert.Equal(t, deployment.Name, service.Name)
	assert.Equal(t, deployment.Namespace, service.Namespace)
	require.Len(t, deployment.Spec.Template.Spec.Containers, 1)
	container := deployment.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "test-sink:latest", container.Image)
	assert.Equal(t, []corev1.ContainerPort{{Name: "otlp-grpc", ContainerPort: 4317}, {Name: "otlp-http", ContainerPort: 4318}, {Name: "query", ContainerPort: 4319}}, container.Ports)
	require.NotNil(t, container.ReadinessProbe)
	assert.Equal(t, &corev1.HTTPGetAction{Path: "/healthz", Port: intstr.FromInt32(4319)}, container.ReadinessProbe.HTTPGet)
	assert.Equal(t, resource.MustParse("10m"), container.Resources.Requests[corev1.ResourceCPU])
	assert.Equal(t, resource.MustParse("32Mi"), container.Resources.Requests[corev1.ResourceMemory])
	assert.Equal(t, []corev1.ServicePort{{Name: "otlp-grpc", Port: 4317, AppProtocol: new("grpc")}, {Name: "otlp-http", Port: 4318}, {Name: "query", Port: 4319}}, service.Spec.Ports)
	assert.Equal(t, "http://otlp-sink:4318", OTLPSinkHTTPEndpoint)
	assert.Equal(t, "http://otlp-sink:4317", OTLPSinkGRPCEndpoint)
}

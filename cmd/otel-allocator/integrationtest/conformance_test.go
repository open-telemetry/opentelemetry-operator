// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package integrationtest

import (
	"maps"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"gopkg.in/yaml.v3"

	taconfig "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
)

// conformanceTestdata is the relabel conformance suite's fixture directory,
// shared with this suite. Fixtures opt into integration testing by adding an
// integration.yaml; the conformance suite ignores that file.
const conformanceTestdata = "../internal/conformance/testdata"

// integrationSpec is the integration-only data attached to a shared fixture.
type integrationSpec struct {
	// Exposition is the Prometheus text every target in the fixture serves.
	Exposition string `yaml:"exposition"`
	// Expect lists metrics that must be emitted, each with relabel-produced
	// labels that must appear on them (as data point or resource attributes).
	Expect []metricExpectation `yaml:"expect"`
	// ExpectAbsent lists metric/label combinations that must NOT appear — e.g.
	// metrics from a target the relabeling dropped.
	ExpectAbsent []metricExpectation `yaml:"expect_absent"`
}

type metricExpectation struct {
	Metric string            `yaml:"metric"`
	Labels map[string]string `yaml:"labels"`
}

// TestSharedConformanceFixtures runs the conformance fixtures that carry an
// integration.yaml through the full TA + receiver pipeline and asserts on the
// emitted OTLP. The scrape config (prometheus.yaml) is shared verbatim with the
// relabel conformance suite; only the targets are repointed at a live mock.
func TestSharedConformanceFixtures(t *testing.T) {
	specs, err := filepath.Glob(filepath.Join(conformanceTestdata, "*", "integration.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, specs, "no integration-enabled conformance fixtures found")

	for _, specPath := range specs {
		dir := filepath.Dir(specPath)
		t.Run(filepath.Base(dir), func(t *testing.T) {
			// Not parallel: each case runs a full target allocator, and the
			// allocator uses process-global state (e.g. the prometheus
			// common-config secret-marshal flag) that is unsafe to drive from
			// several allocator instances at once. One TA per process is also how
			// it runs in production.
			spec := loadIntegrationSpec(t, specPath)
			scrapeConfigs := loadScrapeConfigs(t, filepath.Join(dir, "prometheus.yaml"))

			mock := startMockTarget(t, spec.Exposition)
			repointTargets(scrapeConfigs, mock.Host)
			// Conformance fixtures set no scrape_interval (it defaults to 1m); scrape
			// fast so the first scrape lands inside the test window.
			configureFastScrape(scrapeConfigs)

			taEndpoint := startTargetAllocator(t, scrapeConfigs)
			sink := new(consumertest.MetricsSink)
			startReceiver(t, taEndpoint, sink)

			require.Eventuallyf(t, func() bool {
				metrics := collectMetrics(sink)
				for _, e := range spec.Expect {
					if !hasMetricWithLabels(metrics, e.Metric, e.Labels) {
						return false
					}
				}
				return true
			}, 30*time.Second, 250*time.Millisecond, "expected metrics/labels not emitted")

			// The expected metrics are present, so at least one scrape cycle has
			// completed; anything the relabeling dropped must therefore be absent.
			metrics := collectMetrics(sink)
			for _, e := range spec.ExpectAbsent {
				assert.Falsef(t, hasMetricWithLabels(metrics, e.Metric, e.Labels),
					"metric %q with labels %v should be absent (target dropped by relabeling)", e.Metric, e.Labels)
			}
		})
	}
}

func loadIntegrationSpec(t *testing.T, path string) integrationSpec {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var spec integrationSpec
	require.NoError(t, yaml.Unmarshal(data, &spec))
	require.NotEmpty(t, spec.Expect, "integration.yaml must list expected metrics")
	return spec
}

func loadScrapeConfigs(t *testing.T, path string) []*promconfig.ScrapeConfig {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	cfg, err := promconfig.Load(string(data), taconfig.NopLogger)
	require.NoError(t, err)
	return cfg.ScrapeConfigs
}

// repointTargets rewrites every static target's address to addr, so fixtures
// authored with placeholder addresses can be scraped from a live mock. Fixtures
// whose relabeling depends on specific addresses should not carry an
// integration.yaml.
func repointTargets(scrapeConfigs []*promconfig.ScrapeConfig, addr string) {
	for _, sc := range scrapeConfigs {
		for _, sdc := range sc.ServiceDiscoveryConfigs {
			static, ok := sdc.(discovery.StaticConfig)
			if !ok {
				continue
			}
			for _, group := range static {
				for i := range group.Targets {
					group.Targets[i][model.AddressLabel] = model.LabelValue(addr)
				}
			}
		}
	}
}

// configureFastScrape forces a short scrape interval on every job so live
// scraping completes within the test window.
func configureFastScrape(scrapeConfigs []*promconfig.ScrapeConfig) {
	for _, sc := range scrapeConfigs {
		sc.ScrapeInterval = model.Duration(200 * time.Millisecond)
		sc.ScrapeTimeout = model.Duration(200 * time.Millisecond)
	}
}

// collectMetrics flattens the sink's OTLP batches into a map from metric name to
// the label sets observed for it — each number data point's attributes merged over
// its resource attributes.
func collectMetrics(sink *consumertest.MetricsSink) map[string][]map[string]string {
	byName := map[string][]map[string]string{}
	for _, md := range sink.AllMetrics() {
		rms := md.ResourceMetrics()
		for i := 0; i < rms.Len(); i++ {
			res := attrMap(rms.At(i).Resource().Attributes())
			sms := rms.At(i).ScopeMetrics()
			for j := 0; j < sms.Len(); j++ {
				ms := sms.At(j).Metrics()
				for k := 0; k < ms.Len(); k++ {
					m := ms.At(k)
					for _, dpAttrs := range numberDataPointAttrs(m) {
						labels := map[string]string{}
						maps.Copy(labels, res)
						maps.Copy(labels, dpAttrs)
						byName[m.Name()] = append(byName[m.Name()], labels)
					}
				}
			}
		}
	}
	return byName
}

// hasMetricWithLabels reports whether some label set of the named metric contains
// all of want.
func hasMetricWithLabels(metrics map[string][]map[string]string, name string, want map[string]string) bool {
	for _, labels := range metrics[name] {
		if subset(want, labels) {
			return true
		}
	}
	return false
}

// subset reports whether every key/value in want is present in have.
func subset(want, have map[string]string) bool {
	for k, v := range want {
		if have[k] != v {
			return false
		}
	}
	return true
}

func attrMap(m pcommon.Map) map[string]string {
	out := map[string]string{}
	m.Range(func(k string, v pcommon.Value) bool {
		out[k] = v.AsString()
		return true
	})
	return out
}

func numberDataPointAttrs(m pmetric.Metric) []map[string]string {
	var dps pmetric.NumberDataPointSlice
	switch m.Type() {
	case pmetric.MetricTypeSum:
		dps = m.Sum().DataPoints()
	case pmetric.MetricTypeGauge:
		dps = m.Gauge().DataPoints()
	default:
		return nil
	}
	out := make([]map[string]string, 0, dps.Len())
	for i := 0; i < dps.Len(); i++ {
		out = append(out, attrMap(dps.At(i).Attributes()))
	}
	return out
}

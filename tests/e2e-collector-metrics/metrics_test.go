// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package collectormetrics

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/open-telemetry/opentelemetry-operator/internal/testing/e2e"
)

// These tests validate that the target allocator labels targets correctly, end to
// end. A collector (TA + prometheus receiver) is deployed through the operator,
// scrapes a sample app, and exports OTLP into a single prometheus-operator-managed
// Prometheus running with translation_strategy: NoTranslation (so target labels cross
// the OTLP boundary byte-for-byte). We assert purely on target labels:
//
//   - ServiceMonitor (prometheusCR) path: a live differential. The SAME Prometheus
//     both receives the TA pipeline (pipeline=ta) and natively scrapes the same pod
//     via prometheus-operator (pipeline=oracle). The two must carry identical target
//     labels — the allocator must label a ServiceMonitor target exactly as
//     prometheus-operator does.
//   - raw scrape_configs path: the static target carries exactly job/instance and no
//     service-discovery labels.

const (
	sampleApp = "sample-app"          // the scraped workload (Deployment + Service)
	promSvc   = "prometheus-operated" // headless Service prometheus-operator creates for the Prometheus pods
	promPort  = 9090

	oracleName = "oracle"            // name of the oracle Prometheus CR and its ServiceAccount
	oracleSTS  = "prometheus-oracle" // StatefulSet prometheus-operator derives from the oracle Prometheus

	// The collector CR carries the same name in every test — each test runs in its
	// own namespace — and the operator derives the collector StatefulSet and the
	// target allocator ServiceAccount from it.
	collectorName = "otel"
	collectorSTS  = collectorName + "-collector"
	collectorSA   = collectorName + "-targetallocator"

	groupLabel    = "group"    // ServiceMonitor label that each side's selector matches on
	pipelineLabel = "pipeline" // label that distinguishes the pipelines in one Prometheus
	pipelineTA    = "ta"       // TA pipeline: collector(TA) -> OTLP -> Prometheus
	pipelineProm  = "oracle"   // oracle pipeline: prometheus-operator native scrape
)

// The manifests are static. Nothing here is templated: names can be fixed because
// every test runs in its own namespace, and the one object that genuinely differs
// between the two pipelines is built as a typed object instead (see serviceMonitor).
var (
	//go:embed testdata/sample-app.yaml
	sampleAppManifest string
	//go:embed testdata/oracle-prometheus.yaml
	oraclePrometheusManifest string
	//go:embed testdata/collector-sm.yaml
	smCollectorManifest string
	//go:embed testdata/collector-raw.yaml
	rawCollectorManifest string
)

// serviceMonitor builds the ServiceMonitor for one pipeline: it selects the sample
// app's Service and its named `metrics` port, labels itself so that pipeline's
// selector picks it up, and relabels every target with a constant `pipeline` label so
// the two pipelines can be told apart in one Prometheus.
func serviceMonitor(pipeline string) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "sm-" + pipeline,
			Labels: map[string]string{groupLabel: pipeline},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{MatchLabels: map[string]string{"app": sampleApp}},
			Endpoints: []monitoringv1.Endpoint{{
				Port:     "metrics",
				Interval: "5s",
				RelabelConfigs: []monitoringv1.RelabelConfig{{
					TargetLabel: pipelineLabel,
					Replacement: new(pipeline),
				}},
			}},
		},
	}
}

var testenv env.Environment

func TestMain(m *testing.M) {
	cfg, err := envconf.NewFromFlags()
	if err != nil {
		log.Fatalf("failed to parse e2e flags: %v", err)
	}
	testenv = env.NewWithConfig(cfg)
	os.Exit(testenv.Run(m))
}

// prom returns a handle to the test's oracle Prometheus. It is built per assertion
// rather than per package because the namespace is only known once setup ran.
func prom(t *testing.T, ctx context.Context, cfg *envconf.Config) *e2e.Prom {
	t.Helper()
	return e2e.NewProm(cfg, e2e.Namespace(t, ctx), promSvc, promPort)
}

// setup ensures prometheus-operator is installed, then deploys the sample app, the
// oracle Prometheus (+ RBAC), any extra objects (e.g. ServiceMonitors), the collector
// CR (+ RBAC), waits for readiness, and stashes the namespace in the context.
func setup(collectorManifest string, extra ...crclient.Object) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		e2e.EnsurePrometheusOperator(ctx, t, cfg)

		ctx = e2e.SetupNamespace(ctx, t, cfg)
		namespace := e2e.Namespace(t, ctx)

		e2e.Apply(ctx, t, cfg, namespace, sampleAppManifest)
		e2e.Apply(ctx, t, cfg, namespace, oraclePrometheusManifest)
		e2e.BindTargetAllocatorClusterRole(ctx, t, cfg, namespace, oracleName)
		e2e.ApplyObjects(ctx, t, cfg, namespace, extra...)
		e2e.Apply(ctx, t, cfg, namespace, collectorManifest)
		e2e.BindTargetAllocatorClusterRole(ctx, t, cfg, namespace, collectorSA)

		e2e.WaitForDeployment(ctx, t, cfg, namespace, sampleApp, 2*time.Minute)
		// prometheus-operator reconciles the Prometheus CR into the prometheus-oracle
		// StatefulSet; the operator reconciles the collector CR into <name>-collector.
		e2e.WaitForStatefulSet(ctx, t, cfg, namespace, oracleSTS, 1, 3*time.Minute)
		e2e.WaitForStatefulSet(ctx, t, cfg, namespace, collectorSTS, 1, 5*time.Minute)
		return ctx
	}
}

// TestServiceMonitorDifferential is the headline test: the target allocator must
// label a ServiceMonitor target exactly as prometheus-operator does. Both pipelines
// scrape the same pod and write to the same Prometheus, distinguished by a `pipeline`
// label, so any divergence in target labeling fails the differential.
func TestServiceMonitorDifferential(t *testing.T) {
	feat := features.New("ServiceMonitor targets labeled identically to prometheus-operator").
		Setup(setup(smCollectorManifest, serviceMonitor(pipelineTA), serviceMonitor(pipelineProm))).
		Assess("the allocator path really went through ServiceMonitor relabeling", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ns := e2e.Namespace(t, ctx)
			// Guards the differential against a vacuous pass: the TA series must carry
			// the prometheus-operator relabeling output, not be empty/trivial.
			prom(t, ctx, cfg).Eventually(ctx, t, fmt.Sprintf(`up{%s=%q}`, pipelineLabel, pipelineTA),
				e2e.HasSeries(e2e.Series{
					Labels:  map[string]string{"service": sampleApp, "namespace": ns},
					Present: []string{"endpoint", "pod", "container"},
				}))
			return ctx
		}).
		Assess("target identity matches prometheus-operator (up)", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			prom(t, ctx, cfg).SameLabelsAcross(ctx, t, e2e.Differential{Query: `up`, PartitionLabel: pipelineLabel, WantPartitions: 2})
			return ctx
		}).
		Assess("a scraped series matches prometheus-operator end-to-end (build_info)", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			prom(t, ctx, cfg).SameLabelsAcross(ctx, t, e2e.Differential{Query: `node_exporter_build_info`, PartitionLabel: pipelineLabel, WantPartitions: 2})
			return ctx
		}).
		Feature()

	testenv.Test(t, feat)
}

// TestRawScrapeConfigMetrics validates the target allocator's raw scrape_configs
// path: a static_configs target is allocated, scraped, exported OTLP and arrives
// carrying exactly the scrape config's identity (job/instance) and no
// service-discovery labels.
func TestRawScrapeConfigMetrics(t *testing.T) {
	feat := features.New("raw scrape_configs carry exactly the static identity").
		Setup(setup(rawCollectorManifest)).
		Assess("the static target carries exactly job/instance and no service-discovery labels", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			prom(t, ctx, cfg).Eventually(ctx, t, fmt.Sprintf(`up{job=%q}`, sampleApp),
				e2e.HasSeries(e2e.Series{
					Labels: map[string]string{"job": sampleApp, "instance": sampleApp + ":9100"},
					Exact:  true,
				}))
			return ctx
		}).
		Feature()

	testenv.Test(t, feat)
}

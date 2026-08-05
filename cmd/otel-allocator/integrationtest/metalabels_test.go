// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package integrationtest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"

	taconfig "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
)

const podJobName = "kube-pods"

// podScrapeConfig is a PodMonitor-shaped job: it keeps only running pods and
// copies Kubernetes meta labels onto real target labels. Those copies are what
// make a change to a meta label observable, both in the target's identity and in
// the scraped series.
const podScrapeConfig = `
scrape_configs:
  - job_name: kube-pods
    scrape_interval: 200ms
    scrape_timeout: 200ms
    relabel_configs:
      - {source_labels: [__meta_kubernetes_pod_phase], regex: Running, action: keep}
      - {source_labels: [__meta_kubernetes_namespace], target_label: namespace}
      - {source_labels: [__meta_kubernetes_pod_name], target_label: pod}
      - {source_labels: [__meta_kubernetes_pod_label_version], target_label: version}
`

// TestKubernetesMetaLabelUpdate covers the full path a Kubernetes meta label
// takes through the allocator, and what happens when one of those labels changes
// on a target that keeps its address:
//
//  1. service discovery produces a target carrying `__meta_kubernetes_*` labels,
//     and the allocator serves them verbatim on its HTTP SD endpoint;
//  2. the receiver relabels on those meta labels, so they reach the scraped
//     series (as `version`/`namespace`/`pod`) and the OTLP resource (as
//     `k8s.*`);
//  3. discovery reports the change, at the same address;
//  4. the change propagates: the allocator serves exactly one target, carrying
//     the new meta labels, and the scraped series carries the new relabeled
//     values;
//  5. the old target goes away — the receiver reports its series stale, and
//     nothing scrapes it again — rather than lingering beside the new one.
//
// Steps 4 and 5 are the regression-prone ones: the allocator hashes targets on
// their post-relabel labels precisely so that a meta-label change relabeling
// makes visible yields a new target identity, replacing the old one. The address
// is held constant on purpose — it is what an identity that ignored the
// relabeled labels would key on.
func TestKubernetesMetaLabelUpdate(t *testing.T) {
	const exposition = "# HELP pod_replicas Replicas reported by the pod.\n" +
		"# TYPE pod_replicas gauge\n" +
		"pod_replicas 3\n"
	const metricName = "pod_replicas"

	for _, tc := range []struct {
		name          string
		before, after discoveredPod
		// update is what service discovery emits to go from before to after.
		update func(addr string, before, after discoveredPod) []*targetgroup.Group
	}{
		{
			// A relabeled pod label changes. It surfaces in the series labels
			// only: the OTLP resource is built from the job, the address and the
			// k8s meta labels, none of which move.
			name:   "pod label changed",
			before: discoveredPod{name: "web-0", version: "v1"},
			after:  discoveredPod{name: "web-0", version: "v2"},
			// The same pod, hence the same discovery source: SD re-emits its
			// group in place, as it does when a pod is updated.
			update: func(addr string, _, after discoveredPod) []*targetgroup.Group {
				return []*targetgroup.Group{podTargetGroup(addr, after)}
			},
		},
		{
			// __meta_kubernetes_pod_name changes. That label is relabeled *and*
			// mapped to a resource attribute (k8s.pod.name), so the receiver
			// emits a different OTLP resource for the new target, not just
			// different series labels.
			name:   "pod replaced at the same address",
			before: discoveredPod{name: "web-0", version: "v1"},
			after:  discoveredPod{name: "web-1", version: "v1"},
			// A pod's name never changes, so this is a delete plus an add: web-0
			// goes away and web-1 comes up on its recycled address. SD reports
			// the departure as a group with no targets, and both land in one
			// update — so the allocator sees the address handed from one
			// discovery source to another.
			update: func(addr string, before, after discoveredPod) []*targetgroup.Group {
				return []*targetgroup.Group{removedPod(before), podTargetGroup(addr, after)}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Not parallel: each case runs a full target allocator, and the
			// allocator uses process-global state that is unsafe to drive from
			// several instances at once.
			mock := startMockTarget(t, exposition)

			cfg, err := promconfig.Load(podScrapeConfig, taconfig.NopLogger)
			require.NoError(t, err)
			require.Len(t, cfg.ScrapeConfigs, 1)
			// Drive discovery from the test instead of a cluster. Everything
			// downstream — the discovery manager, the allocator's
			// merge/relabel/hash, allocation, HTTP SD, the receiver — is the real
			// code path.
			sd := newFakeKubeSD()
			cfg.ScrapeConfigs[0].ServiceDiscoveryConfigs = discovery.Configs{sd}

			taEndpoint := startTargetAllocator(t, cfg.ScrapeConfigs)
			sink := new(consumertest.MetricsSink)
			startReceiver(t, taEndpoint, sink)

			before, after := seriesLabels(tc.before), seriesLabels(tc.after)

			// 1. Discovery yields the pod; the allocator serves its meta labels
			// untouched.
			sd.publish(t, podTargetGroup(mock.Host, tc.before))
			requireServedTarget(t, taEndpoint, servedMetaLabels(mock.Host, tc.before), podJobName)

			// 2. Relabeling on those meta labels reaches the scraped metric.
			waitForDataPoints(t, sink, metricName, before, 1)

			// 3. Discovery reports the change, at the same address.
			sd.publish(t, tc.update(mock.Host, tc.before, tc.after)...)

			// 4. The allocator serves the new target, and only it; the scrape
			// result changes with it.
			requireServedTarget(t, taEndpoint, servedMetaLabels(mock.Host, tc.after), podJobName)
			waitForDataPoints(t, sink, metricName, after, 3)

			// 5. The old target goes away rather than lingering beside the new
			// one. Its departure is reported directly: the receiver emits a
			// staleness marker for the old series when that target leaves the
			// scrape pool.
			waitForStaleSeries(t, sink, metricName, before)

			// And it is not scraped again afterwards. Counting is progress-driven
			// rather than clock-driven: by the time three more points of the new
			// series have landed, several scrape cycles have passed, so a
			// surviving old target would have produced points of its own.
			beforeCount := countDataPoints(sink, metricName, before)
			waitForDataPoints(t, sink, metricName, after, 6)
			assert.Equal(t, beforeCount, countDataPoints(sink, metricName, before),
				"the target with the old meta labels should be replaced, not scraped alongside the new one")
		})
	}
}

// discoveredPod is a pod as Kubernetes service discovery sees it, reduced to the
// fields these cases vary.
type discoveredPod struct {
	name    string
	version string // the pod's `version` label
}

// source is the discovery source the kubernetes_sd pod role derives for p. It
// keys the group inside the discovery manager, so re-emitting the same source
// replaces that pod's previous state — and a differently-named pod is a
// different source, because it is a different object.
func (p discoveredPod) source() string { return "pod/default/" + p.name }

// seriesLabels is what a scrape of p should produce: the relabel-produced series
// labels, plus the resource attributes the receiver derives from p's meta labels.
// The pod name feeds both, which is what makes it a wider-reaching change than
// the pod's `version` label.
func seriesLabels(p discoveredPod) map[string]string {
	return map[string]string{
		"service.name":       podJobName,
		"k8s.namespace.name": "default",
		"k8s.node.name":      "node-1",
		"k8s.container.name": "web",
		"k8s.pod.name":       p.name,
		"namespace":          "default",
		"pod":                p.name,
		"version":            p.version,
	}
}

// servedMetaLabels is the subset of p's Kubernetes meta labels the allocator must
// serve verbatim on its HTTP SD endpoint.
func servedMetaLabels(addr string, p discoveredPod) map[string]string {
	return map[string]string{
		"__address__":                         addr,
		"__meta_kubernetes_namespace":         "default",
		"__meta_kubernetes_pod_name":          p.name,
		"__meta_kubernetes_pod_label_version": p.version,
	}
}

// podTargetGroup builds the target group Prometheus's kubernetes_sd pod role
// produces for p, a single-container pod whose only pod label is `version`.
// Pod-level meta labels sit on the group and container-level ones on the target,
// mirroring discovery/kubernetes/pod.go's buildPod.
func podTargetGroup(addr string, p discoveredPod) *targetgroup.Group {
	return &targetgroup.Group{
		Source: p.source(),
		Labels: model.LabelSet{
			"__meta_kubernetes_namespace": "default",
			"__meta_kubernetes_pod_name":  model.LabelValue(p.name),
			// A replacement pod is a different Kubernetes object, so it must not
			// reuse its predecessor's UID; deriving it from the name keeps that
			// true without the cases having to carry one.
			"__meta_kubernetes_pod_uid": model.LabelValue("pod-uid-" + p.name),
			// The pod IP is deliberately shared: these cases are about a target
			// whose address stays put.
			"__meta_kubernetes_pod_ip":                   "10.1.0.1",
			"__meta_kubernetes_pod_node_name":            "node-1",
			"__meta_kubernetes_pod_phase":                "Running",
			"__meta_kubernetes_pod_ready":                "true",
			"__meta_kubernetes_pod_label_version":        model.LabelValue(p.version),
			"__meta_kubernetes_pod_labelpresent_version": "true",
		},
		Targets: []model.LabelSet{{
			model.AddressLabel:                            model.LabelValue(addr),
			"__meta_kubernetes_pod_container_name":        "web",
			"__meta_kubernetes_pod_container_port_name":   "metrics",
			"__meta_kubernetes_pod_container_port_number": "8080",
		}},
	}
}

// removedPod is what Kubernetes SD emits when a pod goes away: the pod's source
// carrying no targets, which drops the group.
func removedPod(p discoveredPod) *targetgroup.Group {
	return &targetgroup.Group{Source: p.source()}
}

// fakeKubeSD stands in for kubernetes_sd_config: a discovery.Config/Discoverer
// pair the test drives directly, so it can emit a pod's target group and later
// re-emit it with one meta label changed, at a moment of the test's choosing.
// Everything downstream is the real thing — Prometheus's discovery.Manager runs
// it like any other SD, including the replace-group-by-source semantics that
// make a pod update an update rather than a second target.
//
// The real Kubernetes SD is upstream Prometheus code the allocator uses
// unchanged, and running it would need a cluster; what this suite tests is what
// the allocator does with the labels it produces. The group this fake emits is
// shaped exactly like the pod role's (see podTargetGroup).
type fakeKubeSD struct {
	// Role mirrors kubernetes_sd_config's `role`. Nothing reads it: it exists so
	// the config marshals to a non-empty YAML section, which Prometheus's config
	// reader rejects otherwise — the allocator serves this scrape config on
	// /scrape_configs and the receiver parses it back.
	Role string `yaml:"role"`

	updates chan []*targetgroup.Group
}

// Prometheus resolves SD configs through a process-global registry, both to
// marshal them (the allocator's /scrape_configs response) and to parse them back
// (the receiver). Registering in init keeps it ahead of the first config parse,
// after which the registry's derived struct types are cached.
func init() {
	discovery.RegisterConfig(&fakeKubeSD{})
}

func newFakeKubeSD() *fakeKubeSD {
	return &fakeKubeSD{Role: "pod", updates: make(chan []*targetgroup.Group)}
}

// Name must be a valid Go identifier fragment: Prometheus's registry builds a
// struct field named AUTO_DISCOVERY_<name>_sd_configs from it.
func (*fakeKubeSD) Name() string { return "fake_kubernetes" }

func (sd *fakeKubeSD) NewDiscoverer(discovery.DiscovererOptions) (discovery.Discoverer, error) {
	return sd, nil
}

func (*fakeKubeSD) NewDiscovererMetrics(prometheus.Registerer, discovery.RefreshMetricsInstantiator) discovery.DiscovererMetrics {
	return &discovery.NoopDiscovererMetrics{}
}

func (sd *fakeKubeSD) Run(ctx context.Context, up chan<- []*targetgroup.Group) {
	for {
		select {
		case <-ctx.Done():
			return
		case groups := <-sd.updates:
			select {
			case up <- groups:
			case <-ctx.Done():
				return
			}
		}
	}
}

// publish hands a discovery update to the manager, blocking until the running
// discoverer picks it up — so a test that publishes has, by return, handed the
// update to the real discovery pipeline.
func (sd *fakeKubeSD) publish(t *testing.T, groups ...*targetgroup.Group) {
	t.Helper()
	select {
	case sd.updates <- groups:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out publishing a service discovery update; is the discovery manager running?")
	}
}

// servedTarget is one entry of the allocator's HTTP SD response.
type servedTarget struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

// requireServedTarget waits until the allocator serves exactly one target for
// the job and that target carries all of want. Requiring exactly one is the
// point: a pod update must replace the previous target rather than add to it.
// want is matched as a subset, so a caller need only name the labels it cares
// about.
func requireServedTarget(t *testing.T, taEndpoint *url.URL, want map[string]string, job string) {
	t.Helper()
	var last []servedTarget
	require.Eventuallyf(t, func() bool {
		last = fetchServedTargets(t, taEndpoint, job)
		return len(last) == 1 && subset(want, last[0].Labels)
	}, 30*time.Second, 100*time.Millisecond,
		"target allocator never served exactly one target carrying %v; last response: %+v", want, &last)
}

func fetchServedTargets(t *testing.T, taEndpoint *url.URL, job string) []servedTarget {
	t.Helper()
	u := fmt.Sprintf("%s/jobs/%s/targets?collector_id=%s", taEndpoint, url.PathEscape(job), url.QueryEscape(collectorID))
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, u, http.NoBody)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var targets []servedTarget
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil
	}
	return targets
}

// waitForDataPoints blocks until the sink holds at least n data points of the
// named metric carrying all of want. On failure it reports the distinct label
// sets the metric was actually seen with, which is what usually explains the
// miss.
func waitForDataPoints(t *testing.T, sink *consumertest.MetricsSink, name string, want map[string]string, n int) {
	t.Helper()
	var seen []map[string]string
	require.Eventuallyf(t, func() bool {
		observed := collectMetrics(sink)[name]
		seen = distinct(observed)
		return countMatching(observed, want) >= n
	}, 30*time.Second, 100*time.Millisecond,
		"expected at least %d data points of %q carrying %v; %q was seen with label sets %+v", n, name, want, name, &seen)
}

// countDataPoints returns how many data points of the named metric carry all of
// want. Staleness markers are excluded (see collectMetrics): a target that has
// gone away gets a final marker per series, which must not read as another
// scrape of that target.
func countDataPoints(sink *consumertest.MetricsSink, name string, want map[string]string) int {
	return countMatching(collectMetrics(sink)[name], want)
}

// waitForStaleSeries blocks until the sink holds a staleness marker for the named
// metric carrying all of want. That marker is the receiver's report that the
// target producing the series has left its scrape pool — the direct signal that
// the target is gone, as opposed to merely having stopped producing points.
func waitForStaleSeries(t *testing.T, sink *consumertest.MetricsSink, name string, want map[string]string) {
	t.Helper()
	var seen []map[string]string
	require.Eventuallyf(t, func() bool {
		stale := staleSeries(sink, name)
		seen = distinct(stale)
		return countMatching(stale, want) > 0
	}, 30*time.Second, 100*time.Millisecond,
		"expected a staleness marker for %q carrying %v; %q went stale with label sets %+v", name, want, name, &seen)
}

// staleSeries returns the label sets of the named metric's staleness markers.
func staleSeries(sink *consumertest.MetricsSink, name string) []map[string]string {
	var out []map[string]string
	for _, p := range collectDataPoints(sink)[name] {
		if p.stale {
			out = append(out, p.labels)
		}
	}
	return out
}

func countMatching(observed []map[string]string, want map[string]string) int {
	count := 0
	for _, labels := range observed {
		if subset(want, labels) {
			count++
		}
	}
	return count
}

// distinct deduplicates label sets. Every scrape of a target repeats its label
// set, so a raw dump is hundreds of copies of a handful of values.
func distinct(labelSets []map[string]string) []map[string]string {
	var out []map[string]string
	seen := map[string]struct{}{}
	for _, ls := range labelSets {
		// fmt prints maps in sorted key order, so this is a stable identity.
		key := fmt.Sprint(ls)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ls)
	}
	return out
}

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package integrationtest

import (
	"testing"

	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"

	taconfig "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
)

const daemonSetJobName = "node-exporter"

// daemonSetScrapeConfig is a node-exporter-shaped job for a hostNetwork
// DaemonSet: it labels targets by node, not by pod, because with hostNetwork the
// pod backing a node's endpoint is an implementation detail. Nothing carries the
// pod's identity into the target labels — which is precisely what makes a pod
// replacement invisible to the allocator's target hash.
const daemonSetScrapeConfig = `
scrape_configs:
  - job_name: node-exporter
    scrape_interval: 200ms
    scrape_timeout: 200ms
    relabel_configs:
      - {source_labels: [__meta_kubernetes_pod_phase], regex: Running, action: keep}
      - {source_labels: [__meta_kubernetes_namespace], target_label: namespace}
      - {source_labels: [__meta_kubernetes_pod_node_name], target_label: node}
`

// TestHostNetworkDaemonSetPodReplaced is a regression test for a stale target.
//
// A DaemonSet pod with hostNetwork keeps its address across a replacement — the
// address is the host's, not the pod's — and a job that labels by node keeps its
// post-relabel labels too. The only things that change are Kubernetes meta
// labels: the pod's generated name and its UID.
//
// This test verifies that target allocator's target hashing and relabelling
// implementation handles this scenario correctly. It's a regression test for
// https://github.com/open-telemetry/opentelemetry-operator/issues/4839.
//
// TestKubernetesMetaLabelUpdate/pod_replaced_at_the_same_address is the control:
// the same replacement, at the same address, but with the pod name relabeled onto
// a target label. That moves the hash, so it exercises the ordinary add/remove
// path — what this test isolates is the same-hash path.
//
// allocator.refreshSurvivingTargets is what makes this pass.
func TestHostNetworkDaemonSetPodReplaced(t *testing.T) {
	const exposition = "# HELP node_cpu_seconds_total Seconds the CPUs spent in each mode.\n" +
		"# TYPE node_cpu_seconds_total counter\n" +
		"node_cpu_seconds_total 12345\n"
	const metricName = "node_cpu_seconds_total"

	old := daemonSetPod{name: "node-exporter-aaaaa"}
	replacement := daemonSetPod{name: "node-exporter-bbbbb"}

	mock := startMockTarget(t, exposition)

	cfg, err := promconfig.Load(daemonSetScrapeConfig, taconfig.NopLogger)
	require.NoError(t, err)
	require.Len(t, cfg.ScrapeConfigs, 1)
	sd := newFakeKubeSD()
	cfg.ScrapeConfigs[0].ServiceDiscoveryConfigs = discovery.Configs{sd}

	taEndpoint := startTargetAllocator(t, cfg.ScrapeConfigs)
	sink := new(consumertest.MetricsSink)
	startReceiver(t, taEndpoint, sink)

	// The DaemonSet pod is discovered and scraped.
	sd.publish(t, daemonSetTargetGroup(mock.Host, old))
	requireServedTarget(t, taEndpoint, daemonSetMetaLabels(mock.Host, old), daemonSetJobName)
	waitForDataPoints(t, sink, metricName, daemonSetSeriesLabels(old), 1)

	// The pod is replaced: the old one goes away and its successor comes up on
	// the same host address. Only the pod's name and UID differ.
	sd.publish(t,
		&targetgroup.Group{Source: daemonSetSource(old)},
		daemonSetTargetGroup(mock.Host, replacement),
	)

	// The allocator must serve the replacement's meta labels, not its
	// predecessor's.
	requireServedTarget(t, taEndpoint, daemonSetMetaLabels(mock.Host, replacement), daemonSetJobName)

	// And the metrics scraped from it must be attributed to the pod that is
	// actually running.
	waitForDataPoints(t, sink, metricName, daemonSetSeriesLabels(replacement), 1)
}

// daemonSetPod is a hostNetwork DaemonSet pod as Kubernetes service discovery
// sees it. Only the generated name varies across a replacement; everything else
// belongs to the node, which does not change.
type daemonSetPod struct {
	name string
}

func daemonSetSource(p daemonSetPod) string { return "pod/monitoring/" + p.name }

// daemonSetTargetGroup builds the target group the kubernetes_sd pod role
// produces for p. The address is the host's, as hostNetwork makes the pod IP the
// node IP, so it survives the pod being replaced.
func daemonSetTargetGroup(hostAddr string, p daemonSetPod) *targetgroup.Group {
	return &targetgroup.Group{
		Source: daemonSetSource(p),
		Labels: model.LabelSet{
			"__meta_kubernetes_namespace": "monitoring",
			// The pod's identity — the only thing that changes on a replacement.
			"__meta_kubernetes_pod_name": model.LabelValue(p.name),
			"__meta_kubernetes_pod_uid":  model.LabelValue("pod-uid-" + p.name),
			// Everything below describes the node or the DaemonSet, so a
			// replacement pod reports it identically.
			"__meta_kubernetes_pod_node_name":        "node-1",
			"__meta_kubernetes_pod_host_ip":          "10.0.0.1",
			"__meta_kubernetes_pod_ip":               "10.0.0.1",
			"__meta_kubernetes_pod_phase":            "Running",
			"__meta_kubernetes_pod_ready":            "true",
			"__meta_kubernetes_pod_controller_kind":  "DaemonSet",
			"__meta_kubernetes_pod_controller_name":  "node-exporter",
			"__meta_kubernetes_pod_label_app":        "node-exporter",
			"__meta_kubernetes_pod_labelpresent_app": "true",
		},
		Targets: []model.LabelSet{{
			model.AddressLabel:                            model.LabelValue(hostAddr),
			"__meta_kubernetes_pod_container_name":        "node-exporter",
			"__meta_kubernetes_pod_container_port_name":   "metrics",
			"__meta_kubernetes_pod_container_port_number": "9100",
		}},
	}
}

// daemonSetMetaLabels is the subset of p's meta labels the allocator must serve
// verbatim: the pod's identity, which is what goes stale.
func daemonSetMetaLabels(addr string, p daemonSetPod) map[string]string {
	return map[string]string{
		"__address__":                addr,
		"__meta_kubernetes_pod_name": p.name,
		"__meta_kubernetes_pod_uid":  "pod-uid-" + p.name,
	}
}

// daemonSetSeriesLabels is what a scrape of p should produce. The relabeled
// series labels are identical for both pods by construction; only the resource
// attributes the receiver derives from the meta labels tell them apart.
func daemonSetSeriesLabels(p daemonSetPod) map[string]string {
	return map[string]string{
		"service.name":       daemonSetJobName,
		"namespace":          "monitoring",
		"node":               "node-1",
		"k8s.namespace.name": "monitoring",
		"k8s.node.name":      "node-1",
		"k8s.container.name": "node-exporter",
		"k8s.pod.name":       p.name,
		"k8s.pod.uid":        "pod-uid-" + p.name,
	}
}

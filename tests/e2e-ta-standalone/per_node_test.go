// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build e2e

package e2e_ta_standalone

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// TestPerNodeTargetAllocator validates the per-node allocation strategy.
//
// The per-node strategy assigns each scrape target to the collector co-located
// on the same Kubernetes node (matched via __meta_kubernetes_pod_node_name).
// This test requires a multi-node kind cluster (kind-multinode.yaml at repo root
// with control-plane + 2 workers). Static scrape configs cannot be used because
// they don't populate __meta_kubernetes_pod_node_name; Kubernetes SD is required.
//
// Topology:
//
//	worker-1: collector-0, scrape-target-0
//	worker-2: collector-1, scrape-target-1
//
// Assertion: each target pod's node matches its assigned collector pod's node.
func TestPerNodeTargetAllocator(t *testing.T) {
	env := newTestEnv(t)
	ctx, ns := env.ctx, env.ns

	// Verify multi-node cluster (need at least 2 worker nodes).
	workers := getWorkerNodes(t, ctx)
	if len(workers) < 2 {
		t.Skipf("per-node test requires ≥2 worker nodes, got %d (run with kind-multinode.yaml)", len(workers))
	}
	worker1 := workers[0]
	worker2 := workers[1]

	taConfig := newTAConfig("per-node").withKubernetesSD(ns).build()
	deployTA(t, ctx, ns, taConfig)
	waitForDeploymentReady(t, ctx, ns, "target-allocator", 1)

	// Deploy collectors with topologySpreadConstraints so they land on different nodes.
	deployPerNodeCollectors(t, ctx, ns)
	waitForStatefulSetReady(t, ctx, ns, "collector", 2)

	// Deploy one scrape-target pod pinned to each worker.
	deployScrapeTarget(t, ctx, ns, "scrape-target-0", worker1)
	deployScrapeTarget(t, ctx, ns, "scrape-target-1", worker2)
	waitForPodsReady(t, ctx, ns, "app=scrape-target", 2)

	// Wait for Kubernetes SD discovery + TA target allocation (~40s).
	time.Sleep(45 * time.Second)

	t.Run("each target assigned to collector on same node", func(t *testing.T) {
		proxyBase := taProxyBase(ns)

		// Determine which node each collector landed on.
		collectorNodeMap := collectorsToNodes(t, ctx, ns, 2)
		t.Logf("collector node map: %v", collectorNodeMap)

		// Determine which node each scrape target landed on.
		targetNodeMap := scrapeTargetsToNodes(t, ctx, ns)
		t.Logf("scrape target node map: %v", targetNodeMap)

		// For each collector, fetch its assigned targets and verify co-location.
		allCount := 0
		for i := 0; i < 2; i++ {
			collectorID := fmt.Sprintf("collector-%d", i)
			collectorNode := collectorNodeMap[collectorID]
			require.NotEmpty(t, collectorNode, "could not determine node for %s", collectorID)

			addresses := getCollectorTargets(t, ctx, proxyBase, "per-node-targets", collectorID)
			allCount += len(addresses)

			for _, addr := range addresses {
				podName, podNode := findPodByIP(t, ctx, ns, addr)
				require.NotEmpty(t, podNode, "could not find pod node for IP %s (pod %s)", addr, podName)
				assert.Equal(t, collectorNode, podNode,
					"target %s (pod %s, node %s) should be on same node as %s (node %s)",
					addr, podName, podNode, collectorID, collectorNode)
			}
			t.Logf("%s (node %s): assigned targets %v", collectorID, collectorNode, addresses)
		}

		// Both scrape targets should be assigned (no unallocated targets).
		assert.Equal(t, 2, allCount, "both scrape targets should be assigned across collectors")

		// Cross-check with targetNodeMap: each target pod should appear on the
		// collector that matches its node.
		for targetPod, targetNode := range targetNodeMap {
			assignedCollector := findCollectorForNode(collectorNodeMap, targetNode)
			if assignedCollector == "" {
				t.Logf("no collector on node %s (pod %s) - skipping co-location check", targetNode, targetPod)
				continue
			}
			collectorTargets := getCollectorTargets(t, ctx, proxyBase, "per-node-targets", assignedCollector)
			podIP := getPodIP(t, ctx, ns, targetPod)
			assert.True(t, targetAddressMatchesIP(collectorTargets, podIP),
				"target pod %s (IP %s, node %s) should be assigned to %s (node %s); got targets %v",
				targetPod, podIP, targetNode, assignedCollector, targetNode, collectorTargets)
		}
	})
}

// ---------------------------------------------------------------------------
// Per-node deployment helpers
// ---------------------------------------------------------------------------

// deployPerNodeCollectors creates a StatefulSet with topologySpreadConstraints
// forcing the two collector pods onto different worker nodes.
func deployPerNodeCollectors(t *testing.T, ctx context.Context, ns string) {
	t.Helper()
	maxSkew := int32(1)
	deployCollectorsWithOpts(t, ctx, ns, 2, &collectorOpts{
		affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{{
						MatchExpressions: []corev1.NodeSelectorRequirement{{
							Key:      "node-role.kubernetes.io/control-plane",
							Operator: corev1.NodeSelectorOpDoesNotExist,
						}},
					}},
				},
			},
		},
		topologySpreadConstraints: []corev1.TopologySpreadConstraint{{
			MaxSkew:           maxSkew,
			TopologyKey:       "kubernetes.io/hostname",
			WhenUnsatisfiable: corev1.DoNotSchedule,
			LabelSelector:     &metav1.LabelSelector{MatchLabels: collectorLabel},
		}},
	})
}

// deployScrapeTarget deploys a pod annotated for Prometheus scraping, pinned to nodeName.
func deployScrapeTarget(t *testing.T, ctx context.Context, ns, name, nodeName string) {
	t.Helper()
	t.Logf("deploying scrape target %s on node %s", name, nodeName)
	_, err := clientset.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    map[string]string{"app": "scrape-target"},
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8080",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
			Containers: []corev1.Container{{
				Name:  "target",
				Image: "ghcr.io/open-telemetry/opentelemetry-operator/e2e-test-app-metrics-basic-auth:main",
				Ports: []corev1.ContainerPort{{ContainerPort: 9123, Name: "metrics"}},
			}},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
}

func waitForPodsReady(t *testing.T, ctx context.Context, ns, labelSelector string, count int) {
	t.Helper()
	t.Logf("waiting for %d pods ready in %s with selector %s", count, ns, labelSelector)
	err := wait.PollUntilContextTimeout(ctx, pollInterval, testTimeout, true, func(ctx context.Context) (bool, error) {
		pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			return false, nil
		}
		ready := 0
		for _, p := range pods.Items {
			if p.Status.Phase == corev1.PodRunning {
				ready++
			}
		}
		return ready >= count, nil
	})
	require.NoError(t, err, "pods %s did not become ready", labelSelector)
}

// ---------------------------------------------------------------------------
// Node-mapping helpers
// ---------------------------------------------------------------------------

func getWorkerNodes(t *testing.T, ctx context.Context) []string {
	t.Helper()
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	var workers []string
	for _, n := range nodes.Items {
		if _, isCP := n.Labels["node-role.kubernetes.io/control-plane"]; !isCP {
			workers = append(workers, n.Name)
		}
	}
	return workers
}

func collectorsToNodes(t *testing.T, ctx context.Context, ns string, count int) map[string]string {
	t.Helper()
	result := make(map[string]string, count)
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("collector-%d", i)
		pod, err := clientset.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
		require.NoError(t, err)
		result[name] = pod.Spec.NodeName
	}
	return result
}

func scrapeTargetsToNodes(t *testing.T, ctx context.Context, ns string) map[string]string {
	t.Helper()
	pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: "app=scrape-target"})
	require.NoError(t, err)
	result := make(map[string]string, len(pods.Items))
	for _, p := range pods.Items {
		result[p.Name] = p.Spec.NodeName
	}
	return result
}

func findPodByIP(t *testing.T, ctx context.Context, ns, addr string) (podName, nodeName string) {
	t.Helper()
	// addr may be "ip:port" (from __address__ label); extract just the IP.
	ip := addr
	if idx := strings.LastIndex(addr, ":"); idx > 0 {
		ip = addr[:idx]
	}
	pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", ""
	}
	for _, p := range pods.Items {
		if p.Status.PodIP == ip {
			return p.Name, p.Spec.NodeName
		}
	}
	return "", ""
}

func findCollectorForNode(collectorNodeMap map[string]string, node string) string {
	for collector, n := range collectorNodeMap {
		if n == node {
			return collector
		}
	}
	return ""
}

func getPodIP(t *testing.T, ctx context.Context, ns, podName string) string {
	t.Helper()
	pod, err := clientset.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return ""
	}
	return pod.Status.PodIP
}

// targetAddressMatchesIP returns true if any address in the slice starts with
// the given IP. Addresses from the TA API are "ip:port"; pod IPs are bare "ip".
func targetAddressMatchesIP(addresses []string, ip string) bool {
	for _, addr := range addresses {
		if addr == ip || strings.HasPrefix(addr, ip+":") {
			return true
		}
	}
	return false
}

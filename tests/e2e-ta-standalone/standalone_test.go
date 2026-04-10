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
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset        *kubernetes.Clientset
	taImg            string
	collectorImg     string
	kustomizeDir     string
	collectorLabel   = map[string]string{"app": "otel-collector"}
	testTargets      = []string{"target-a:8080", "target-b:8080", "target-c:8080", "target-d:8080", "target-e:8080", "target-f:8080"}
	testTimeout      = 5 * time.Minute
	pollInterval     = 2 * time.Second
	discoveryTimeout = 90 * time.Second
)

func TestMain(m *testing.M) {
	taImg = os.Getenv("TARGETALLOCATOR_IMG")
	if taImg == "" {
		log.Fatal("TARGETALLOCATOR_IMG environment variable must be set")
	}
	collectorImg = os.Getenv("COLLECTOR_IMG")
	if collectorImg == "" {
		log.Fatal("COLLECTOR_IMG environment variable must be set")
	}

	// Resolve the kustomize base directory relative to this test file.
	kustomizeDir = filepath.Join("..", "..", "cmd", "otel-allocator", "deploy")
	if _, err := os.Stat(filepath.Join(kustomizeDir, "kustomization.yaml")); err != nil {
		log.Fatalf("kustomize base not found at %s: %v", kustomizeDir, err)
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("failed to build kubeconfig: %v", err)
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create kubernetes client: %v", err)
	}

	os.Exit(m.Run())
}

// TestStandaloneTargetAllocator validates the standalone TA deployment using
// the official kustomize manifests. It deploys the TA + collectors into a fresh
// namespace, verifies target distribution, exercises scale-up/down, and checks
// the HTTP API contract.
func TestStandaloneTargetAllocator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ns := createTestNamespace(t, ctx)
	defer cleanupNamespace(t, ns)

	// 1. Deploy TA from the official kustomize base.
	deployTAFromKustomize(t, ctx, ns)
	waitForDeploymentReady(t, ctx, ns, "target-allocator", 1)

	// 2. Deploy collectors (not part of TA kustomize — users bring their own).
	deployCollectors(t, ctx, ns, 2)
	waitForStatefulSetReady(t, ctx, ns, "collector", 2)

	// 3. Wait for TA to discover collectors and assign targets.
	initialAssignment := waitForTargetDistribution(t, ctx, ns, 2)

	t.Run("targets distributed across collectors", func(t *testing.T) {
		allTargets := allAssignedTargets(initialAssignment)
		assert.Len(t, allTargets, len(testTargets), "all targets should be assigned")
		// With consistent hashing, at least one collector must have targets.
		hasTargets := false
		for _, targets := range initialAssignment {
			if len(targets) > 0 {
				hasTargets = true
				break
			}
		}
		assert.True(t, hasTargets, "at least one collector should have targets")
	})

	t.Run("scale up preserves consistency", func(t *testing.T) {
		scaleStatefulSet(t, ctx, ns, "collector", 3)
		waitForStatefulSetReady(t, ctx, ns, "collector", 3)

		afterScaleUp := waitForTargetDistribution(t, ctx, ns, 3)
		allTargets := allAssignedTargets(afterScaleUp)
		assert.Len(t, allTargets, len(testTargets), "all %d targets should remain assigned after scale-up", len(testTargets))

		// Consistent hashing: majority of targets should stay on original collector.
		stayed := countStayedTargets(initialAssignment, afterScaleUp)
		assert.GreaterOrEqual(t, stayed, 3, "at least 3/6 targets should stay on original collector (consistent hashing)")
	})

	t.Run("scale down reassigns targets", func(t *testing.T) {
		scaleStatefulSet(t, ctx, ns, "collector", 2)
		waitForStatefulSetReady(t, ctx, ns, "collector", 2)

		afterScaleDown := waitForTargetDistribution(t, ctx, ns, 2)
		allTargets := allAssignedTargets(afterScaleDown)
		assert.Len(t, allTargets, len(testTargets), "all %d targets should be assigned after scale-down", len(testTargets))

		// collector-2 should have no targets.
		assert.Empty(t, afterScaleDown["collector-2"], "collector-2 should have no targets after scale-down")
	})

	t.Run("HTTP API contract", func(t *testing.T) {
		proxyBase := fmt.Sprintf("/api/v1/namespaces/%s/services/target-allocator:80/proxy", ns)

		// /jobs
		body := kubectlGetRaw(t, ctx, proxyBase+"/jobs")
		assert.Contains(t, string(body), "test-targets", "/jobs should contain test-targets")

		// /scrape_configs
		body = kubectlGetRaw(t, ctx, proxyBase+"/scrape_configs")
		assert.Contains(t, string(body), "test-targets", "/scrape_configs should contain test-targets")

		// /livez — a successful GET (no error) means healthy.
		kubectlGetRaw(t, ctx, proxyBase+"/livez")

		// /readyz
		kubectlGetRaw(t, ctx, proxyBase+"/readyz")

		// Unknown collector_id should return no targets.
		body, err := clientset.CoreV1().RESTClient().Get().
			AbsPath(proxyBase + "/jobs/test-targets/targets").
			Param("collector_id", "nonexistent").
			DoRaw(ctx)
		require.NoError(t, err, "GET targets for unknown collector failed")
		assert.NotContains(t, string(body), "target-a:8080", "unknown collector should have no targets")
	})
}

// ---------------------------------------------------------------------------
// Deployment helpers
// ---------------------------------------------------------------------------

func createTestNamespace(t *testing.T, ctx context.Context) string {
	t.Helper()
	name := fmt.Sprintf("ta-test-%s", randomSuffix())
	_, err := clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}, metav1.CreateOptions{})
	require.NoError(t, err, "create namespace")
	t.Logf("created namespace %s", name)
	return name
}

func cleanupNamespace(t *testing.T, ns string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Clean up cluster-scoped resources named after the namespace.
	_ = clientset.RbacV1().ClusterRoleBindings().Delete(ctx, ns, metav1.DeleteOptions{})
	_ = clientset.RbacV1().ClusterRoles().Delete(ctx, ns, metav1.DeleteOptions{})
	_ = clientset.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
	t.Logf("cleaned up namespace %s and cluster RBAC", ns)
}

// deployTAFromKustomize builds the kustomize base and applies it with namespace
// and image overrides, verifying that the official kustomize manifests render and
// produce working resources.
func deployTAFromKustomize(t *testing.T, ctx context.Context, ns string) {
	t.Helper()

	// Create a temporary overlay as a sibling of the kustomize base directory.
	// It cannot be inside the base (kustomize detects a cycle).
	absKustomizeDir, err := filepath.Abs(kustomizeDir)
	require.NoError(t, err)
	parentDir := filepath.Dir(absKustomizeDir)
	overlayDir, err := os.MkdirTemp(parentDir, ".test-overlay-*")
	require.NoError(t, err)
	defer os.RemoveAll(overlayDir)

	relBase, err := filepath.Rel(overlayDir, absKustomizeDir)
	require.NoError(t, err)

	imgName, imgTag := splitImageNameTag(taImg)
	overlayContent := fmt.Sprintf(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: %[1]s

resources:
  - %[4]s

images:
  - name: target-allocator
    newName: %[2]s
    newTag: "%[3]s"

patches:
  # Rename cluster-scoped resources to namespace-unique names.
  - target:
      kind: ClusterRole
      name: target-allocator
    patch: |
      - op: replace
        path: /metadata/name
        value: %[1]s
  - target:
      kind: ClusterRoleBinding
      name: target-allocator
    patch: |
      - op: replace
        path: /metadata/name
        value: %[1]s
      - op: replace
        path: /roleRef/name
        value: %[1]s
      - op: replace
        path: /subjects/0/namespace
        value: %[1]s
`, ns, imgName, imgTag, relBase)

	err = os.WriteFile(filepath.Join(overlayDir, "kustomization.yaml"), []byte(overlayContent), 0o600)
	require.NoError(t, err)

	// Build with kustomize to verify the manifests render correctly.
	out, err := exec.CommandContext(ctx, "kustomize", "build", overlayDir).CombinedOutput()
	require.NoError(t, err, "kustomize build failed: %s", string(out))
	t.Logf("kustomize build succeeded (%d bytes)", len(out))

	// Apply rendered manifests to the cluster.
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(string(out))
	applyOut, err := cmd.CombinedOutput()
	require.NoError(t, err, "kubectl apply failed: %s", string(applyOut))
	t.Logf("applied kustomize manifests:\n%s", string(applyOut))

	// Create the ConfigMap with test scrape config. The ConfigMap is not part
	// of the kustomize base because it is deployment-specific configuration —
	// users provide their own.
	taConfig := fmt.Sprintf(`allocation_strategy: consistent-hashing
filter_strategy: relabel-config
collector_selector:
  matchLabels:
    app: otel-collector
config:
  scrape_configs:
    - job_name: test-targets
      scrape_interval: 30s
      static_configs:
        - targets: [%s]
`, quotedTargets())

	_, err = clientset.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "target-allocator"},
		Data:       map[string]string{"targetallocator.yaml": taConfig},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
}

func deployCollectors(t *testing.T, ctx context.Context, ns string, replicas int32) {
	t.Helper()

	collectorConfig := `receivers:
  prometheus:
    config: {}
    target_allocator:
      collector_id: ${POD_NAME}
      endpoint: http://target-allocator:80
      interval: 30s
exporters:
  debug: {}
service:
  pipelines:
    metrics:
      receivers: [prometheus]
      exporters: [debug]
  telemetry:
    metrics:
      readers:
        - pull:
            exporter:
              prometheus:
                host: 0.0.0.0
                port: 8888
`
	_, err := clientset.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "collector-config"},
		Data:       map[string]string{"collector.yaml": collectorConfig},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = clientset.AppsV1().StatefulSets(ns).Create(ctx, &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "collector"},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "collector",
			Selector:    &metav1.LabelSelector{MatchLabels: collectorLabel},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: collectorLabel},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:         "collector",
						Image:        collectorImg,
						Args:         []string{"--config=/conf/collector.yaml"},
						Env:          []corev1.EnvVar{{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}}},
						Ports:        []corev1.ContainerPort{{ContainerPort: 8888, Name: "metrics"}},
						VolumeMounts: []corev1.VolumeMount{{Name: "config", MountPath: "/conf"}},
					}},
					Volumes: []corev1.Volume{{
						Name:         "config",
						VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "collector-config"}}},
					}},
				},
			},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Wait helpers
// ---------------------------------------------------------------------------

func waitForDeploymentReady(t *testing.T, ctx context.Context, ns, name string, replicas int32) {
	t.Helper()
	t.Logf("waiting for deployment %s/%s to have %d ready replicas", ns, name, replicas)
	err := wait.PollUntilContextTimeout(ctx, pollInterval, testTimeout, true, func(ctx context.Context) (bool, error) {
		d, err := clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		return d.Status.ReadyReplicas >= replicas, nil
	})
	require.NoError(t, err, "deployment %s did not become ready", name)
}

func waitForStatefulSetReady(t *testing.T, ctx context.Context, ns, name string, replicas int32) {
	t.Helper()
	t.Logf("waiting for statefulset %s/%s to have %d ready replicas", ns, name, replicas)
	err := wait.PollUntilContextTimeout(ctx, pollInterval, testTimeout, true, func(ctx context.Context) (bool, error) {
		ss, err := clientset.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		return ss.Status.ReadyReplicas >= replicas, nil
	})
	require.NoError(t, err, "statefulset %s did not become ready", name)
}

// waitForTargetDistribution polls the TA until all targets are assigned across
// the expected number of collectors.
func waitForTargetDistribution(t *testing.T, ctx context.Context, ns string, expectedCollectors int) map[string][]string {
	t.Helper()
	proxyBase := fmt.Sprintf("/api/v1/namespaces/%s/services/target-allocator:80/proxy", ns)

	var assignment map[string][]string
	err := wait.PollUntilContextTimeout(ctx, pollInterval, discoveryTimeout, true, func(ctx context.Context) (bool, error) {
		assignment = getTargetAssignment(t, ctx, proxyBase, expectedCollectors)
		if assignment == nil {
			return false, nil
		}
		total := len(allAssignedTargets(assignment))
		if total != len(testTargets) {
			t.Logf("waiting for target discovery: %d/%d targets assigned", total, len(testTargets))
			return false, nil
		}
		return true, nil
	})
	require.NoError(t, err, "targets were not distributed to %d collectors", expectedCollectors)
	t.Logf("target assignment: %v", assignment)
	return assignment
}

// ---------------------------------------------------------------------------
// TA API helpers
// ---------------------------------------------------------------------------

func getTargetAssignment(t *testing.T, ctx context.Context, proxyBase string, expectedCollectors int) map[string][]string {
	t.Helper()
	result := make(map[string][]string)
	for i := 0; i < expectedCollectors; i++ {
		collectorID := fmt.Sprintf("collector-%d", i)
		path := fmt.Sprintf("%s/jobs/test-targets/targets", proxyBase)
		body, err := clientset.CoreV1().RESTClient().Get().
			AbsPath(path).
			Param("collector_id", collectorID).
			DoRaw(ctx)
		if err != nil {
			return nil
		}
		targets := parseTargetAddresses(body)
		result[collectorID] = targets
	}
	return result
}

func parseTargetAddresses(body []byte) []string {
	// Try array format: [{"targets":["addr"],"labels":{"__address__":"addr"}}]
	var groups []struct {
		Labels map[string]string `json:"labels"`
	}
	if err := json.Unmarshal(body, &groups); err == nil && len(groups) > 0 {
		var addresses []string
		for _, g := range groups {
			if addr, ok := g.Labels["__address__"]; ok {
				addresses = append(addresses, addr)
			}
		}
		sort.Strings(addresses)
		return addresses
	}

	// Try map format: {"hash":{"labels":{"__address__":"addr"}}}
	var items map[string]struct {
		Labels map[string]string `json:"labels"`
	}
	if err := json.Unmarshal(body, &items); err == nil {
		var addresses []string
		for _, item := range items {
			if addr, ok := item.Labels["__address__"]; ok {
				addresses = append(addresses, addr)
			}
		}
		sort.Strings(addresses)
		return addresses
	}
	return nil
}

func kubectlGetRaw(t *testing.T, ctx context.Context, path string) []byte {
	t.Helper()
	body, err := clientset.CoreV1().RESTClient().Get().AbsPath(path).DoRaw(ctx)
	require.NoError(t, err, "GET %s failed", path)
	return body
}

// ---------------------------------------------------------------------------
// Scale helpers
// ---------------------------------------------------------------------------

func scaleStatefulSet(t *testing.T, ctx context.Context, ns, name string, replicas int32) {
	t.Helper()
	t.Logf("scaling statefulset %s/%s to %d replicas", ns, name, replicas)
	scale, err := clientset.AppsV1().StatefulSets(ns).GetScale(ctx, name, metav1.GetOptions{})
	require.NoError(t, err)
	scale.Spec.Replicas = replicas
	_, err = clientset.AppsV1().StatefulSets(ns).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Assertion helpers
// ---------------------------------------------------------------------------

func allAssignedTargets(assignment map[string][]string) []string {
	seen := make(map[string]bool)
	for _, targets := range assignment {
		for _, t := range targets {
			seen[t] = true
		}
	}
	result := make([]string, 0, len(seen))
	for t := range seen {
		result = append(result, t)
	}
	sort.Strings(result)
	return result
}

func countStayedTargets(before, after map[string][]string) int {
	beforeByCollector := make(map[string]map[string]bool)
	for c, targets := range before {
		beforeByCollector[c] = make(map[string]bool)
		for _, t := range targets {
			beforeByCollector[c][t] = true
		}
	}
	count := 0
	for c, targets := range after {
		for _, t := range targets {
			if beforeByCollector[c][t] {
				count++
			}
		}
	}
	return count
}

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

func randomSuffix() string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.IntN(len(letters))]
	}
	return string(b)
}

func quotedTargets() string {
	parts := make([]string, len(testTargets))
	for i, t := range testTargets {
		parts[i] = fmt.Sprintf("%q", t)
	}
	return strings.Join(parts, ", ")
}

// splitImageNameTag splits "registry/name:tag" into ("registry/name", "tag").
func splitImageNameTag(img string) (string, string) {
	if idx := strings.LastIndex(img, ":"); idx > 0 {
		return img[:idx], img[idx+1:]
	}
	return img, "latest"
}

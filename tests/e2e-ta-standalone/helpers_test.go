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

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset        *kubernetes.Clientset
	restCfg          *rest.Config
	taImg            string
	collectorImg     string
	kustomizeBaseDir string

	collectorLabel = map[string]string{"app": "otel-collector"}
	testTargets    = []string{
		"target-a:8080", "target-b:8080", "target-c:8080",
		"target-d:8080", "target-e:8080", "target-f:8080",
	}
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

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: %v", err)
	}
	kustomizeBaseDir = filepath.Join(wd, "..", "..", "config", "target-allocator")
	if _, err := os.Stat(filepath.Join(kustomizeBaseDir, "kustomization.yaml")); err != nil {
		log.Fatalf("kustomize base not found at %s: %v", kustomizeBaseDir, err)
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	restCfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("failed to build kubeconfig: %v", err)
	}
	clientset, err = kubernetes.NewForConfig(restCfg)
	if err != nil {
		log.Fatalf("failed to create kubernetes client: %v", err)
	}

	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Test environment
// ---------------------------------------------------------------------------

// testEnv holds per-test state: a unique namespace and a context with timeout.
// Cleanup (namespace deletion + context cancel) is registered via t.Cleanup.
type testEnv struct {
	ns  string
	ctx context.Context
}

func newTestEnv(t *testing.T) testEnv {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)
	ns := createTestNamespace(t, ctx)
	t.Cleanup(func() { cleanupNamespace(t, ns) })
	return testEnv{ns: ns, ctx: ctx}
}

// ---------------------------------------------------------------------------
// Config builder
// ---------------------------------------------------------------------------

// TAConfigBuilder constructs TA YAML configuration for different test scenarios
// using a fluent API. Call build() to get the final YAML string.
type TAConfigBuilder struct {
	strategy       string
	staticTargets  []string
	k8sSDNamespace string
	enablePromCR   bool
}

func newTAConfig(strategy string) *TAConfigBuilder {
	return &TAConfigBuilder{strategy: strategy}
}

func (b *TAConfigBuilder) withStaticTargets(targets []string) *TAConfigBuilder {
	b.staticTargets = targets
	return b
}

func (b *TAConfigBuilder) withKubernetesSD(namespace string) *TAConfigBuilder {
	b.k8sSDNamespace = namespace
	return b
}

func (b *TAConfigBuilder) withPrometheusCR() *TAConfigBuilder {
	b.enablePromCR = true
	return b
}

func (b *TAConfigBuilder) build() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "allocation_strategy: %s\n", b.strategy)
	sb.WriteString("filter_strategy: relabel-config\n")
	sb.WriteString("collector_selector:\n  matchLabels:\n    app: otel-collector\n")

	if b.enablePromCR {
		sb.WriteString(`prometheus_cr:
  enabled: true
  scrape_interval: 30s
  service_monitor_selector: {}
  pod_monitor_selector: {}
`)
	}

	switch {
	case len(b.staticTargets) > 0:
		quoted := make([]string, len(b.staticTargets))
		for i, tgt := range b.staticTargets {
			quoted[i] = fmt.Sprintf("%q", tgt)
		}
		fmt.Fprintf(&sb, `config:
  scrape_configs:
    - job_name: test-targets
      scrape_interval: 30s
      static_configs:
        - targets: [%s]
`, strings.Join(quoted, ", "))
	case b.k8sSDNamespace != "":
		fmt.Fprintf(&sb, `config:
  scrape_configs:
    - job_name: per-node-targets
      scrape_interval: 30s
      kubernetes_sd_configs:
        - role: pod
          namespaces:
            names: [%s]
          selectors:
            - role: pod
              label: "app=scrape-target"
      relabel_configs:
        - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
          action: keep
          regex: "true"
`, b.k8sSDNamespace)
	case b.enablePromCR:
		sb.WriteString("config:\n  scrape_configs: []\n")
	}

	return sb.String()
}

// ---------------------------------------------------------------------------
// TA deployment helpers
// ---------------------------------------------------------------------------

// deployTA applies all resources from config/target-allocator/ into ns,
// then overwrites the ConfigMap with test-specific content.
// The kustomize overlay is a sibling of the base directory; --load-restrictor=LoadRestrictionsNone
// is required because kustomize v5 otherwise blocks references outside the overlay root.
func deployTA(t *testing.T, ctx context.Context, ns, taConfig string) {
	t.Helper()

	absBase, err := filepath.Abs(kustomizeBaseDir)
	require.NoError(t, err)

	// Temp overlay as a sibling of the base directory to avoid a kustomize cycle.
	parentDir := filepath.Dir(absBase)
	overlayDir, err := os.MkdirTemp(parentDir, ".test-overlay-*")
	require.NoError(t, err)
	defer os.RemoveAll(overlayDir)

	relBase, err := filepath.Rel(overlayDir, absBase)
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
  # Make cluster-scoped resources unique per test namespace to avoid collisions.
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

	// --load-restrictor=LoadRestrictionsNone is required when the overlay references
	// a sibling directory (relBase = "../target-allocator").
	out, err := exec.CommandContext(ctx, "kustomize", "build",
		"--load-restrictor=LoadRestrictionsNone", overlayDir).CombinedOutput()
	require.NoError(t, err, "kustomize build failed: %s", string(out))

	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(string(out))
	applyOut, err := cmd.CombinedOutput()
	require.NoError(t, err, "kubectl apply failed: %s", string(applyOut))
	t.Logf("applied TA manifests:\n%s", string(applyOut))

	// The base kustomization includes a ConfigMap with default content.
	// Overwrite it immediately with test-specific config before the pod starts.
	cm, err := clientset.CoreV1().ConfigMaps(ns).Get(ctx, "target-allocator", metav1.GetOptions{})
	require.NoError(t, err, "get TA ConfigMap")
	cm.Data = map[string]string{"targetallocator.yaml": taConfig}
	_, err = clientset.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{})
	require.NoError(t, err, "update TA ConfigMap with test config")
}

// collectorOpts customises StatefulSet scheduling for collector pods.
type collectorOpts struct {
	affinity                  *corev1.Affinity
	topologySpreadConstraints []corev1.TopologySpreadConstraint
}

func deployCollectors(t *testing.T, ctx context.Context, ns string, replicas int32) {
	t.Helper()
	deployCollectorsWithOpts(t, ctx, ns, replicas, nil)
}

func deployCollectorsWithOpts(t *testing.T, ctx context.Context, ns string, replicas int32, opts *collectorOpts) {
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

	var affinity *corev1.Affinity
	var spreadConstraints []corev1.TopologySpreadConstraint
	if opts != nil {
		affinity = opts.affinity
		spreadConstraints = opts.topologySpreadConstraints
	}

	_, err = clientset.AppsV1().StatefulSets(ns).Create(ctx, &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "collector"},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "collector",
			Selector:    &metav1.LabelSelector{MatchLabels: collectorLabel},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: collectorLabel},
				Spec: corev1.PodSpec{
					Affinity:                  affinity,
					TopologySpreadConstraints: spreadConstraints,
					Containers: []corev1.Container{{
						Name:  "collector",
						Image: collectorImg,
						Args:  []string{"--config=/conf/collector.yaml"},
						Env: []corev1.EnvVar{{
							Name:      "POD_NAME",
							ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}},
						}},
						Ports:        []corev1.ContainerPort{{ContainerPort: 8888, Name: "metrics"}},
						VolumeMounts: []corev1.VolumeMount{{Name: "config", MountPath: "/conf"}},
					}},
					Volumes: []corev1.Volume{{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: "collector-config"},
							},
						},
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
	t.Logf("waiting for deployment %s/%s → %d ready replicas", ns, name, replicas)
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
	t.Logf("waiting for statefulset %s/%s → %d ready replicas", ns, name, replicas)
	err := wait.PollUntilContextTimeout(ctx, pollInterval, testTimeout, true, func(ctx context.Context) (bool, error) {
		ss, err := clientset.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		return ss.Status.ReadyReplicas >= replicas, nil
	})
	require.NoError(t, err, "statefulset %s did not become ready", name)
}

func waitForTargetDistribution(t *testing.T, ctx context.Context, ns, jobName string, expectedCollectors int) map[string][]string {
	return waitForTargetDistributionWithPredicate(t, ctx, ns, jobName, expectedCollectors, func(a map[string][]string) bool {
		return len(allAssignedTargets(a)) == len(testTargets)
	})
}

func waitForTargetDistributionWithPredicate(t *testing.T, ctx context.Context, ns, jobName string, expectedCollectors int, done func(map[string][]string) bool) map[string][]string {
	t.Helper()
	proxyBase := taProxyBase(ns)

	var assignment map[string][]string
	err := wait.PollUntilContextTimeout(ctx, pollInterval, discoveryTimeout, true, func(ctx context.Context) (bool, error) {
		assignment = getTargetAssignment(t, ctx, proxyBase, jobName, expectedCollectors)
		if assignment == nil {
			return false, nil
		}
		if !done(assignment) {
			total := len(allAssignedTargets(assignment))
			t.Logf("target discovery: %d/%d targets assigned across %d collectors", total, len(testTargets), expectedCollectors)
			return false, nil
		}
		return true, nil
	})
	require.NoError(t, err, "targets not distributed to %d collectors for job %s", expectedCollectors, jobName)
	t.Logf("final assignment: %v", assignment)
	return assignment
}

// ---------------------------------------------------------------------------
// TA API helpers
// ---------------------------------------------------------------------------

func taProxyBase(ns string) string {
	return fmt.Sprintf("/api/v1/namespaces/%s/services/target-allocator:80/proxy", ns)
}

// fetchCollectorTargets is the raw HTTP primitive used by both getCollectorTargets
// and getTargetAssignment. Returns (body, true) on success, (nil, false) on error.
func fetchCollectorTargets(ctx context.Context, proxyBase, jobName, collectorID string) ([]byte, bool) {
	body, err := clientset.CoreV1().RESTClient().Get().
		AbsPath(fmt.Sprintf("%s/jobs/%s/targets", proxyBase, jobName)).
		Param("collector_id", collectorID).
		DoRaw(ctx)
	return body, err == nil
}

// getCollectorTargets returns the targets assigned to collectorID for jobName.
// Returns nil if the API call fails or the response contains no addresses.
func getCollectorTargets(t *testing.T, ctx context.Context, proxyBase, jobName, collectorID string) []string {
	t.Helper()
	body, ok := fetchCollectorTargets(ctx, proxyBase, jobName, collectorID)
	if !ok {
		return nil
	}
	return parseTargetAddresses(body)
}

// getTargetAssignment queries all expectedCollectors and returns a map of
// collectorID → assigned target addresses. Returns nil if any API call fails,
// signalling that the caller should retry.
func getTargetAssignment(t *testing.T, ctx context.Context, proxyBase, jobName string, expectedCollectors int) map[string][]string {
	t.Helper()
	result := make(map[string][]string)
	for i := 0; i < expectedCollectors; i++ {
		collectorID := fmt.Sprintf("collector-%d", i)
		body, ok := fetchCollectorTargets(ctx, proxyBase, jobName, collectorID)
		if !ok {
			return nil
		}
		result[collectorID] = parseTargetAddresses(body)
	}
	return result
}

func parseTargetAddresses(body []byte) []string {
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
	t.Logf("scaling statefulset %s/%s → %d", ns, name, replicas)
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
		for _, tgt := range targets {
			seen[tgt] = true
		}
	}
	result := make([]string, 0, len(seen))
	for tgt := range seen {
		result = append(result, tgt)
	}
	sort.Strings(result)
	return result
}

func countStayedTargets(before, after map[string][]string) int {
	beforeByCollector := make(map[string]map[string]bool)
	for c, targets := range before {
		beforeByCollector[c] = make(map[string]bool)
		for _, tgt := range targets {
			beforeByCollector[c][tgt] = true
		}
	}
	count := 0
	for c, targets := range after {
		for _, tgt := range targets {
			if beforeByCollector[c][tgt] {
				count++
			}
		}
	}
	return count
}

// ---------------------------------------------------------------------------
// Namespace lifecycle
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
	_ = clientset.RbacV1().ClusterRoleBindings().Delete(ctx, ns, metav1.DeleteOptions{})
	_ = clientset.RbacV1().ClusterRoles().Delete(ctx, ns, metav1.DeleteOptions{})
	_ = clientset.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
	t.Logf("cleaned up namespace %s and cluster RBAC", ns)
}

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

func randomSuffix() string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.IntN(len(letters))]
	}
	return string(b)
}

func splitImageNameTag(img string) (string, string) {
	if idx := strings.LastIndex(img, ":"); idx > 0 {
		return img[:idx], img[idx+1:]
	}
	return img, "latest"
}

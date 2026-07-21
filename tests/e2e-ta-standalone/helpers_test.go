// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package tastandalone

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"

	"github.com/open-telemetry/opentelemetry-operator/internal/testing/e2e"
)

// Infrastructure shared with the other e2e suites (namespace lifecycle, readiness
// waits, client construction, target-allocator RBAC, repo-root resolution) lives in
// internal/testing/e2e and is used through the e2e.* helpers below. This file keeps
// only what is specific to the standalone target allocator: kustomize-based deployment,
// the collector StatefulSet, the TA HTTP allocation API, scaling, and the config
// builder.

var (
	testenv      env.Environment
	taImg        string
	collectorImg string
	kustomizeBin string

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
	kustomizeBin = os.Getenv("KUSTOMIZE")
	if kustomizeBin == "" {
		kustomizeBin = "kustomize"
	}

	cfg, err := envconf.NewFromFlags()
	if err != nil {
		log.Fatalf("failed to parse e2e flags: %v", err)
	}
	testenv = env.NewWithConfig(cfg)
	os.Exit(testenv.Run(m))
}

// ---------------------------------------------------------------------------
// Namespace lifecycle
// ---------------------------------------------------------------------------

// nsContextKey is the context key used to store the per-feature namespace name.
type nsContextKey struct{}

// nsFromCtx retrieves the per-feature namespace stored by setupTestNamespace.
func nsFromCtx(ctx context.Context) string {
	return ctx.Value(nsContextKey{}).(string)
}

// setupTestNamespace creates a unique namespace for the current test feature, stores
// its name in the context, registers a t.Cleanup for teardown, and returns the updated
// context and namespace name. Cluster RBAC is created (and cleaned up) per binding by
// e2e.BindTargetAllocatorClusterRole, so namespace teardown only removes the namespace.
func setupTestNamespace(ctx context.Context, t *testing.T, cfg *envconf.Config) (nsCtx context.Context, ns string) {
	t.Helper()
	ns = envconf.RandomName("ta-test", 16)
	e2e.CreateNamespace(ctx, t, cfg, ns)
	t.Logf("created namespace %s", ns)
	t.Cleanup(func() {
		e2e.DeleteNamespace(context.WithoutCancel(ctx), t, cfg, ns)
		t.Logf("cleaned up namespace %s", ns)
	})
	return context.WithValue(ctx, nsContextKey{}, ns), ns
}

// ---------------------------------------------------------------------------
// Config builder
// ---------------------------------------------------------------------------

// TAConfigBuilder constructs TA YAML configuration for different test scenarios
// using a fluent API. Call build() to get the final YAML string.
type TAConfigBuilder struct {
	strategy      string
	staticTargets []string
	enablePromCR  bool
}

func newTAConfig(strategy string) *TAConfigBuilder {
	return &TAConfigBuilder{strategy: strategy}
}

func (b *TAConfigBuilder) withStaticTargets(targets []string) *TAConfigBuilder {
	b.staticTargets = targets
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
  scrape_config_selector: {}
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
	case b.enablePromCR:
		sb.WriteString("config:\n  scrape_configs: []\n")
	}

	return sb.String()
}

// ---------------------------------------------------------------------------
// TA deployment helpers
// ---------------------------------------------------------------------------

// deployTA applies the namespaced target-allocator resources from
// config/target-allocator/ into ns, binds the shared target-allocator ClusterRole to
// the TA ServiceAccount, then overwrites the ConfigMap with test-specific content.
//
// The overlay references only the namespaced base resources (ServiceAccount, ConfigMap,
// Deployment, Service); the cluster-scoped ClusterRole/ClusterRoleBinding are provided
// by e2e.BindTargetAllocatorClusterRole instead, so the cluster RBAC is shared with the
// other e2e suites and cleaned up per binding. The overlay is a sibling of the base
// directory; --load-restrictor=LoadRestrictionsNone is required because kustomize v5
// otherwise blocks references outside the overlay root.
func deployTA(t *testing.T, ctx context.Context, cfg *envconf.Config, ns, taConfig string) {
	t.Helper()

	absBase, err := filepath.Abs(filepath.Join(e2e.RepoRoot(t), "config", "target-allocator"))
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
  - %[4]s/serviceaccount.yaml
  - %[4]s/configmap.yaml
  - %[4]s/deployment.yaml
  - %[4]s/service.yaml

images:
  - name: target-allocator
    newName: %[2]s
    newTag: "%[3]s"
`, ns, imgName, imgTag, relBase)

	err = os.WriteFile(filepath.Join(overlayDir, "kustomization.yaml"), []byte(overlayContent), 0o600)
	require.NoError(t, err)

	// --load-restrictor=LoadRestrictionsNone is required when the overlay references
	// a sibling directory (relBase = "../target-allocator").
	out, err := exec.CommandContext(ctx, kustomizeBin, "build",
		"--load-restrictor=LoadRestrictionsNone", overlayDir).CombinedOutput()
	require.NoError(t, err, "kustomize build failed: %s", string(out))

	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(string(out))
	applyOut, err := cmd.CombinedOutput()
	require.NoError(t, err, "kubectl apply failed: %s", string(applyOut))
	t.Logf("applied TA manifests:\n%s", string(applyOut))

	// The standalone TA needs the project's target-allocator ClusterRole bound to its
	// ServiceAccount; the operator does not create this RBAC for a standalone TA.
	e2e.BindTargetAllocatorClusterRole(ctx, t, cfg, ns, "target-allocator")

	// The base kustomization includes a ConfigMap with default content.
	// Overwrite it immediately with test-specific config before the pod starts.
	cs := e2e.ClientSet(t, cfg)
	cm, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, "target-allocator", metav1.GetOptions{})
	require.NoError(t, err, "get TA ConfigMap")
	cm.Data = map[string]string{"targetallocator.yaml": taConfig}
	_, err = cs.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{})
	require.NoError(t, err, "update TA ConfigMap with test config")
}

func deployCollectors(t *testing.T, ctx context.Context, cfg *envconf.Config, ns string, replicas int32) {
	t.Helper()
	cs := e2e.ClientSet(t, cfg)

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
	_, err := cs.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "collector-config"},
		Data:       map[string]string{"collector.yaml": collectorConfig},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = cs.AppsV1().StatefulSets(ns).Create(ctx, &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "collector"},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "collector",
			Selector:    &metav1.LabelSelector{MatchLabels: collectorLabel},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: collectorLabel},
				Spec: corev1.PodSpec{
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
// Target distribution polling
// ---------------------------------------------------------------------------

func waitForTargetDistribution(t *testing.T, ctx context.Context, cfg *envconf.Config, ns, jobName string, expectedCollectors int) map[string][]string {
	return waitForTargetDistributionWithPredicate(t, ctx, cfg, ns, jobName, expectedCollectors, func(a map[string][]string) bool {
		return len(allAssignedTargets(a)) == len(testTargets)
	})
}

func waitForTargetDistributionWithPredicate(t *testing.T, ctx context.Context, cfg *envconf.Config, ns, jobName string, expectedCollectors int, done func(map[string][]string) bool) map[string][]string {
	t.Helper()
	cs := e2e.ClientSet(t, cfg)
	proxyBase := taProxyBase(ns)

	var assignment map[string][]string
	err := wait.PollUntilContextTimeout(ctx, pollInterval, discoveryTimeout, true, func(ctx context.Context) (bool, error) {
		assignment = getTargetAssignment(t, ctx, cs, proxyBase, jobName, expectedCollectors)
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

// getTargetAssignment queries all expectedCollectors and returns a map of
// collectorID → assigned target addresses. Returns nil if any API call fails,
// signaling that the caller should retry.
func getTargetAssignment(t *testing.T, ctx context.Context, cs *kubernetes.Clientset, proxyBase, jobName string, expectedCollectors int) map[string][]string {
	t.Helper()
	result := make(map[string][]string)
	for i := range expectedCollectors {
		collectorID := fmt.Sprintf("collector-%d", i)
		body, err := cs.CoreV1().RESTClient().Get().
			AbsPath(fmt.Sprintf("%s/jobs/%s/targets", proxyBase, jobName)).
			Param("collector_id", collectorID).
			DoRaw(ctx)
		if err != nil {
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
		slices.Sort(addresses)
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
		slices.Sort(addresses)
		return addresses
	}
	return nil
}

func kubectlGetRaw(t *testing.T, ctx context.Context, cfg *envconf.Config, path string) []byte {
	t.Helper()
	body, err := e2e.ClientSet(t, cfg).CoreV1().RESTClient().Get().AbsPath(path).DoRaw(ctx)
	require.NoError(t, err, "GET %s failed", path)
	return body
}

// ---------------------------------------------------------------------------
// Scale helpers
// ---------------------------------------------------------------------------

func scaleStatefulSet(t *testing.T, ctx context.Context, cfg *envconf.Config, ns, name string, replicas int32) {
	t.Helper()
	t.Logf("scaling statefulset %s/%s → %d", ns, name, replicas)
	cs := e2e.ClientSet(t, cfg)
	scale, err := cs.AppsV1().StatefulSets(ns).GetScale(ctx, name, metav1.GetOptions{})
	require.NoError(t, err)
	scale.Spec.Replicas = replicas
	_, err = cs.AppsV1().StatefulSets(ns).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
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
	slices.Sort(result)
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
// Utilities
// ---------------------------------------------------------------------------

func splitImageNameTag(img string) (name, tag string) {
	if idx := strings.LastIndex(img, ":"); idx > 0 {
		return img[:idx], img[idx+1:]
	}
	return img, "latest"
}

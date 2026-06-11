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

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	monitoringclient "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type (
	promCRStateKey struct{}
	promCRState    struct {
		mclient *monitoringclient.Clientset
		smName  string
		pmName  string
		scName  string
	}
)

// TestPrometheusCRTargetAllocator validates ServiceMonitor and PodMonitor
// discovery in standalone TA mode (prometheus_cr.enabled: true).
//
// The test deploys:
//   - A workload (metrics-basic-auth image) exposed as a Service
//   - A ServiceMonitor targeting that Service
//   - A PodMonitor targeting the workload pods
//
// Both CRs are created before the TA starts so the informer picks them up on
// first sync. The test then:
//
//  1. Asserts both SM and PM job names appear in /scrape_configs
//  2. Deletes the ServiceMonitor
//  3. Asserts the SM job disappears but the PM job remains
//
// Prerequisites: ServiceMonitor/PodMonitor CRDs must be installed
// (hack/install-targetallocator-prometheus-crds.sh, run by prepare-e2e-ta-standalone).
func TestPrometheusCRTargetAllocator(t *testing.T) {
	feat := features.New("prometheus CR discovery in standalone TA").
		Setup(func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			var ns string
			ctx, ns = setupTestNamespace(ctx, t)

			mclient := newMonitoringClient(t)

			// Deploy workload and monitoring CRs BEFORE starting the TA so the informer
			// picks them up during its first sync.
			deployMetricsWorkload(t, ctx, ns)
			smName := deployServiceMonitor(t, ctx, mclient, ns)
			pmName := deployPodMonitor(t, ctx, mclient, ns)
			scName := deployScrapeConfig(t, ctx, mclient, ns)

			taConfig := newTAConfig("consistent-hashing").withPrometheusCR().build()
			deployTA(t, ctx, ns, taConfig)
			waitForDeploymentReady(t, ctx, ns, "target-allocator", 1)

			deployCollectors(t, ctx, ns, 1)
			waitForStatefulSetReady(t, ctx, ns, "collector", 1)

			return context.WithValue(ctx, promCRStateKey{}, promCRState{
				mclient: mclient,
				smName:  smName,
				pmName:  pmName,
				scName:  scName,
			})
		}).
		Assess("ServiceMonitor, PodMonitor and ScrapeConfig targets discovered", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			ns := nsFromCtx(ctx)
			state := ctx.Value(promCRStateKey{}).(promCRState)
			proxyBase := taProxyBase(ns)

			// TA informer resync is 30s; allow up to 90s for all CRs to appear.
			smJobName := fmt.Sprintf("serviceMonitor/%s/%s/0", ns, state.smName)
			pmJobName := fmt.Sprintf("podMonitor/%s/%s/0", ns, state.pmName)
			scJobName := fmt.Sprintf("scrapeConfig/%s/%s", ns, state.scName)

			waitForJobInScrapeConfigs(t, ctx, proxyBase, smJobName)
			waitForJobInScrapeConfigs(t, ctx, proxyBase, pmJobName)
			waitForJobInScrapeConfigs(t, ctx, proxyBase, scJobName)
			t.Logf("SM, PM and ScrapeConfig jobs found in /scrape_configs")
			return ctx
		}).
		Assess("ServiceMonitor targets disappear after deletion", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			ns := nsFromCtx(ctx)
			state := ctx.Value(promCRStateKey{}).(promCRState)
			proxyBase := taProxyBase(ns)

			smJobName := fmt.Sprintf("serviceMonitor/%s/%s/0", ns, state.smName)
			pmJobName := fmt.Sprintf("podMonitor/%s/%s/0", ns, state.pmName)
			scJobName := fmt.Sprintf("scrapeConfig/%s/%s", ns, state.scName)

			err := state.mclient.MonitoringV1().ServiceMonitors(ns).Delete(ctx, state.smName, metav1.DeleteOptions{})
			require.NoError(t, err, "delete ServiceMonitor %s", state.smName)
			t.Logf("deleted ServiceMonitor %s", state.smName)

			// Wait for SM job to disappear (informer resync ~30s + rate limit ~5s).
			err = wait.PollUntilContextTimeout(ctx, 5*time.Second, discoveryTimeout, false, func(ctx context.Context) (bool, error) {
				body, err := clientset.CoreV1().RESTClient().Get().
					AbsPath(proxyBase + "/scrape_configs").DoRaw(ctx)
				if err != nil {
					return false, nil
				}
				return !strings.Contains(string(body), smJobName), nil
			})
			require.NoError(t, err, "ServiceMonitor job %s should disappear from /scrape_configs after deletion", smJobName)
			t.Logf("ServiceMonitor job %s removed from /scrape_configs", smJobName)

			// PodMonitor and ScrapeConfig jobs must still be present.
			body := kubectlGetRaw(t, ctx, proxyBase+"/scrape_configs")
			assert.Contains(t, string(body), pmJobName,
				"PodMonitor job %s should still be present in /scrape_configs", pmJobName)
			assert.Contains(t, string(body), scJobName,
				"ScrapeConfig job %s should still be present in /scrape_configs", scJobName)
			return ctx
		}).
		Feature()

	testenv.Test(t, feat)
}

// ---------------------------------------------------------------------------
// Workload + monitoring CR deployment
// ---------------------------------------------------------------------------

const metricsAppImage = "ghcr.io/open-telemetry/opentelemetry-operator/e2e-test-app-metrics-basic-auth:main"

func deployMetricsWorkload(t *testing.T, ctx context.Context, ns string) {
	t.Helper()
	labels := map[string]string{"app": "metrics-app"}
	replicas := int32(1)

	_, err := clientset.AppsV1().Deployments(ns).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "metrics-app"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "app",
						Image: metricsAppImage,
						Ports: []corev1.ContainerPort{{ContainerPort: 9123, Name: "metrics"}},
					}},
				},
			},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err, "create metrics-app Deployment")

	_, err = clientset.CoreV1().Services(ns).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "metrics-app", Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Name:       "metrics",
				Port:       9123,
				TargetPort: intstr.FromString("metrics"),
			}},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err, "create metrics-app Service")

	// Wait for the Deployment to have at least one ready pod so the
	// ServiceMonitor and PodMonitor endpoints resolve.
	err = wait.PollUntilContextTimeout(ctx, pollInterval, testTimeout, true, func(ctx context.Context) (bool, error) {
		d, err := clientset.AppsV1().Deployments(ns).Get(ctx, "metrics-app", metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		return d.Status.ReadyReplicas >= 1, nil
	})
	require.NoError(t, err, "metrics-app deployment did not become ready")
}

func deployServiceMonitor(t *testing.T, ctx context.Context, mclient *monitoringclient.Clientset, ns string) string {
	t.Helper()
	sm := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metrics-app-sm",
			Namespace: ns,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "metrics-app"},
			},
			Endpoints: []monitoringv1.Endpoint{{
				Port: "metrics",
			}},
		},
	}
	created, err := mclient.MonitoringV1().ServiceMonitors(ns).Create(ctx, sm, metav1.CreateOptions{})
	require.NoError(t, err, "create ServiceMonitor")
	t.Logf("created ServiceMonitor %s/%s", ns, created.Name)
	return created.Name
}

func deployPodMonitor(t *testing.T, ctx context.Context, mclient *monitoringclient.Clientset, ns string) string {
	t.Helper()
	pm := &monitoringv1.PodMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metrics-app-pm",
			Namespace: ns,
		},
		Spec: monitoringv1.PodMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "metrics-app"},
			},
			PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{{
				Port: func() *string { s := "metrics"; return &s }(),
			}},
		},
	}
	created, err := mclient.MonitoringV1().PodMonitors(ns).Create(ctx, pm, metav1.CreateOptions{})
	require.NoError(t, err, "create PodMonitor")
	t.Logf("created PodMonitor %s/%s", ns, created.Name)
	return created.Name
}

func deployScrapeConfig(t *testing.T, ctx context.Context, mclient *monitoringclient.Clientset, ns string) string {
	t.Helper()
	sc := &monitoringv1alpha1.ScrapeConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metrics-app-sc",
			Namespace: ns,
		},
		Spec: monitoringv1alpha1.ScrapeConfigSpec{
			StaticConfigs: []monitoringv1alpha1.StaticConfig{{
				Targets: []monitoringv1alpha1.Target{"metrics-app:9123"},
			}},
		},
	}
	created, err := mclient.MonitoringV1alpha1().ScrapeConfigs(ns).Create(ctx, sc, metav1.CreateOptions{})
	require.NoError(t, err, "create ScrapeConfig")
	t.Logf("created ScrapeConfig %s/%s", ns, created.Name)
	return created.Name
}

// ---------------------------------------------------------------------------
// Wait helper for PromCR discovery
// ---------------------------------------------------------------------------

func waitForJobInScrapeConfigs(t *testing.T, ctx context.Context, proxyBase, jobName string) {
	t.Helper()
	t.Logf("waiting for job %q to appear in /scrape_configs", jobName)
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, discoveryTimeout, true, func(ctx context.Context) (bool, error) {
		body, err := clientset.CoreV1().RESTClient().Get().
			AbsPath(proxyBase + "/scrape_configs").DoRaw(ctx)
		if err != nil {
			return false, nil
		}
		found := strings.Contains(string(body), jobName)
		if !found {
			t.Logf("job %q not yet in /scrape_configs, retrying…", jobName)
		}
		return found, nil
	})
	require.NoError(t, err, "job %q did not appear in /scrape_configs within timeout", jobName)
}

// ---------------------------------------------------------------------------
// Monitoring client factory
// ---------------------------------------------------------------------------

func newMonitoringClient(t *testing.T) *monitoringclient.Clientset {
	t.Helper()
	mclient, err := monitoringclient.NewForConfig(restCfg)
	require.NoError(t, err, "create monitoring client")
	return mclient
}

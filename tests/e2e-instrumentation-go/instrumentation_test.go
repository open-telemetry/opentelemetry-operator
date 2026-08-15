// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package instrumentation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/open-telemetry/opentelemetry-operator/internal/testing/e2e"
)

// testCase is one auto-instrumentation scenario: an Instrumentation CR, an app
// Deployment carrying the inject annotation, and the telemetry the instrumented app
// must deliver end-to-end (app -> agent -> collector -> otlp-sink).
type testCase struct {
	name            string
	instrumentation string            // testdata manifest with the Instrumentation CR
	app             string            // testdata manifest with the app Deployment
	appData         map[string]string // template data for the app manifest

	// initContainers the webhook must have injected into the app pod. Fast, precise
	// failure signal: if these are missing, injection itself failed and there is no
	// point waiting on telemetry.
	initContainers []string
	// podCheck optionally asserts language-specific pod spec details.
	podCheck func(t *testing.T, pod *corev1.Pod)

	// externalInstrumentation marks the Instrumentation CR as applied by the test
	// itself (e.g. into another namespace); the runner then does not apply it into
	// the test namespace.
	externalInstrumentation bool

	// podNameInstanceID marks scenarios where the injection path derives
	// service.instance.id from the pod name alone: the apache-httpd/nginx SDK
	// injection predates the <namespace>.<pod>.<container> format the language
	// SDK path uses.
	podNameInstanceID bool

	// span, when set, must eventually be received by the sink. The runner fills in
	// the namespace-dependent resource identity.
	span *e2e.SpanMatch
	// metric, when set, must eventually be received by the sink.
	metric *e2e.MetricMatch
}

// The HTTP span attribute expectations below assert stable values through
// whichever attribute name the agent's semconv version uses.
var (
	httpMethodGet = e2e.AttrMatch{Keys: []string{"http.request.method", "http.method"}, Value: "GET"}
	httpStatus200 = e2e.AttrMatch{Keys: []string{"http.response.status_code", "http.status_code"}, Value: "200"}
)

func urlPath(path string) e2e.AttrMatch {
	return e2e.AttrMatch{Keys: []string{"url.path", "http.target"}, Value: path}
}

var testCases = []testCase{
	{
		name:            "java",
		instrumentation: "instrumentation-java.yaml",
		app:             "app-java.yaml",
		appData:         map[string]string{"Inject": "true"},
		initContainers:  []string{"opentelemetry-auto-instrumentation-java"},
		span: &e2e.SpanMatch{
			Resource:       map[string]string{"telemetry.sdk.language": "java"},
			Attrs:          []e2e.AttrMatch{httpMethodGet, httpStatus200, urlPath("/")},
			Kind:           ptrace.SpanKindServer,
			StatusNotError: true,
		},
		// The metric name depends on the agent's semconv: 1.x emits
		// process.runtime.jvm.memory.usage, 2.x emits jvm.memory.used. Accepting
		// either keeps the assertion valid across agent updates.
		metric: &e2e.MetricMatch{
			NamesAnyOf: []string{"process.runtime.jvm.memory.usage", "jvm.memory.used"},
			Resource:   map[string]string{"telemetry.sdk.language": "java"},
			Type:       pmetric.MetricTypeSum,
			Unit:       "By",
			NonZero:    true,
		},
	},
	{
		name:            "dotnet",
		instrumentation: "instrumentation-dotnet.yaml",
		app:             "app-dotnet.yaml",
		initContainers:  []string{"opentelemetry-auto-instrumentation-dotnet"},
		span: &e2e.SpanMatch{
			Resource:       map[string]string{"telemetry.sdk.language": "dotnet"},
			Attrs:          []e2e.AttrMatch{httpMethodGet, httpStatus200, urlPath("/rolldice")},
			Kind:           ptrace.SpanKindServer,
			StatusNotError: true,
		},
		metric: &e2e.MetricMatch{
			NamesAnyOf: []string{"process.cpu.time"},
			Resource:   map[string]string{"telemetry.sdk.language": "dotnet"},
			Type:       pmetric.MetricTypeSum,
			Unit:       "s",
			NonZero:    true,
		},
	},
	{
		name:            "nodejs",
		instrumentation: "instrumentation-nodejs.yaml",
		app:             "app-nodejs.yaml",
		initContainers:  []string{"opentelemetry-auto-instrumentation-nodejs"},
		span: &e2e.SpanMatch{
			Resource:       map[string]string{"telemetry.sdk.language": "nodejs"},
			Attrs:          []e2e.AttrMatch{httpMethodGet, httpStatus200, urlPath("/rolldice")},
			Kind:           ptrace.SpanKindServer,
			StatusNotError: true,
		},
		// The two names differ in unit (ms vs s), so the unit is not asserted.
		metric: &e2e.MetricMatch{
			NamesAnyOf: []string{"http.server.duration", "http.server.request.duration"},
			Resource:   map[string]string{"telemetry.sdk.language": "nodejs"},
			Type:       pmetric.MetricTypeHistogram,
			NonZero:    true,
		},
	},
	{
		name:            "nodejs-volume",
		instrumentation: "instrumentation-nodejs-volume.yaml",
		app:             "app-nodejs.yaml",
		initContainers:  []string{"opentelemetry-auto-instrumentation-nodejs"},
		// The Instrumentation CR sets a volumeClaimTemplate for the agent volume; it
		// must materialize as a generic ephemeral volume instead of an emptyDir.
		podCheck: func(t *testing.T, pod *corev1.Pod) {
			for _, v := range pod.Spec.Volumes {
				if v.Name == "opentelemetry-auto-instrumentation-nodejs" {
					assert.NotNil(t, v.Ephemeral, "agent volume %q is not a generic ephemeral volume: %+v", v.Name, v.VolumeSource)
					return
				}
			}
			t.Error("agent volume opentelemetry-auto-instrumentation-nodejs not found")
		},
		span: &e2e.SpanMatch{
			Resource:       map[string]string{"telemetry.sdk.language": "nodejs"},
			Attrs:          []e2e.AttrMatch{httpMethodGet, httpStatus200, urlPath("/rolldice")},
			Kind:           ptrace.SpanKindServer,
			StatusNotError: true,
		},
	},
	{
		name:            "python",
		instrumentation: "instrumentation-python.yaml",
		app:             "app-python.yaml",
		appData:         map[string]string{"Image": "ghcr.io/open-telemetry/opentelemetry-operator/e2e-test-app-python:main"},
		initContainers:  []string{"opentelemetry-auto-instrumentation-python"},
		span: &e2e.SpanMatch{
			Resource:       map[string]string{"telemetry.sdk.language": "python"},
			Attrs:          []e2e.AttrMatch{httpMethodGet, httpStatus200, urlPath("/")},
			Kind:           ptrace.SpanKindServer,
			StatusNotError: true,
		},
	},
	{
		// The same scenario against the oldest supported Python.
		name:            "python-oldest",
		instrumentation: "instrumentation-python.yaml",
		app:             "app-python.yaml",
		appData:         map[string]string{"Image": "ghcr.io/open-telemetry/opentelemetry-operator/e2e-test-app-python:main-3.10"},
		initContainers:  []string{"opentelemetry-auto-instrumentation-python"},
		span: &e2e.SpanMatch{
			Resource:       map[string]string{"telemetry.sdk.language": "python"},
			Attrs:          []e2e.AttrMatch{httpMethodGet, httpStatus200, urlPath("/")},
			Kind:           ptrace.SpanKindServer,
			StatusNotError: true,
		},
	},
	{
		name:            "python-musl",
		instrumentation: "instrumentation-python-musl.yaml",
		app:             "app-python-musl.yaml",
		initContainers:  []string{"opentelemetry-auto-instrumentation-python"},
		// system_metrics is experimental in python (and implements this semconv
		// UpDownCounter as a gauge), so per the stability policy only the unit and
		// a non-zero value are asserted, not the type.
		metric: &e2e.MetricMatch{
			NamesAnyOf: []string{"system.memory.usage"},
			Resource:   map[string]string{"telemetry.sdk.language": "python"},
			Unit:       "By",
			NonZero:    true,
		},
	},
	{
		name:              "apache-httpd",
		instrumentation:   "instrumentation-apache-httpd.yaml",
		app:               "app-apache-httpd.yaml",
		initContainers:    []string{"otel-agent-source-container-clone", "otel-agent-attach-apache"},
		podNameInstanceID: true,
		span: &e2e.SpanMatch{
			Resource:       map[string]string{"telemetry.sdk.language": "cpp", "webengine.name": "Apache"},
			Attrs:          []e2e.AttrMatch{httpMethodGet, httpStatus200},
			Kind:           ptrace.SpanKindServer,
			StatusNotError: true,
		},
	},
	{
		name:              "nginx",
		instrumentation:   "instrumentation-nginx.yaml",
		app:               "app-nginx.yaml",
		initContainers:    []string{"otel-agent-source-container-clone", "otel-agent-attach-nginx"},
		podNameInstanceID: true,
		span: &e2e.SpanMatch{
			Resource:       map[string]string{"telemetry.sdk.language": "cpp", "webengine.name": "Nginx"},
			Attrs:          []e2e.AttrMatch{httpMethodGet, httpStatus200},
			Kind:           ptrace.SpanKindServer,
			StatusNotError: true,
		},
	},
	{
		name:              "nginx-container-security-context",
		instrumentation:   "instrumentation-nginx.yaml",
		app:               "app-nginx-secctx.yaml",
		initContainers:    []string{"otel-agent-source-container-clone", "otel-agent-attach-nginx"},
		podNameInstanceID: true,
		span: &e2e.SpanMatch{
			Resource:       map[string]string{"telemetry.sdk.language": "cpp", "webengine.name": "Nginx"},
			Attrs:          []e2e.AttrMatch{httpMethodGet, httpStatus200},
			Kind:           ptrace.SpanKindServer,
			StatusNotError: true,
		},
	},
}

func TestInstrumentation(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testenv.Test(t, instrumentationFeature(tc, nil))
		})
	}
}

// TestJavaCrossNamespaceInstrumentation verifies an app can reference an
// Instrumentation CR living in another namespace ("<namespace>/<name>" annotation).
func TestJavaCrossNamespaceInstrumentation(t *testing.T) {
	t.Parallel()
	tc := caseByName(t, "java")
	tc.name = "java-other-ns"
	tc.externalInstrumentation = true

	// The Instrumentation CR lives in a second namespace of its own, applied before
	// the app so the webhook can resolve the cross-namespace reference.
	appData := func(ctx context.Context, t *testing.T, cfg *envconf.Config) map[string]string {
		instNS := e2e.NamespaceFromT(t)
		e2e.CreateNamespace(ctx, t, cfg, instNS)
		t.Cleanup(func() { e2e.DeleteNamespace(context.WithoutCancel(ctx), t, cfg, instNS) })
		e2e.Apply(ctx, t, cfg, instNS, render("instrumentation-java.yaml", nil))
		return map[string]string{"Inject": instNS + "/java"}
	}
	testenv.Test(t, instrumentationFeature(tc, appData))
}

// instrumentationFeature builds the feature that runs one scenario: deploy the sink,
// the Instrumentation CR, the collector and the app into a fresh namespace, assert the
// webhook injected the agent, then assert the telemetry the sink received. appData,
// when non-nil, runs first and supersedes tc.appData, so a variant can template the app
// from cluster state it establishes itself (see the cross-namespace test).
func instrumentationFeature(tc testCase, appData func(context.Context, *testing.T, *envconf.Config) map[string]string) features.Feature {
	// Populated by Setup and the injection assessment, consumed by the telemetry one.
	var (
		app      appInfo
		pod      *corev1.Pod
		sink     *e2e.OTLPSink
		injected bool
	)

	return features.New(tc.name+" auto-instrumentation").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			data := tc.appData
			if appData != nil {
				data = appData(ctx, t, cfg)
			}

			ctx = e2e.SetupNamespace(ctx, t, cfg)
			ns := e2e.Namespace(t, ctx)
			e2e.DumpNamespaceOnFailure(ctx, t, cfg, ns)

			sink = e2e.DeployOTLPSink(ctx, t, cfg, ns, sinkImage)
			if !tc.externalInstrumentation {
				e2e.Apply(ctx, t, cfg, ns, render(tc.instrumentation, nil))
			}
			e2e.Apply(ctx, t, cfg, ns, render("collector.yaml", nil))
			// Waiting for the collector is also the barrier that guarantees the webhook
			// has seen the Instrumentation CR before the app pod is created.
			e2e.WaitForDeployment(ctx, t, cfg, ns, "deployment-collector", 3*time.Minute)

			appManifest := render(tc.app, data)
			app = parseAppInfo(t, appManifest)
			e2e.Apply(ctx, t, cfg, ns, appManifest)
			e2e.WaitForDeployment(ctx, t, cfg, ns, app.deployment, 5*time.Minute)
			return ctx
		}).
		Assess("the webhook injected the agent into the app pod", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			pod = verifyInjection(ctx, t, cfg, e2e.Namespace(t, ctx), tc, app)
			injected = !t.Failed()
			return ctx
		}).
		Assess("the instrumented app delivers telemetry end to end", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Assessments are separate subtests, so a failed injection does not stop
			// this one on its own — and without injection telemetry could only time out.
			if !injected {
				t.Skip("injection assessment failed")
			}
			assertTelemetry(ctx, t, cfg, e2e.Namespace(t, ctx), tc, app, pod, sink)
			return ctx
		}).
		Feature()
}

// caseByName returns a copy of the named table entry, for tests that derive a
// variant from it.
func caseByName(t *testing.T, name string) testCase {
	t.Helper()
	for _, tc := range testCases {
		if tc.name == name {
			return tc
		}
	}
	require.FailNow(t, "no test case named "+name)
	return testCase{}
}

// appInfo is what the tests need to know about the sample app — parsed from its
// rendered manifest, so the table does not duplicate names, selectors and ports.
// The request path driven by the test is the app's readiness probe path: it is
// the endpoint the manifest already declares as meaningful.
type appInfo struct {
	deployment string
	selector   string
	container  string
	port       int
	path       string
}

func parseAppInfo(t *testing.T, manifest string) appInfo {
	t.Helper()
	dec := utilyaml.NewYAMLOrJSONDecoder(strings.NewReader(manifest), 4096)
	for {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err, "decode app manifest")
		}
		var head struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(raw, &head); err != nil || head.Kind != "Deployment" {
			continue
		}
		dep := &appsv1.Deployment{}
		require.NoError(t, json.Unmarshal(raw, dep), "parse app Deployment")
		sel, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
		require.NoError(t, err, "app Deployment %s selector", dep.Name)
		c := dep.Spec.Template.Spec.Containers[0]
		info := appInfo{
			deployment: dep.Name,
			selector:   sel.String(),
			container:  c.Name,
			path:       "/",
		}
		if len(c.Ports) > 0 {
			info.port = int(c.Ports[0].ContainerPort)
		}
		if p := c.ReadinessProbe; p != nil && p.HTTPGet != nil && p.HTTPGet.Path != "" {
			info.path = p.HTTPGet.Path
		}
		require.NotZero(t, info.port, "app Deployment %s: first container declares no port", dep.Name)
		return info
	}
	t.Fatal("no Deployment found in app manifest")
	return appInfo{}
}

// appRequestInterval is how often driveApp requests the instrumented app.
const appRequestInterval = time.Second

// driveApp requests the instrumented app over the API server's pod proxy until the
// test ends, so request-scoped telemetry keeps being produced while the assertions
// poll. Some agents only emit a server span per request, and the python
// Instrumentation samples at 25%, so a single request before asserting would flake.
// Request errors are ignored: the workload may still be starting, and if telemetry
// never arrives the assertions report what did.
func driveApp(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string, app appInfo) {
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)
	stopped := make(chan struct{})
	t.Cleanup(func() {
		cancel()
		<-stopped
	})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(appRequestInterval)
		defer ticker.Stop()
		for {
			//nolint:errcheck // see the doc comment: request errors are expected while the app starts.
			e2e.PodHTTPGet(ctx, cfg, ns, app.selector, app.port, app.path)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// assertTelemetry asserts the scenario's expected span and metric arrive at the
// sink carrying the workload's full resource identity.
func assertTelemetry(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string, tc testCase, app appInfo, pod *corev1.Pod, sink *e2e.OTLPSink) {
	t.Helper()
	identity := resourceIdentity(ns, pod, tc, app)
	driveApp(ctx, t, cfg, ns, app)

	// The operator injects the workload's identity via OTEL_SERVICE_NAME and
	// OTEL_RESOURCE_ATTRIBUTES; requiring the full identity, with exact values
	// derived from cluster state, proves it survived to the backend.
	if tc.span != nil {
		want := *tc.span
		want.Resource = mergeResource(identity, want.Resource)
		sink.EventuallySpans(ctx, t, cfg, e2e.HasSpan(want))
	}
	if tc.metric != nil {
		want := *tc.metric
		want.Resource = mergeResource(identity, want.Resource)
		sink.EventuallyMetrics(ctx, t, cfg, e2e.HasMetric(want))
	}
}

// resourceIdentity returns the resource attributes the operator injects into the
// instrumented workload, with expected values derived from live cluster state.
// These are operator-owned end to end and asserted exactly, unlike agent-owned
// telemetry (see the README's stability policy).
func resourceIdentity(ns string, pod *corev1.Pod, tc testCase, app appInfo) map[string]string {
	instanceID := fmt.Sprintf("%s.%s.%s", ns, pod.Name, app.container)
	if tc.podNameInstanceID {
		instanceID = pod.Name
	}
	id := map[string]string{
		"service.name":        app.deployment,
		"service.namespace":   ns,
		"service.instance.id": instanceID,
		"k8s.namespace.name":  ns,
		"k8s.pod.name":        pod.Name,
		"k8s.container.name":  app.container,
		"k8s.deployment.name": app.deployment,
		"k8s.node.name":       pod.Spec.NodeName,
	}
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			id["k8s.replicaset.name"] = owner.Name
		}
	}
	// The operator derives service.version from the app container's image tag.
	for _, c := range pod.Spec.Containers {
		if c.Name == app.container {
			if tag := imageTag(c.Image); tag != "" {
				id["service.version"] = tag
			}
		}
	}
	return id
}

// imageTag extracts the tag from an image reference, ignoring any digest.
func imageTag(image string) string {
	if at := strings.Index(image, "@"); at >= 0 {
		image = image[:at]
	}
	idx := strings.LastIndex(image, ":")
	if idx < 0 || strings.Contains(image[idx:], "/") {
		return ""
	}
	return image[idx+1:]
}

// mergeResource overlays row-specific resource expectations onto the derived
// identity.
func mergeResource(identity, extra map[string]string) map[string]string {
	merged := maps.Clone(identity)
	maps.Copy(merged, extra)
	return merged
}

// verifyInjection asserts the webhook actually mutated the app pod: the agent init
// containers exist and the instrumented container carries the OTEL_SERVICE_NAME the
// SDK will report. This fails fast and precisely when injection is broken, instead
// of a telemetry timeout. It returns the pod so the caller can derive the expected
// resource identity from it.
func verifyInjection(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string, tc testCase, app appInfo) *corev1.Pod {
	t.Helper()
	cs := e2e.ClientSet(t, cfg)
	pods, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: app.selector})
	require.NoError(t, err, "list app pods")
	var pod *corev1.Pod
	for i := range pods.Items {
		if pods.Items[i].DeletionTimestamp == nil {
			pod = &pods.Items[i]
			break
		}
	}
	require.NotNil(t, pod, "no app pod matches %q in %s", app.selector, ns)

	var initNames []string
	for _, c := range pod.Spec.InitContainers {
		initNames = append(initNames, c.Name)
	}
	for _, want := range tc.initContainers {
		assert.Contains(t, initNames, want, "pod %s: injected init container missing", pod.Name)
	}

	assert.NoError(t, containerHasEnv(pod, app.container, "OTEL_SERVICE_NAME", app.deployment), "pod %s", pod.Name)

	if tc.podCheck != nil {
		tc.podCheck(t, pod)
	}
	return pod
}

func containerHasEnv(pod *corev1.Pod, container, name, value string) error {
	for _, c := range pod.Spec.Containers {
		if c.Name != container {
			continue
		}
		for _, env := range c.Env {
			if env.Name == name {
				if env.Value != value {
					return fmt.Errorf("container %s: env %s = %q, want %q", container, name, env.Value, value)
				}
				return nil
			}
		}
		return fmt.Errorf("container %s: env %s not set", container, name)
	}
	return fmt.Errorf("container %s not found", container)
}

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"
	"text/template"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/yaml"
)

// diagnosticsLogTail is how many trailing log lines are dumped per container.
const diagnosticsLogTail = 150

// diagnosticsBanner is the banner used to delimit the diagnostic dump. Using a
// single delimiter line for the whole dump (rather than per item) keeps the output
// readable: it is emitted through a single t.Log call, so it gets one file:line
// prefix instead of one per line.
var diagnosticsBanner = "=" + strings.Repeat("-", 58) + "="

// diagnosticCRs are the custom resources DumpNamespaceOnFailure renders, so a failed
// test shows the exact CRs (including status) the operator acted on. A kind whose CRD
// is not installed just logs a line and is skipped, so suites that drive only some of
// these pay nothing for the others.
var diagnosticCRs = []schema.GroupVersionKind{
	{Group: "opentelemetry.io", Version: "v1beta1", Kind: "OpenTelemetryCollectorList"},
	{Group: "opentelemetry.io", Version: "v1alpha1", Kind: "InstrumentationList"},
	{Group: "opentelemetry.io", Version: "v1alpha1", Kind: "TargetAllocatorList"},
	{Group: "monitoring.coreos.com", Version: "v1", Kind: "PrometheusList"},
	{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitorList"},
}

// diagnosticsTemplate lays out the diagnostic dump: banner, section headings and the
// collected data. Everything is rendered into a single buffer and emitted with one
// t.Log call, so the file:line prefix appears once instead of once per line.
var diagnosticsTemplate = template.Must(template.New("namespace-diagnostics").Funcs(template.FuncMap{
	"join": strings.Join,
}).Parse(`{{.Banner}}
  DIAGNOSTICS FOR NAMESPACE {{.Namespace}}
{{.Banner}}
{{range .Warnings}}  WARNING: {{.}}
{{end}}
--- PODS AND CONTAINERS ---
{{if .Pods}}{{join .Pods "\n\n"}}{{else}}no pods in namespace{{end}}

--- EVENTS ---
{{join .Events "\n"}}

--- CUSTOM RESOURCES ---
{{if .Resources}}{{join .Resources "\n\n"}}{{else}}no {{.Namespace}} custom resources{{end}}

--- CONTAINER LOGS ---
{{if .Logs}}{{join .Logs "\n\n"}}{{else}}no pods in namespace{{end}}

{{.Banner}}
  END DIAGNOSTICS FOR NAMESPACE {{.Namespace}}
{{.Banner}}`))

// diagnosticsData is what diagnosticsTemplate renders. The fields are pre-formatted
// strings so the template stays a pure layout: it only joins blocks with headings.
type diagnosticsData struct {
	Banner    string
	Namespace string
	Warnings  []string
	Pods      []string
	Events    []string
	Resources []string
	Logs      []string
}

// DumpNamespaceOnFailure registers a cleanup that, if the test failed, logs a
// diagnostic snapshot of ns: pod and container statuses, recent events, the trailing
// logs of every container (init containers included), the OpenTelemetry CRs, and the
// operator's own trailing log. This is the Go replacement for chainsaw's `catch`
// blocks (podLogs/events) — a failed test must be diagnosable from its output alone.
func DumpNamespaceOnFailure(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string) {
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		// The test context may already be canceled when cleanups run.
		dumpNamespace(context.WithoutCancel(ctx), t, cfg, ns)
	})
}

func dumpNamespace(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string) {
	data := diagnosticsData{Banner: diagnosticsBanner, Namespace: ns}

	cs, err := clientSet(cfg)
	if err != nil {
		data.Warnings = append(data.Warnings, "clientset: "+err.Error())
		t.Log(renderDiagnostics(data))
		return
	}

	pods, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		data.Warnings = append(data.Warnings, "list pods: "+err.Error())
		t.Log(renderDiagnostics(data))
		return
	}
	for _, p := range pods.Items {
		data.Pods = append(data.Pods, strings.TrimSuffix(podStatus(p), "\n"))
	}

	events, err := cs.CoreV1().Events(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		data.Warnings = append(data.Warnings, "list events: "+err.Error())
	} else {
		slices.SortFunc(events.Items, func(a, b corev1.Event) int {
			return a.LastTimestamp.Compare(b.LastTimestamp.Time)
		})
		for _, e := range events.Items {
			data.Events = append(data.Events, fmt.Sprintf("%s %s %s/%s: %s (x%d)",
				e.Type, e.Reason, e.InvolvedObject.Kind, e.InvolvedObject.Name, e.Message, max(e.Count, 1)))
		}
	}

	for _, gvk := range diagnosticCRs {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)
		if err := CRClient(t, cfg).List(ctx, list, crclient.InNamespace(ns)); err != nil {
			data.Warnings = append(data.Warnings, fmt.Sprintf("list %s: %v", gvk.Kind, err))
			continue
		}
		for _, item := range list.Items {
			// managedFields are noise at this size.
			unstructured.RemoveNestedField(item.Object, "metadata", "managedFields")
			resource, err := yaml.Marshal(item.Object)
			if err != nil {
				data.Warnings = append(data.Warnings, fmt.Sprintf("marshal %s %s: %v", item.GetKind(), item.GetName(), err))
				continue
			}
			data.Resources = append(data.Resources, strings.TrimSuffix(fmt.Sprintf("%s %s:\n%s", item.GetKind(), item.GetName(), resource), "\n"))
		}
	}

	for _, p := range pods.Items {
		for _, container := range podContainers(p) {
			data.Logs = append(data.Logs, containerLog(ctx, cs, ns, p.Name, container))
		}
	}
	if operatorLog := operatorLog(ctx, cfg); operatorLog != "" {
		data.Logs = append(data.Logs, operatorLog)
	}
	t.Log(renderDiagnostics(data))
}

// renderDiagnostics renders the dump template with data.
func renderDiagnostics(data diagnosticsData) string {
	var b strings.Builder
	if err := diagnosticsTemplate.Execute(&b, data); err != nil {
		// The template is static; a render failure is a programming error.
		return fmt.Sprintf("diagnostics: render: %v", err)
	}
	return b.String()
}

// podContainers returns the init and regular container names of p, in run order.
func podContainers(p corev1.Pod) []string {
	containers := make([]string, 0, len(p.Spec.InitContainers)+len(p.Spec.Containers))
	for _, c := range p.Spec.InitContainers {
		containers = append(containers, c.Name)
	}
	for _, c := range p.Spec.Containers {
		containers = append(containers, c.Name)
	}
	return containers
}

// podStatus renders p's phase, readiness and per-container state for the dump.
func podStatus(p corev1.Pod) string {
	var b strings.Builder
	fmt.Fprintf(&b, "pod %s: phase=%s", p.Name, p.Status.Phase)
	for _, cond := range p.Status.Conditions {
		if cond.Type == corev1.PodReady {
			fmt.Fprintf(&b, " ready=%s", cond.Status)
		}
	}
	b.WriteByte('\n')
	for _, cst := range append(append([]corev1.ContainerStatus{}, p.Status.InitContainerStatuses...), p.Status.ContainerStatuses...) {
		fmt.Fprintf(&b, "  container %s: ready=%t restarts=%d image=%s state=%s\n",
			cst.Name, cst.Ready, cst.RestartCount, cst.Image, containerState(cst.State))
	}
	return b.String()
}

func containerState(s corev1.ContainerState) string {
	switch {
	case s.Running != nil:
		return "running"
	case s.Waiting != nil:
		return fmt.Sprintf("waiting(%s: %s)", s.Waiting.Reason, s.Waiting.Message)
	case s.Terminated != nil:
		return fmt.Sprintf("terminated(%s, exit %d)", s.Terminated.Reason, s.Terminated.ExitCode)
	default:
		return "unknown"
	}
}

// containerLog renders the trailing lines of one container's log for the dump.
func containerLog(ctx context.Context, cs *kubernetes.Clientset, ns, pod, container string) string {
	tail := int64(diagnosticsLogTail)
	stream, err := cs.CoreV1().Pods(ns).
		GetLogs(pod, &corev1.PodLogOptions{Container: container, TailLines: &tail}).
		Stream(ctx)
	if err != nil {
		return fmt.Sprintf("logs %s/%s (container %s): %v", ns, pod, container, err)
	}
	defer stream.Close()
	data, err := io.ReadAll(stream)
	if err != nil {
		return fmt.Sprintf("logs %s/%s (container %s): read: %v", ns, pod, container, err)
	}
	return strings.TrimSuffix(fmt.Sprintf("logs %s/%s (container %s, last %d lines):\n%s", ns, pod, container, tail, data), "\n")
}

// operatorLog renders the trailing log of the operator manager, if it is installed
// in the conventional namespace. The empty string means it is not installed.
func operatorLog(ctx context.Context, cfg *envconf.Config) string {
	const operatorNS = "opentelemetry-operator-system"
	cs, err := clientSet(cfg)
	if err != nil {
		return ""
	}
	pods, err := cs.CoreV1().Pods(operatorNS).List(ctx, metav1.ListOptions{
		LabelSelector: "control-plane=controller-manager",
	})
	if err != nil || len(pods.Items) == 0 {
		return ""
	}
	return containerLog(ctx, cs, operatorNS, pods.Items[0].Name, "manager")
}

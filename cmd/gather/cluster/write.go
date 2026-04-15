// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	cgocorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// resourceDir returns the omc-compatible directory path for the given resource.
//
// Namespace-scoped: <collectionDir>/namespaces/<namespace>/<group>/<plural>
// Cluster-scoped:   <collectionDir>/cluster-scoped-resources/<group>/<plural>
//
// The empty API group (core resources) is written as "core".
func resourceDir(collectionDir, namespace, group, plural string) string {
	if group == "" {
		group = "core"
	}
	if namespace == "" {
		return filepath.Join(collectionDir, "cluster-scoped-resources", group, plural)
	}
	return filepath.Join(collectionDir, "namespaces", namespace, group, plural)
}

// logOutputPath returns the omc-compatible path for a container log file.
// omc expects the container directory repeated twice:
// <collectionDir>/namespaces/<namespace>/pods/<podName>/<container>/<container>/logs/current.log.
func logOutputPath(collectionDir, namespace, podName, container string) string {
	return filepath.Join(collectionDir, "namespaces", namespace, "pods", podName, container, container, "logs", "current.log")
}

// kindToPlural maps each Kind collected by the gather tool to its correct
// Kubernetes REST plural. Kubernetes pluralization is irregular (e.g. Ingress →
// ingresses, NetworkPolicy → networkpolicies), so a lookup table is safer than
// a naive string-suffix heuristic. log.Fatalf on unknown kinds makes omissions
// visible immediately during development rather than silently writing to a wrong
// path that omc cannot find.
var kindToPlural = map[string]string{
	// core (v1)
	"ConfigMap":             "configmaps",
	"PersistentVolume":      "persistentvolumes",
	"PersistentVolumeClaim": "persistentvolumeclaims",
	"Pod":                   "pods",
	"Service":               "services",
	"ServiceAccount":        "serviceaccounts",
	// apps
	"DaemonSet":   "daemonsets",
	"Deployment":  "deployments",
	"StatefulSet": "statefulsets",
	// autoscaling
	"HorizontalPodAutoscaler": "horizontalpodautoscalers",
	// networking.k8s.io
	"Ingress":       "ingresses",
	"NetworkPolicy": "networkpolicies",
	// policy
	"PodDisruptionBudget": "poddisruptionbudgets",
	// rbac.authorization.k8s.io
	"ClusterRole":        "clusterroles",
	"ClusterRoleBinding": "clusterrolebindings",
	"Role":               "roles",
	"RoleBinding":        "rolebindings",
	// apiextensions.k8s.io
	"CustomResourceDefinition": "customresourcedefinitions",
	// operators.coreos.com / OLM
	"ClusterServiceVersion": "clusterserviceversions",
	"InstallPlan":           "installplans",
	"Operator":              "operators",
	"OperatorGroup":         "operatorgroups",
	"Subscription":          "subscriptions",
	// monitoring.coreos.com
	"PodMonitor":     "podmonitors",
	"ServiceMonitor": "servicemonitors",
	// route.openshift.io
	"Route": "routes",
	// opentelemetry.io
	"Instrumentation":        "instrumentations",
	"OpAMPBridge":            "opampbridges",
	"OpenTelemetryCollector": "opentelemetrycollectors",
	"TargetAllocator":        "targetallocators",
}

// pluralFor returns the lowercase plural resource name for the given Kind.
// It panics for unknown kinds so that omissions are caught immediately during
// development rather than silently producing paths that omc cannot parse.
func pluralFor(kind string) string {
	if plural, ok := kindToPlural[kind]; ok {
		return plural
	}
	log.Fatalf("pluralFor: unknown Kind %q — add it to kindToPlural in write.go", kind)
	return ""
}

// writeToFile serializes obj to a YAML file at the omc-compatible path under collectionDir.
// The GVK is looked up from scheme so that controller-runtime List items (which have empty
// TypeMeta) are written with the correct apiVersion/kind fields.
func writeToFile(collectionDir string, obj client.Object, scheme *runtime.Scheme) {
	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		log.Fatalf("Failed to get GVK for object %s: %v", obj.GetName(), err)
	}
	gvk := gvks[0]

	outDir := resourceDir(collectionDir, obj.GetNamespace(), gvk.Group, pluralFor(gvk.Kind))
	if err = os.MkdirAll(outDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create directory %s: %v", outDir, err)
	}

	path := filepath.Join(outDir, fmt.Sprintf("%s.yaml", obj.GetName()))
	outputFile, err := os.Create(path)
	if err != nil {
		log.Fatalf("Failed to create file %s: %v", path, err)
	}
	defer outputFile.Close()

	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		log.Fatalf("Error converting object to unstructured: %v", err)
	}

	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}
	unstructuredObj.SetGroupVersionKind(gvk)

	serializer := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)
	if err = serializer.Encode(unstructuredObj, outputFile); err != nil {
		log.Fatalf("Error encoding to YAML: %v", err)
	}
}

// writePodYAMLToLogDir writes the pod YAML into the log directory at
// pods/<podName>/<podName>.yaml. omc's "logs" command discovers pods from
// this path when the aggregated core/pods.yaml is absent.
func writePodYAMLToLogDir(collectionDir string, pod *corev1.Pod, scheme *runtime.Scheme) {
	outDir := filepath.Join(collectionDir, "namespaces", pod.Namespace, "pods", pod.Name)
	if err := os.MkdirAll(outDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create directory %s: %v", outDir, err)
	}

	path := filepath.Join(outDir, fmt.Sprintf("%s.yaml", pod.Name))
	outputFile, err := os.Create(path)
	if err != nil {
		log.Fatalf("Failed to create file %s: %v", path, err)
	}
	defer outputFile.Close()

	gvks, _, err := scheme.ObjectKinds(pod)
	if err != nil {
		log.Fatalf("Failed to get GVK for pod %s: %v", pod.Name, err)
	}

	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pod)
	if err != nil {
		log.Fatalf("Error converting pod to unstructured: %v", err)
	}

	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}
	unstructuredObj.SetGroupVersionKind(gvks[0])

	serializer := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)
	if err = serializer.Encode(unstructuredObj, outputFile); err != nil {
		log.Fatalf("Error encoding pod to YAML: %v", err)
	}
}

// writeLogToFile streams pod container logs to the omc-compatible path under collectionDir.
// The format is: namespaces/<namespace>/pods/<podName>/<container>/<container>/logs/current.log.
func writeLogToFile(collectionDir, namespace, podName, container string, p cgocorev1.PodInterface) {
	req := p.GetLogs(podName, &corev1.PodLogOptions{Container: container})
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		log.Fatalf("Error getting pod logs: %v\n", err)
		return
	}
	defer podLogs.Close()

	logPath := logOutputPath(collectionDir, namespace, podName, container)
	if err = os.MkdirAll(filepath.Dir(logPath), os.ModePerm); err != nil {
		log.Fatalln(err)
		return
	}

	outputFile, err := os.Create(logPath)
	if err != nil {
		log.Fatalf("Error creating log file: %v\n", err)
		return
	}
	defer outputFile.Close()

	if _, err = io.Copy(outputFile, podLogs); err != nil {
		log.Fatalf("Error copying logs to file: %v\n", err)
	}
}

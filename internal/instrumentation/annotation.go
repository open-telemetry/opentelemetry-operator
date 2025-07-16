// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// annotationInjectJava indicates whether java auto-instrumentation should be injected or not.
	// Possible values are "true", "false" or "<Instrumentation>" name.
	annotationInjectContainerName             = "instrumentation.opentelemetry.io/container-names"
	annotationInjectJava                      = "instrumentation.opentelemetry.io/inject-java"
	annotationInjectJavaContainersName        = "instrumentation.opentelemetry.io/java-container-names"
	annotationInjectNodeJS                    = "instrumentation.opentelemetry.io/inject-nodejs"
	annotationInjectNodeJSContainersName      = "instrumentation.opentelemetry.io/nodejs-container-names"
	annotationInjectPython                    = "instrumentation.opentelemetry.io/inject-python"
	annotationInjectPythonContainersName      = "instrumentation.opentelemetry.io/python-container-names"
	annotationPythonPlatform                  = "instrumentation.opentelemetry.io/otel-python-platform"
	annotationInjectDotNet                    = "instrumentation.opentelemetry.io/inject-dotnet"
	annotationDotNetRuntime                   = "instrumentation.opentelemetry.io/otel-dotnet-auto-runtime"
	annotationInjectDotnetContainersName      = "instrumentation.opentelemetry.io/dotnet-container-names"
	annotationInjectGo                        = "instrumentation.opentelemetry.io/inject-go"
	annotationInjectGoContainersName          = "instrumentation.opentelemetry.io/go-container-names"
	annotationGoExecPath                      = "instrumentation.opentelemetry.io/otel-go-auto-target-exe"
	annotationInjectSdk                       = "instrumentation.opentelemetry.io/inject-sdk"
	annotationInjectSdkContainersName         = "instrumentation.opentelemetry.io/sdk-container-names"
	annotationInjectApacheHttpd               = "instrumentation.opentelemetry.io/inject-apache-httpd"
	annotationInjectApacheHttpdContainersName = "instrumentation.opentelemetry.io/apache-httpd-container-names"
	annotationInjectNginx                     = "instrumentation.opentelemetry.io/inject-nginx"
	annotationInjectNginxContainersName       = "instrumentation.opentelemetry.io/inject-nginx-container-names"
)

func hasMatchingInjectAnnotation(annotations map[string]string, containerAnnotation string) (string, bool) {
	annotationPairs := map[string]string{
		annotationInjectJavaContainersName:        annotationInjectJava,
		annotationInjectNodeJSContainersName:      annotationInjectNodeJS,
		annotationInjectPythonContainersName:      annotationInjectPython,
		annotationInjectDotnetContainersName:      annotationInjectDotNet,
		annotationInjectGoContainersName:          annotationInjectGo,
		annotationInjectSdkContainersName:         annotationInjectSdk,
		annotationInjectApacheHttpdContainersName: annotationInjectApacheHttpd,
		annotationInjectNginxContainersName:       annotationInjectNginx,
	}

	injectAnnotation, exists := annotationPairs[containerAnnotation]
	if !exists {
		return "", false
	}

	_, found := annotations[injectAnnotation]
	return injectAnnotation, found
}

func allInstrumentationAnnotations() []string {
	return []string{
		annotationInjectJava,
		annotationInjectNodeJS,
		annotationInjectPython,
		annotationInjectDotNet,
		annotationInjectGo,
		annotationInjectSdk,
		annotationInjectApacheHttpd,
		annotationInjectNginx,
	}
}

func hasAnyInstrumentationAnnotation(annotations map[string]string) bool {
	if annotations == nil {
		return false
	}
	for _, annotation := range allInstrumentationAnnotations() {
		if val, exists := annotations[annotation]; exists && len(val) > 0 {
			return true
		}
	}
	return false
}

// annotationValue returns the effective annotation value, based on the annotations from the pod and namespace.
// Implementation of the unified precedence: if pod has any instrumentation annotation,
// namespace instrumentation annotations are ignored completely.
func annotationValue(ns metav1.ObjectMeta, pod metav1.ObjectMeta, annotation string) string {
	if injectAnnotation, isContainerNames := hasMatchingInjectAnnotation(pod.Annotations, annotation); isContainerNames {
		if _, injectExists := pod.Annotations[injectAnnotation]; injectExists {
			if val, exists := pod.Annotations[annotation]; exists && len(val) > 0 {
				return val
			}
		}
	}

	if injectAnnotation, isContainerNames := hasMatchingInjectAnnotation(ns.Annotations, annotation); isContainerNames {
		if _, injectExists := ns.Annotations[injectAnnotation]; injectExists {
			if val, exists := ns.Annotations[annotation]; exists && len(val) > 0 {
				return val
			}
		}
	}

	// Check if pod has any instrumentation annotations
	podHasInstrumentationAnnotations := hasAnyInstrumentationAnnotation(pod.Annotations)

	podAnnValue, podExists := pod.Annotations[annotation]
	nsAnnValue, nsExists := ns.Annotations[annotation]

	if podHasInstrumentationAnnotations {
		if podExists && len(podAnnValue) > 0 {
			return podAnnValue
		}
		return ""
	}

	if podExists && len(podAnnValue) > 0 {
		return podAnnValue
	}
	if nsExists && len(nsAnnValue) > 0 {
		return nsAnnValue
	}

	return ""
}

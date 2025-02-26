// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sidecar

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	// Annotation contains the annotation name that pods contain, indicating whether a sidecar is desired.
	Annotation = "sidecar.opentelemetry.io/inject"
)

// annotationValue returns the effective annotation value, based on the annotations from the pod and namespace.
func annotationValue(ns corev1.Namespace, pod corev1.Pod) string {
	// is the pod annotated with instructions to inject sidecars? is the namespace annotated?
	// if any of those is true, a sidecar might be desired.
	podAnnValue := pod.Annotations[Annotation]
	nsAnnValue := ns.Annotations[Annotation]

	// if the namespace value is empty, the pod annotation should be used, whatever it is
	if len(nsAnnValue) == 0 {
		return podAnnValue
	}

	// if the pod value is empty, the annotation annotation should be used (true, false, instance)
	if len(podAnnValue) == 0 {
		return nsAnnValue
	}

	// the pod annotation isn't empty -- if it's an instance name, or false, that's the decision
	if !strings.EqualFold(podAnnValue, "true") {
		return podAnnValue
	}

	// pod annotation is 'true', and if the namespace annotation is false, we just return 'true'
	if strings.EqualFold(nsAnnValue, "false") {
		return podAnnValue
	}

	// by now, the pod annotation is 'true', and the namespace annotation is either true or an instance name
	// so, the namespace annotation can be used
	return nsAnnValue
}

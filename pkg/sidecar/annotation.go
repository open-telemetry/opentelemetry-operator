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

package sidecar

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	// Annotation contains the annotation name that pods contain, indicating whether a sidecar is desired.
	Annotation = "sidecar.opentelemetry.io/inject"
)

// AnnotationValue returns the effective annotation value, based on the annotations from the pod and namespace.
func AnnotationValue(ns corev1.Namespace, pod corev1.Pod) string {
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

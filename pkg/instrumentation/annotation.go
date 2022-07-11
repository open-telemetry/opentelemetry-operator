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

package instrumentation

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// annotationInjectJava indicates whether java auto-instrumentation should be injected or not.
	// Possible values are "true", "false" or "<Instrumentation>" name.
	annotationInjectJava          = "instrumentation.opentelemetry.io/inject-java"
	annotationInjectNodeJS        = "instrumentation.opentelemetry.io/inject-nodejs"
	annotationInjectPython        = "instrumentation.opentelemetry.io/inject-python"
	annotationInjectDotNet        = "instrumentation.opentelemetry.io/inject-dotnet"
	annotationInjectContainerName = "instrumentation.opentelemetry.io/container-names"
)

// annotationValue returns the effective annotationInjectJava value, based on the annotations from the pod and namespace.
func annotationValue(ns metav1.ObjectMeta, pod metav1.ObjectMeta, annotation string) string {
	// is the pod annotated with instructions to inject sidecars? is the namespace annotated?
	// if any of those is true, a sidecar might be desired.
	podAnnValue := pod.Annotations[annotation]
	nsAnnValue := ns.Annotations[annotation]

	// if the namespace value is empty, the pod annotation should be used, whatever it is
	if len(nsAnnValue) == 0 {
		return podAnnValue
	}

	// if the pod value is empty, the annotation should be used (true, false, instance)
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

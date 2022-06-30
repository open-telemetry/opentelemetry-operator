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

	corev1 "k8s.io/api/core/v1"
)

// Calculate if we already inject InitContainers.
func IsInitContainerMissing(pod corev1.Pod) bool {
	for _, initContainer := range pod.Spec.InitContainers {
		if initContainer.Name == initContainerName {
			return false
		}
	}
	return true
}

// Check if opentelemetry-auto-instrumentation volume is on the list.
func IsOtAIVolumeMissing(volumeMounts []corev1.VolumeMount) bool {
	for _, volumeMount := range volumeMounts {
		if volumeMount.Name == volumeName {
			return false
		}
	}
	return true
}

// Check if EnvVar value contains instrumentation string.
func IsEnvVarValueInstrumentationMissing(envVar corev1.EnvVar, instrumentation string) bool {
	if strings.Contains(envVar.Value, instrumentation) {
		return false
	}
	return true
}

// Checks if Pod is already instrumented by checking Instrumentation InitContainer presence.
func IsAutoInstrumentationInjected(pod corev1.Pod) bool {
	for _, cont := range pod.Spec.InitContainers {
		if cont.Name == initContainerName {
			return true
		}
	}
	return false
}

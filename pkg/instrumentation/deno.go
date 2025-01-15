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
	"errors"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	envOtelDeno               = "OTEL_DENO"
	annotationDenoHasInjected = "instrumentation.opentelemetry.io/has-injected-deno"
)

func injectDenoSDK(denoSpec v1alpha1.Deno, pod corev1.Pod, index int) (corev1.Pod, error) {
	// caller checks if there is at least one container.
	container := &pod.Spec.Containers[index]

	idx := getIndexOfEnv(container.Env, envOtelDeno)
	if idx != -1 && container.Env[idx].Value != "true" {
		return pod, errors.New("the container already sets OTEL_DENO")
	}

	// inject Deno instrumentation spec env vars.
	for _, env := range denoSpec.Env {
		idx = getIndexOfEnv(container.Env, env.Name)
		if idx == -1 {
			container.Env = append(container.Env, env)
		}
	}

	idx = getIndexOfEnv(container.Env, envOtelDeno)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOtelDeno,
			Value: "true",
		})
	}

	pod.Annotations[annotationDenoHasInjected] = "true"

	return pod, nil
}

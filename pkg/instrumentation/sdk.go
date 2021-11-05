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
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/instrumentation/v1alpha1"
)

const (
	volumeName        = "opentelemetry-auto-instrumentation"
	initContainerName = "opentelemetry-auto-instrumentation"

	envOTELServiceName          = "OTEL_SERVICE_NAME"
	envOTELExporterOTLPEndpoint = "OTEL_EXPORTER_OTLP_ENDPOINT"
)

// inject a new sidecar container to the given pod, based on the given OpenTelemetryCollector.
func inject(logger logr.Logger, otelinst v1alpha1.Instrumentation, pod corev1.Pod, language string) corev1.Pod {
	if len(pod.Spec.Containers) < 1 {
		return pod
	}

	// inject only to the first container for now
	// in the future we can define an annotation to configure this
	pod = injectCommonSDKConfig(otelinst, pod)
	if language == "java" {
		pod = injectJavaagent(logger, otelinst.Spec.Java, pod)
	}
	if language == "nodejs" {
		pod = injectNodeJSSDK(logger, otelinst.Spec.NodeJS, pod)
	}
	return pod
}

func injectCommonSDKConfig(otelinst v1alpha1.Instrumentation, pod corev1.Pod) corev1.Pod {
	container := &pod.Spec.Containers[0]
	idx := getIndexOfEnv(container.Env, envOTELServiceName)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name: envOTELServiceName,
			// TODO use more meaningful service name - e.g. deployment name
			Value: container.Name,
		})
	}

	idx = getIndexOfEnv(container.Env, envOTELExporterOTLPEndpoint)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOTELExporterOTLPEndpoint,
			Value: otelinst.Spec.Endpoint,
		})
	}

	return pod
}

func getIndexOfEnv(envs []corev1.EnvVar, name string) int {
	for i := range envs {
		if envs[i].Name == name {
			return i
		}
	}
	return -1
}

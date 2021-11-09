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
	"fmt"
	"sort"
	"strings"
	"unsafe"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/instrumentation/v1alpha1"
)

const (
	volumeName        = "opentelemetry-auto-instrumentation"
	initContainerName = "opentelemetry-auto-instrumentation"

	envOTELServiceName          = "OTEL_SERVICE_NAME"
	envOTELExporterOTLPEndpoint = "OTEL_EXPORTER_OTLP_ENDPOINT"
	envOTELResourceAttrs        = "OTEL_RESOURCE_ATTRIBUTES"
	envOTELPropagators          = "OTEL_PROPAGATORS"
	envOTELTracesSampler        = "OTEL_TRACES_SAMPLER"
	envOTELTracesSamplerArg     = "OTEL_TRACES_SAMPLER_ARG"
)

// inject a new sidecar container to the given pod, based on the given OpenTelemetryCollector.
func inject(logger logr.Logger, otelinst v1alpha1.Instrumentation, ns corev1.Namespace, pod corev1.Pod) corev1.Pod {
	if len(pod.Spec.Containers) < 1 {
		return pod
	}

	// inject only to the first container for now
	// in the future we can define an annotation to configure this
	pod = injectCommonSDKConfig(otelinst, ns, pod)
	pod = injectJavaagent(logger, otelinst.Spec.Java, pod)
	return pod
}

func injectCommonSDKConfig(otelinst v1alpha1.Instrumentation, ns corev1.Namespace, pod corev1.Pod) corev1.Pod {
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
	idx = getIndexOfEnv(container.Env, envOTELResourceAttrs)
	resourceMap := createResourceMap(otelinst, ns, pod)
	resStr := resourceMapToStr(resourceMap)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOTELResourceAttrs,
			Value: resStr,
		})
	} else {
		if !strings.HasSuffix(container.Env[idx].Value, ",") {
			resStr = "," + resStr
		}
		container.Env[idx].Value += resStr
	}
	idx = getIndexOfEnv(container.Env, envOTELPropagators)
	if idx == -1 && len(otelinst.Spec.Propagators) > 0 {
		propagators := *(*[]string)((unsafe.Pointer(&otelinst.Spec.Propagators)))
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOTELPropagators,
			Value: strings.Join(propagators, ","),
		})
	}

	idx = getIndexOfEnv(container.Env, envOTELTracesSampler)
	// configure sampler only if it is configured in the CR
	if idx == -1 && otelinst.Spec.Sampler.Type != "" {
		idxSamplerArg := getIndexOfEnv(container.Env, envOTELTracesSamplerArg)
		if idxSamplerArg == -1 {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  envOTELTracesSampler,
				Value: string(otelinst.Spec.Sampler.Type),
			})
			if otelinst.Spec.Sampler.Argument != "" {
				container.Env = append(container.Env, corev1.EnvVar{
					Name:  envOTELTracesSamplerArg,
					Value: otelinst.Spec.Sampler.Argument,
				})
			}
		}
	}

	return pod
}

// createResourceMap creates resource attribute map.
// User defined attributes (in explicitly set env var) have higher precedence.
func createResourceMap(otelinst v1alpha1.Instrumentation, ns corev1.Namespace, pod corev1.Pod) map[string]string {
	// get existing resources env var and parse it into a map
	existingRes := map[string]bool{}
	existingResourceEnvIdx := getIndexOfEnv(pod.Spec.Containers[0].Env, envOTELResourceAttrs)
	if existingResourceEnvIdx > -1 {
		existingResArr := strings.Split(pod.Spec.Containers[0].Env[existingResourceEnvIdx].Value, ",")
		for _, kv := range existingResArr {
			keyValueArr := strings.Split(strings.TrimSpace(kv), "=")
			if len(keyValueArr) != 2 {
				continue
			}
			existingRes[keyValueArr[0]] = true
		}
	}

	res := map[string]string{}
	for k, v := range otelinst.Spec.ResourceAttributes {
		if !existingRes[k] {
			res[k] = v
		}
	}
	if !existingRes["k8s.namespace.name"] {
		res["k8s.namespace.name"] = ns.Name
	}
	if pod.Name != "" {
		// The pod name might be empty if the pod is created form deployment template
		if !existingRes["k8s.pod.name"] {
			res["k8s.pod.name"] = pod.Name
		}
	}
	if !existingRes["k8s.container.name"] {
		res["k8s.container.name"] = pod.Spec.Containers[0].Name
	}
	// TODO add more attributes once the parent object (deployment) is accessible here
	return res
}

func resourceMapToStr(res map[string]string) string {
	keys := make([]string, 0, len(res))
	for k := range res {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var str = ""
	for _, k := range keys {
		if str != "" {
			str += ","
		}
		str += fmt.Sprintf("%s=%s", k, res[k])
	}

	return str
}

func getIndexOfEnv(envs []corev1.EnvVar, name string) int {
	for i := range envs {
		if envs[i].Name == name {
			return i
		}
	}
	return -1
}

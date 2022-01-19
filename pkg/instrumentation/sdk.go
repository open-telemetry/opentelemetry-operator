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
	"context"
	"fmt"
	"sort"
	"strings"
	"unsafe"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
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

	envPodName  = "OTEL_RESOURCE_ATTRIBUTES_POD_NAME"
	envPodUID   = "OTEL_RESOURCE_ATTRIBUTES_POD_UID"
	envNodeName = "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME"
)

// inject a new sidecar container to the given pod, based on the given OpenTelemetryCollector.

type sdkInjector struct {
	logger logr.Logger
	client client.Client
}

func (i *sdkInjector) inject(ctx context.Context, insts languageInstrumentations, ns corev1.Namespace, pod corev1.Pod) corev1.Pod {
	if len(pod.Spec.Containers) < 1 {
		return pod
	}

	// inject only to the first container for now
	// in the future we can define an annotation to configure this
	if insts.Java != nil {
		otelinst := *insts.Java
		i.logger.V(1).Info("injecting java instrumentation into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)
		pod = injectJavaagent(i.logger, otelinst.Spec.Java, pod)
		pod = i.injectCommonCustomizedEnv(otelinst, ns, pod)
		pod = i.injectCommonSDKConfig(ctx, otelinst, ns, pod)
	}
	if insts.NodeJS != nil {
		otelinst := *insts.NodeJS
		i.logger.V(1).Info("injecting nodejs instrumentation into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)
		pod = injectNodeJSSDK(i.logger, otelinst.Spec.NodeJS, pod)
		pod = i.injectCommonCustomizedEnv(otelinst, ns, pod)
		pod = i.injectCommonSDKConfig(ctx, otelinst, ns, pod)
	}
	if insts.Python != nil {
		otelinst := *insts.Python
		i.logger.V(1).Info("injecting python instrumentation into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)
		pod = injectPythonSDK(i.logger, otelinst.Spec.Python, pod)
		pod = i.injectCommonCustomizedEnv(otelinst, ns, pod)
		pod = i.injectCommonSDKConfig(ctx, otelinst, ns, pod)
	}
	return pod
}

func (i *sdkInjector) injectCommonCustomizedEnv(otelinst v1alpha1.Instrumentation, ns corev1.Namespace, pod corev1.Pod) corev1.Pod {
	container := &pod.Spec.Containers[0]
	for _, env := range otelinst.Spec.Env {
		idx := getIndexOfEnv(container.Env, env.Name)
		if idx == -1 && len(env.Value) > 0 {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  env.Name,
				Value: env.Value,
			})
		}
	}
	return pod
}

func (i *sdkInjector) injectCommonSDKConfig(ctx context.Context, otelinst v1alpha1.Instrumentation, ns corev1.Namespace, pod corev1.Pod) corev1.Pod {
	container := &pod.Spec.Containers[0]
	resourceMap := i.createResourceMap(ctx, otelinst, ns, pod)
	idx := getIndexOfEnv(container.Env, envOTELServiceName)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOTELServiceName,
			Value: chooseServiceName(pod, resourceMap),
		})
	}
	if otelinst.Spec.Exporter.Endpoint != "" {
		idx = getIndexOfEnv(container.Env, envOTELExporterOTLPEndpoint)
		if idx == -1 {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  envOTELExporterOTLPEndpoint,
				Value: otelinst.Spec.Endpoint,
			})
		}
	}

	// Some attributes might be empty, we should get them via k8s downward API
	if resourceMap[string(semconv.K8SPodNameKey)] == "" {
		container.Env = append(container.Env, corev1.EnvVar{
			Name: envPodName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		})
		resourceMap[string(semconv.K8SPodNameKey)] = fmt.Sprintf("$(%s)", envPodName)
	}
	if otelinst.Spec.Resource.AddK8sUIDAttributes {
		if resourceMap[string(semconv.K8SPodUIDKey)] == "" {
			container.Env = append(container.Env, corev1.EnvVar{
				Name: envPodUID,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.uid",
					},
				},
			})
			resourceMap[string(semconv.K8SPodUIDKey)] = fmt.Sprintf("$(%s)", envPodUID)
		}
	}
	if resourceMap[string(semconv.K8SNodeNameKey)] == "" {
		container.Env = append(container.Env, corev1.EnvVar{
			Name: envNodeName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		})
		resourceMap[string(semconv.K8SNodeNameKey)] = fmt.Sprintf("$(%s)", envNodeName)
	}

	idx = getIndexOfEnv(container.Env, envOTELResourceAttrs)
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

func chooseServiceName(pod corev1.Pod, resources map[string]string) string {
	if name := resources[string(semconv.K8SDeploymentNameKey)]; name != "" {
		return name
	}
	if name := resources[string(semconv.K8SStatefulSetNameKey)]; name != "" {
		return name
	}
	if name := resources[string(semconv.K8SJobNameKey)]; name != "" {
		return name
	}
	if name := resources[string(semconv.K8SCronJobNameKey)]; name != "" {
		return name
	}
	if name := resources[string(semconv.K8SPodNameKey)]; name != "" {
		return name
	}
	return pod.Spec.Containers[0].Name
}

// createResourceMap creates resource attribute map.
// User defined attributes (in explicitly set env var) have higher precedence.
func (i *sdkInjector) createResourceMap(ctx context.Context, otelinst v1alpha1.Instrumentation, ns corev1.Namespace, pod corev1.Pod) map[string]string {
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
	for k, v := range otelinst.Spec.Resource.Attributes {
		if !existingRes[k] {
			res[k] = v
		}
	}

	k8sResources := map[attribute.Key]string{}
	k8sResources[semconv.K8SNamespaceNameKey] = ns.Name
	k8sResources[semconv.K8SContainerNameKey] = pod.Spec.Containers[0].Name
	// Some fields might be empty - node name, pod name
	// The pod name might be empty if the pod is created form deployment template
	k8sResources[semconv.K8SPodNameKey] = pod.Name
	k8sResources[semconv.K8SPodUIDKey] = string(pod.UID)
	k8sResources[semconv.K8SNodeNameKey] = pod.Spec.NodeName
	i.addParentResourceLabels(ctx, otelinst.Spec.Resource.AddK8sUIDAttributes, ns, pod.ObjectMeta, k8sResources)
	for k, v := range k8sResources {
		if !existingRes[string(k)] && v != "" {
			res[string(k)] = v
		}
	}
	return res
}

func (i *sdkInjector) addParentResourceLabels(ctx context.Context, uid bool, ns corev1.Namespace, objectMeta metav1.ObjectMeta, resources map[attribute.Key]string) {
	for _, owner := range objectMeta.OwnerReferences {
		switch strings.ToLower(owner.Kind) {
		case "replicaset":
			resources[semconv.K8SReplicaSetNameKey] = owner.Name
			if uid {
				resources[semconv.K8SReplicaSetUIDKey] = string(owner.UID)
			}
			// parent of ReplicaSet is e.g. Deployment which we are interested to know
			rs := appsv1.ReplicaSet{}
			// ignore the error. The object might not exist, the error is not important, getting labels is just the best effort
			//nolint:errcheck
			i.client.Get(ctx, types.NamespacedName{
				Namespace: ns.Name,
				Name:      owner.Name,
			}, &rs)
			i.addParentResourceLabels(ctx, uid, ns, rs.ObjectMeta, resources)
		case "deployment":
			resources[semconv.K8SDeploymentNameKey] = owner.Name
			if uid {
				resources[semconv.K8SDeploymentUIDKey] = string(owner.UID)
			}
		case "statefulset":
			resources[semconv.K8SStatefulSetNameKey] = owner.Name
			if uid {
				resources[semconv.K8SStatefulSetUIDKey] = string(owner.UID)
			}
		case "daemonset":
			resources[semconv.K8SDaemonSetNameKey] = owner.Name
			if uid {
				resources[semconv.K8SDaemonSetUIDKey] = string(owner.UID)
			}
		case "job":
			resources[semconv.K8SJobNameKey] = owner.Name
			if uid {
				resources[semconv.K8SJobUIDKey] = string(owner.UID)
			}
		case "cronjob":
			resources[semconv.K8SCronJobNameKey] = owner.Name
			if uid {
				resources[semconv.K8SCronJobUIDKey] = string(owner.UID)
			}
		}
	}
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

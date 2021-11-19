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

	// Kubernetes resource attributes are defined in
	// https://github.com/open-telemetry/opentelemetry-specification/blob/v1.8.0/specification/resource/semantic_conventions/k8s.md
	resourceK8sNsName          = "k8s.namespace.name"
	resourceK8sNodeName        = "k8s.node.name"
	resourceK8sPodName         = "k8s.pod.name"
	resourceK8sPodUID          = "k8s.pod.uid"
	resourceK8sContainerName   = "k8s.container.name"
	resourceK8sReplicaSetName  = "k8s.replicaset.name"
	resourceK8sReplicaSetUID   = "k8s.replicaset.uid"
	resourceK8sDeploymentName  = "k8s.deployment.name"
	resourceK8sDeploymentUID   = "k8s.deployment.uid"
	resourceK8sStatefulSetName = "k8s.statefulset.name"
	resourceK8sStatefulSetUID  = "k8s.statefulset.uid"
	resourceK8DaemonSetName    = "k8s.daemonset.name"
	resourceK8sDaemonSetUID    = "k8s.daemonset.uid"
	resourceK8sJobName         = "k8s.job.name"
	resourceK8sJobUID          = "k8s.job.uid"
	resourceK8sCronJobName     = "k8s.cronjob.name"
	resourceK8sCronJobUID      = "k8s.cronjob.uid"

	envOTELServiceName          = "OTEL_SERVICE_NAME"
	envOTELExporterOTLPEndpoint = "OTEL_EXPORTER_OTLP_ENDPOINT"
	envOTELResourceAttrs        = "OTEL_RESOURCE_ATTRIBUTES"
	envOTELPropagators          = "OTEL_PROPAGATORS"
	envOTELTracesSampler        = "OTEL_TRACES_SAMPLER"
	envOTELTracesSamplerArg     = "OTEL_TRACES_SAMPLER_ARG"
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
		pod = i.injectCommonSDKConfig(ctx, otelinst, ns, pod)
		pod = injectJavaagent(i.logger, otelinst.Spec.Java, pod)
	}
	if insts.NodeJS != nil {
		otelinst := *insts.NodeJS
		i.logger.V(1).Info("injecting nodejs instrumentation into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)
		pod = i.injectCommonSDKConfig(ctx, otelinst, ns, pod)
		pod = injectNodeJSSDK(i.logger, otelinst.Spec.NodeJS, pod)
	}
	if insts.Python != nil {
		otelinst := *insts.Python
		i.logger.V(1).Info("injecting python instrumentation into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)
		pod = i.injectCommonSDKConfig(ctx, otelinst, ns, pod)
		pod = injectPythonSDK(i.logger, otelinst.Spec.Python, pod)
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
	idx = getIndexOfEnv(container.Env, envOTELExporterOTLPEndpoint)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOTELExporterOTLPEndpoint,
			Value: otelinst.Spec.Endpoint,
		})
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
	if name := resources[resourceK8sDeploymentName]; name != "" {
		return name
	}
	if name := resources[resourceK8sStatefulSetName]; name != "" {
		return name
	}
	if name := resources[resourceK8sJobName]; name != "" {
		return name
	}
	if name := resources[resourceK8sCronJobName]; name != "" {
		return name
	}
	if name := resources[resourceK8sPodName]; name != "" {
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

	resources := map[string]string{}
	resources[resourceK8sNsName] = ns.Name
	resources[resourceK8sContainerName] = pod.Spec.Containers[0].Name
	// Some fields might be empty - node name, pod name
	// The pod name might be empty if the pod is created form deployment template
	resources[resourceK8sPodName] = pod.Name
	resources[resourceK8sPodUID] = string(pod.UID)
	resources[resourceK8sNodeName] = pod.Spec.NodeName
	i.getParentResourceLabels(ctx, otelinst.Spec.Resource.AddK8sUIDAttributes, ns, pod.ObjectMeta, resources)
	for k, v := range resources {
		if !existingRes[k] && v != "" {
			res[k] = v
		}
	}
	return res
}

func (i *sdkInjector) getParentResourceLabels(ctx context.Context, uid bool, ns corev1.Namespace, objectMeta metav1.ObjectMeta, resources map[string]string) {
	for _, owner := range objectMeta.OwnerReferences {
		switch strings.ToLower(owner.Kind) {
		case "replicaset":
			resources[resourceK8sReplicaSetName] = owner.Name
			if uid {
				resources[resourceK8sReplicaSetUID] = string(owner.UID)
			}
			// parent of ReplicaSet is e.g. Deployment which we are interested to know
			rs := appsv1.ReplicaSet{}
			// ignore the error. The object might not exist, the error is not important, getting labels is just the best effort
			//nolint:errcheck
			i.client.Get(ctx, types.NamespacedName{
				Namespace: ns.Name,
				Name:      owner.Name,
			}, &rs)
			i.getParentResourceLabels(ctx, uid, ns, rs.ObjectMeta, resources)
		case "deployment":
			resources[resourceK8sDeploymentName] = owner.Name
			if uid {
				resources[resourceK8sDeploymentUID] = string(owner.UID)
			}
		case "statefulset":
			resources[resourceK8sStatefulSetName] = owner.Name
			if uid {
				resources[resourceK8sStatefulSetUID] = string(owner.UID)
			}
		case "daemonset":
			resources[resourceK8DaemonSetName] = owner.Name
			if uid {
				resources[resourceK8sDaemonSetUID] = string(owner.UID)
			}
		case "job":
			resources[resourceK8sJobName] = owner.Name
			if uid {
				resources[resourceK8sJobUID] = string(owner.UID)
			}
		case "cronjob":
			resources[resourceK8sCronJobName] = owner.Name
			if uid {
				resources[resourceK8sCronJobUID] = string(owner.UID)
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

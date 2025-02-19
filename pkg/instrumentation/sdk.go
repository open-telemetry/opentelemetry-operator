// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unsafe"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

const (
	volumeName        = "opentelemetry-auto-instrumentation"
	initContainerName = "opentelemetry-auto-instrumentation"
	sideCarName       = "opentelemetry-auto-instrumentation"
)

// inject a new sidecar container to the given pod, based on the given OpenTelemetryCollector.

type sdkInjector struct {
	client client.Client
	logger logr.Logger
}

func (i *sdkInjector) inject(ctx context.Context, insts languageInstrumentations, ns corev1.Namespace, pod corev1.Pod, cfg config.Config) corev1.Pod {
	if len(pod.Spec.Containers) < 1 {
		return pod
	}
	if insts.Java.Instrumentation != nil {
		otelinst := *insts.Java.Instrumentation
		var err error
		i.logger.V(1).Info("injecting Java instrumentation into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)

		if len(insts.Java.Containers) == 0 {
			insts.Java.Containers = []string{pod.Spec.Containers[0].Name}
		}

		for _, container := range insts.Java.Containers {
			index := getContainerIndex(container, pod)
			pod, err = injectJavaagent(otelinst.Spec.Java, pod, index)
			if err != nil {
				i.logger.Info("Skipping javaagent injection", "reason", err.Error(), "container", pod.Spec.Containers[index].Name)
			} else {
				pod = i.injectCommonEnvVar(otelinst, pod, index)
				pod = i.injectCommonSDKConfig(ctx, otelinst, ns, pod, index, index)
				pod = i.setInitContainerSecurityContext(pod, pod.Spec.Containers[index].SecurityContext, javaInitContainerName)
			}
		}
	}
	if insts.NodeJS.Instrumentation != nil {
		otelinst := *insts.NodeJS.Instrumentation
		var err error
		i.logger.V(1).Info("injecting NodeJS instrumentation into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)

		if len(insts.NodeJS.Containers) == 0 {
			insts.NodeJS.Containers = []string{pod.Spec.Containers[0].Name}
		}

		for _, container := range insts.NodeJS.Containers {
			index := getContainerIndex(container, pod)
			pod, err = injectNodeJSSDK(otelinst.Spec.NodeJS, pod, index)
			if err != nil {
				i.logger.Info("Skipping NodeJS SDK injection", "reason", err.Error(), "container", pod.Spec.Containers[index].Name)
			} else {
				pod = i.injectCommonEnvVar(otelinst, pod, index)
				pod = i.injectCommonSDKConfig(ctx, otelinst, ns, pod, index, index)
				pod = i.setInitContainerSecurityContext(pod, pod.Spec.Containers[index].SecurityContext, nodejsInitContainerName)
			}
		}
	}
	if insts.Python.Instrumentation != nil {
		otelinst := *insts.Python.Instrumentation
		var err error
		i.logger.V(1).Info("injecting Python instrumentation into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)

		if len(insts.Python.Containers) == 0 {
			insts.Python.Containers = []string{pod.Spec.Containers[0].Name}
		}

		for _, container := range insts.Python.Containers {
			index := getContainerIndex(container, pod)
			pod, err = injectPythonSDK(otelinst.Spec.Python, pod, index, insts.Python.AdditionalAnnotations[annotationPythonPlatform])
			if err != nil {
				i.logger.Info("Skipping Python SDK injection", "reason", err.Error(), "container", pod.Spec.Containers[index].Name)
			} else {
				pod = i.injectCommonEnvVar(otelinst, pod, index)
				pod = i.injectCommonSDKConfig(ctx, otelinst, ns, pod, index, index)
				pod = i.setInitContainerSecurityContext(pod, pod.Spec.Containers[index].SecurityContext, pythonInitContainerName)
			}
		}
	}
	if insts.DotNet.Instrumentation != nil {
		otelinst := *insts.DotNet.Instrumentation
		var err error
		i.logger.V(1).Info("injecting DotNet instrumentation into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)

		if len(insts.DotNet.Containers) == 0 {
			insts.DotNet.Containers = []string{pod.Spec.Containers[0].Name}
		}

		for _, container := range insts.DotNet.Containers {
			index := getContainerIndex(container, pod)
			pod, err = injectDotNetSDK(otelinst.Spec.DotNet, pod, index, insts.DotNet.AdditionalAnnotations[annotationDotNetRuntime])
			if err != nil {
				i.logger.Info("Skipping DotNet SDK injection", "reason", err.Error(), "container", pod.Spec.Containers[index].Name)
			} else {
				pod = i.injectCommonEnvVar(otelinst, pod, index)
				pod = i.injectCommonSDKConfig(ctx, otelinst, ns, pod, index, index)
				pod = i.setInitContainerSecurityContext(pod, pod.Spec.Containers[index].SecurityContext, dotnetInitContainerName)
			}
		}
	}
	if insts.Go.Instrumentation != nil {
		origPod := pod
		otelinst := *insts.Go.Instrumentation
		var err error
		i.logger.V(1).Info("injecting Go instrumentation into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)

		if len(insts.Go.Containers) == 0 {
			insts.Go.Containers = []string{pod.Spec.Containers[0].Name}
		}

		// Go instrumentation supports only single container instrumentation.
		index := getContainerIndex(insts.Go.Containers[0], pod)
		pod, err = injectGoSDK(otelinst.Spec.Go, pod, cfg)
		if err != nil {
			i.logger.Info("Skipping Go SDK injection", "reason", err.Error(), "container", pod.Spec.Containers[index].Name)
		} else {
			// Common env vars and config need to be applied to the agent contain.
			pod = i.injectCommonEnvVar(otelinst, pod, len(pod.Spec.Containers)-1)
			pod = i.injectCommonSDKConfig(ctx, otelinst, ns, pod, len(pod.Spec.Containers)-1, 0)

			// Ensure that after all the env var coalescing we have a value for OTEL_GO_AUTO_TARGET_EXE
			idx := getIndexOfEnv(pod.Spec.Containers[len(pod.Spec.Containers)-1].Env, envOtelTargetExe)
			if idx == -1 {
				i.logger.Info("Skipping Go SDK injection", "reason", "OTEL_GO_AUTO_TARGET_EXE not set", "container", pod.Spec.Containers[index].Name)
				pod = origPod
			}
		}
	}
	if insts.ApacheHttpd.Instrumentation != nil {
		otelinst := *insts.ApacheHttpd.Instrumentation
		i.logger.V(1).Info("injecting Apache Httpd instrumentation into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)

		if len(insts.ApacheHttpd.Containers) == 0 {
			insts.ApacheHttpd.Containers = []string{pod.Spec.Containers[0].Name}
		}

		for _, container := range insts.ApacheHttpd.Containers {
			index := getContainerIndex(container, pod)
			// Apache agent is configured via config files rather than env vars.
			// Therefore, service name, otlp endpoint and other attributes are passed to the agent injection method
			useLabelsForResourceAttributes := otelinst.Spec.Defaults.UseLabelsForResourceAttributes
			pod = injectApacheHttpdagent(i.logger, otelinst.Spec.ApacheHttpd, pod, useLabelsForResourceAttributes, index, otelinst.Spec.Endpoint, i.createResourceMap(ctx, otelinst, ns, pod, index))
			pod = i.injectCommonEnvVar(otelinst, pod, index)
			pod = i.injectCommonSDKConfig(ctx, otelinst, ns, pod, index, index)
			pod = i.setInitContainerSecurityContext(pod, pod.Spec.Containers[index].SecurityContext, apacheAgentInitContainerName)
			pod = i.setInitContainerSecurityContext(pod, pod.Spec.Containers[index].SecurityContext, apacheAgentCloneContainerName)
		}
	}

	if insts.Nginx.Instrumentation != nil {
		otelinst := *insts.Nginx.Instrumentation
		i.logger.V(1).Info("injecting Nginx instrumentation into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)

		if len(insts.Nginx.Containers) == 0 {
			insts.Nginx.Containers = []string{pod.Spec.Containers[0].Name}
		}

		for _, container := range insts.Nginx.Containers {
			index := getContainerIndex(container, pod)
			// Nginx agent is configured via config files rather than env vars.
			// Therefore, service name, otlp endpoint and other attributes are passed to the agent injection method
			useLabelsForResourceAttributes := otelinst.Spec.Defaults.UseLabelsForResourceAttributes
			pod = injectNginxSDK(i.logger, otelinst.Spec.Nginx, pod, useLabelsForResourceAttributes, index, otelinst.Spec.Endpoint, i.createResourceMap(ctx, otelinst, ns, pod, index))
			pod = i.injectCommonEnvVar(otelinst, pod, index)
			pod = i.injectCommonSDKConfig(ctx, otelinst, ns, pod, index, index)
		}
	}

	if insts.Sdk.Instrumentation != nil {
		otelinst := *insts.Sdk.Instrumentation
		i.logger.V(1).Info("injecting sdk-only instrumentation into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)

		if len(insts.Sdk.Containers) == 0 {
			insts.Sdk.Containers = []string{pod.Spec.Containers[0].Name}
		}

		for _, container := range insts.Sdk.Containers {
			index := getContainerIndex(container, pod)
			pod = i.injectCommonEnvVar(otelinst, pod, index)
			pod = i.injectCommonSDKConfig(ctx, otelinst, ns, pod, index, index)
		}
	}

	return pod
}

func (i *sdkInjector) setInitContainerSecurityContext(pod corev1.Pod, securityContext *corev1.SecurityContext, instrInitContainerName string) corev1.Pod {
	for i, initContainer := range pod.Spec.InitContainers {
		if initContainer.Name == instrInitContainerName {
			pod.Spec.InitContainers[i].SecurityContext = securityContext
		}
	}

	return pod
}

func getContainerIndex(containerName string, pod corev1.Pod) int {
	// We search for specific container to inject variables and if no one is found
	// We fallback to first container
	var index = 0
	for idx, ctnair := range pod.Spec.Containers {
		if ctnair.Name == containerName {
			index = idx
		}
	}

	return index
}

func (i *sdkInjector) injectCommonEnvVar(otelinst v1alpha1.Instrumentation, pod corev1.Pod, index int) corev1.Pod {
	container := &pod.Spec.Containers[index]

	idx := getIndexOfEnv(container.Env, constants.EnvPodIP)
	if idx == -1 {
		container.Env = append([]corev1.EnvVar{{
			Name: constants.EnvPodIP,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		}}, container.Env...)
	}

	idx = getIndexOfEnv(container.Env, constants.EnvNodeIP)
	if idx == -1 {
		container.Env = append([]corev1.EnvVar{{
			Name: constants.EnvNodeIP,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.hostIP",
				},
			},
		}}, container.Env...)
	}

	for _, env := range otelinst.Spec.Env {
		idx := getIndexOfEnv(container.Env, env.Name)
		if idx == -1 {
			container.Env = append(container.Env, env)
		}
	}
	return pod
}

// injectCommonSDKConfig adds common SDK configuration environment variables to the necessary pod
// agentIndex represents the index of the pod the needs the env vars to instrument the application.
// appIndex represents the index of the pod the will produce the telemetry.
// When the pod handling the instrumentation is the same as the pod producing the telemetry agentIndex
// and appIndex should be the same value.  This is true for dotnet, java, nodejs, and python instrumentations.
// Go requires the agent to be a different container in the pod, so the agentIndex should represent this new sidecar
// and appIndex should represent the application being instrumented.
func (i *sdkInjector) injectCommonSDKConfig(ctx context.Context, otelinst v1alpha1.Instrumentation, ns corev1.Namespace, pod corev1.Pod, agentIndex int, appIndex int) corev1.Pod {
	container := &pod.Spec.Containers[agentIndex]
	useLabelsForResourceAttributes := otelinst.Spec.Defaults.UseLabelsForResourceAttributes
	resourceMap := i.createResourceMap(ctx, otelinst, ns, pod, appIndex)
	idx := getIndexOfEnv(container.Env, constants.EnvOTELServiceName)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  constants.EnvOTELServiceName,
			Value: chooseServiceName(pod, useLabelsForResourceAttributes, resourceMap, appIndex),
		})
	}
	configureExporter(otelinst.Spec.Exporter, &pod, container)

	// Always retrieve the pod name from the Downward API. Ensure that the OTEL_RESOURCE_ATTRIBUTES_POD_NAME env exists.
	container.Env = append(container.Env, corev1.EnvVar{
		Name: constants.EnvPodName,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	})
	resourceMap[string(semconv.K8SPodNameKey)] = fmt.Sprintf("$(%s)", constants.EnvPodName)

	// Some attributes might be empty, we should get them via k8s downward API
	if otelinst.Spec.Resource.AddK8sUIDAttributes {
		if resourceMap[string(semconv.K8SPodUIDKey)] == "" {
			container.Env = append(container.Env, corev1.EnvVar{
				Name: constants.EnvPodUID,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.uid",
					},
				},
			})
			resourceMap[string(semconv.K8SPodUIDKey)] = fmt.Sprintf("$(%s)", constants.EnvPodUID)
		}
	}

	idx = getIndexOfEnv(container.Env, constants.EnvOTELResourceAttrs)
	if idx == -1 || !strings.Contains(container.Env[idx].Value, string(semconv.ServiceVersionKey)) {
		vsn := chooseServiceVersion(pod, useLabelsForResourceAttributes, appIndex)
		if vsn != "" {
			resourceMap[string(semconv.ServiceVersionKey)] = vsn
		}
	}

	if resourceMap[string(semconv.K8SNodeNameKey)] == "" {
		container.Env = append(container.Env, corev1.EnvVar{
			Name: constants.EnvNodeName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		})
		resourceMap[string(semconv.K8SNodeNameKey)] = fmt.Sprintf("$(%s)", constants.EnvNodeName)
	}

	idx = getIndexOfEnv(container.Env, constants.EnvOTELResourceAttrs)
	resStr := resourceMapToStr(resourceMap)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  constants.EnvOTELResourceAttrs,
			Value: resStr,
		})
	} else {
		if !strings.HasSuffix(container.Env[idx].Value, ",") {
			resStr = "," + resStr
		}
		container.Env[idx].Value += resStr
	}

	idx = getIndexOfEnv(container.Env, constants.EnvOTELPropagators)
	if idx == -1 && len(otelinst.Spec.Propagators) > 0 {
		propagators := *(*[]string)((unsafe.Pointer(&otelinst.Spec.Propagators)))
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  constants.EnvOTELPropagators,
			Value: strings.Join(propagators, ","),
		})
	}

	idx = getIndexOfEnv(container.Env, constants.EnvOTELTracesSampler)
	// configure sampler only if it is configured in the CR
	if idx == -1 && otelinst.Spec.Sampler.Type != "" {
		idxSamplerArg := getIndexOfEnv(container.Env, constants.EnvOTELTracesSamplerArg)
		if idxSamplerArg == -1 {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  constants.EnvOTELTracesSampler,
				Value: string(otelinst.Spec.Sampler.Type),
			})
			if otelinst.Spec.Sampler.Argument != "" {
				container.Env = append(container.Env, corev1.EnvVar{
					Name:  constants.EnvOTELTracesSamplerArg,
					Value: otelinst.Spec.Sampler.Argument,
				})
			}
		}
	}

	// Move OTEL_RESOURCE_ATTRIBUTES to last position on env list.
	// When OTEL_RESOURCE_ATTRIBUTES environment variable uses other env vars
	// as attributes value they have to be configured before.
	// It is mandatory to set right order to avoid attributes with value
	// pointing to the name of used environment variable instead of its value.
	idx = getIndexOfEnv(container.Env, constants.EnvOTELResourceAttrs)
	envs := moveEnvToListEnd(container.Env, idx)
	container.Env = envs

	return pod
}

// chooseServiceName returns the service name to be used in the instrumentation.
// The precedence is as follows:
// 1. label or annotation with key "service.name" or "app.kubernetes.io/name".
// 2. k8s resource name (deployment, replicaset, statefulset, daemonset, cronjob, job, pod).
// 3. container name.
func chooseServiceName(pod corev1.Pod, useLabelsForResourceAttributes bool, resources map[string]string, index int) string {
	if name := chooseLabelOrAnnotation(pod, useLabelsForResourceAttributes, semconv.ServiceNameKey, constants.LabelAppName); name != "" {
		return name
	}
	if name := resources[string(semconv.K8SDeploymentNameKey)]; name != "" {
		return name
	}
	if name := resources[string(semconv.K8SReplicaSetNameKey)]; name != "" {
		return name
	}
	if name := resources[string(semconv.K8SStatefulSetNameKey)]; name != "" {
		return name
	}
	if name := resources[string(semconv.K8SDaemonSetNameKey)]; name != "" {
		return name
	}
	if name := resources[string(semconv.K8SCronJobNameKey)]; name != "" {
		return name
	}
	if name := resources[string(semconv.K8SJobNameKey)]; name != "" {
		return name
	}
	if name := resources[string(semconv.K8SPodNameKey)]; name != "" {
		return name
	}
	return pod.Spec.Containers[index].Name
}

// chooseLabelOrAnnotation returns the value of the label or annotation with the given key.
// The precedence is as follows:
// 1. annotation with key resource.opentelemetry.io/<resource>.
// 2. label with key labelKey.
func chooseLabelOrAnnotation(pod corev1.Pod, useLabelsForResourceAttributes bool, resource attribute.Key, labelKey string) string {
	if v := pod.GetAnnotations()[(constants.ResourceAttributeAnnotationPrefix + string(resource))]; v != "" {
		return v
	}
	if useLabelsForResourceAttributes {
		if v := pod.GetLabels()[labelKey]; v != "" {
			return v
		}
	}
	return ""
}

// chooseServiceVersion returns the service version to be used in the instrumentation.
// The precedence is as follows:
// 1. label or annotation with key "service.version" or "app.kubernetes.io/version".
// 2. image tag (by splitting image string on ":" and extracting final element from resulting array).
func chooseServiceVersion(pod corev1.Pod, useLabelsForResourceAttributes bool, index int) string {
	v := chooseLabelOrAnnotation(pod, useLabelsForResourceAttributes, semconv.ServiceVersionKey, constants.LabelAppVersion)
	if v != "" {
		return v
	}
	parts := strings.Split(pod.Spec.Containers[index].Image, ":")
	tag := parts[len(parts)-1]
	//guard statement to handle case where image name has a port number
	if strings.Contains(tag, "/") {
		return ""
	}
	return tag
}

// chooseServiceInstanceId returns the service.instance.id to be used in the instrumentation.
// The precedence is as follows:
//  1. annotation with key "service.instance.id"
//  2. namespace name + pod name + container name
//     (as defined by https://opentelemetry.io/docs/specs/semconv/resource/#service-experimental)
func createServiceInstanceId(pod corev1.Pod, namespaceName, podName, containerName string) string {
	// Do not use labels for service instance id,
	// because multiple containers in the same pod would get the same service instance id,
	// which violates the uniqueness requirement of service instance id -
	// see https://opentelemetry.io/docs/specs/semconv/resource/#service-experimental.
	// We still allow the user to set the service instance id via annotation, because this is explicitly set by the user.
	serviceInstanceId := chooseLabelOrAnnotation(pod, false, semconv.ServiceInstanceIDKey, "")
	if serviceInstanceId != "" {
		return serviceInstanceId
	}

	if namespaceName != "" && podName != "" && containerName != "" {
		resNames := []string{namespaceName, podName, containerName}
		serviceInstanceId = strings.Join(resNames, ".")
	}
	return serviceInstanceId
}

// createResourceMap creates resource attribute map.
// User defined attributes (in explicitly set env var) have higher precedence.
func (i *sdkInjector) createResourceMap(ctx context.Context, otelinst v1alpha1.Instrumentation, ns corev1.Namespace, pod corev1.Pod, index int) map[string]string {
	// get existing resources env var and parse it into a map
	existingRes := map[string]bool{}
	existingResourceEnvIdx := getIndexOfEnv(pod.Spec.Containers[index].Env, constants.EnvOTELResourceAttrs)
	if existingResourceEnvIdx > -1 {
		existingResArr := strings.Split(pod.Spec.Containers[index].Env[existingResourceEnvIdx].Value, ",")
		for _, kv := range existingResArr {
			keyValueArr := strings.Split(strings.TrimSpace(kv), "=")
			if len(keyValueArr) != 2 {
				continue
			}
			existingRes[keyValueArr[0]] = true
		}
	}

	res := map[string]string{}

	// entries from the CRD have the lowest precedence - they are overridden by later values
	for k, v := range otelinst.Spec.Resource.Attributes {
		if !existingRes[k] {
			res[k] = v
		}
	}

	useLabelsForResourceAttributes := otelinst.Spec.Defaults.UseLabelsForResourceAttributes

	// k8s resources have a higher precedence than CRD entries
	k8sResources := map[attribute.Key]string{}
	k8sResources[semconv.K8SNamespaceNameKey] = ns.Name
	k8sResources[semconv.K8SContainerNameKey] = pod.Spec.Containers[index].Name
	// Some fields might be empty - node name, pod name
	// The pod name might be empty if the pod is created form deployment template
	k8sResources[semconv.K8SPodNameKey] = pod.Name
	k8sResources[semconv.K8SPodUIDKey] = string(pod.UID)
	k8sResources[semconv.K8SNodeNameKey] = pod.Spec.NodeName
	k8sResources[semconv.ServiceInstanceIDKey] = createServiceInstanceId(pod, ns.Name, fmt.Sprintf("$(%s)", constants.EnvPodName), pod.Spec.Containers[index].Name)
	i.addParentResourceLabels(ctx, otelinst.Spec.Resource.AddK8sUIDAttributes, ns, pod.ObjectMeta, k8sResources)

	for k, v := range k8sResources {
		if !existingRes[string(k)] && v != "" {
			res[string(k)] = v
		}
	}

	// attributes and labels from the pod have the highest precedence (except for values set in environment variables)
	for k, v := range pod.GetAnnotations() {
		if strings.HasPrefix(k, constants.ResourceAttributeAnnotationPrefix) {
			key := strings.TrimPrefix(k, constants.ResourceAttributeAnnotationPrefix)
			if !existingRes[key] && key != string(semconv.ServiceNameKey) {
				res[key] = v
			}
		}
	}
	partOf := chooseLabelOrAnnotation(pod, useLabelsForResourceAttributes, semconv.ServiceNamespaceKey, constants.LabelAppPartOf)
	if partOf != "" && !existingRes[string(semconv.ServiceNamespaceKey)] {
		res[string(semconv.ServiceNamespaceKey)] = partOf
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
			nsn := types.NamespacedName{Namespace: ns.Name, Name: owner.Name}
			backOff := wait.Backoff{Duration: 10 * time.Millisecond, Factor: 1.5, Jitter: 0.1, Steps: 20, Cap: 2 * time.Second}

			checkError := func(err error) bool {
				return apierrors.IsNotFound(err)
			}

			getReplicaSet := func() error {
				return i.client.Get(ctx, nsn, &rs)
			}

			// use a retry loop to get the Deployment. A single call to client.get fails occasionally
			err := retry.OnError(backOff, checkError, getReplicaSet)
			if err != nil {
				i.logger.Error(err, "failed to get replicaset", "replicaset", nsn.Name, "namespace", nsn.Namespace)
			}
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

			// parent of Job can be CronJob which we are interested to know
			j := batchv1.Job{}
			nsn := types.NamespacedName{Namespace: ns.Name, Name: owner.Name}
			backOff := wait.Backoff{Duration: 10 * time.Millisecond, Factor: 1.5, Jitter: 0.1, Steps: 20, Cap: 2 * time.Second}

			checkError := func(err error) bool {
				return apierrors.IsNotFound(err)
			}

			getJob := func() error {
				return i.client.Get(ctx, nsn, &j)
			}

			// use a retry loop to get the Job. A single call to client.get fails occasionally
			err := retry.OnError(backOff, checkError, getJob)
			if err != nil {
				i.logger.Error(err, "failed to get job", "job", nsn.Name, "namespace", nsn.Namespace)
			}
			i.addParentResourceLabels(ctx, uid, ns, j.ObjectMeta, resources)
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

func moveEnvToListEnd(envs []corev1.EnvVar, idx int) []corev1.EnvVar {
	if idx >= 0 && idx < len(envs) {
		envToMove := envs[idx]
		envs = append(envs[:idx], envs[idx+1:]...)
		envs = append(envs, envToMove)
	}

	return envs
}

func validateContainerEnv(envs []corev1.EnvVar, envsToBeValidated ...string) error {
	for _, envToBeValidated := range envsToBeValidated {
		for _, containerEnv := range envs {
			if containerEnv.Name == envToBeValidated {
				if containerEnv.ValueFrom != nil {
					return fmt.Errorf("the container defines env var value via ValueFrom, envVar: %s", containerEnv.Name)
				}
				break
			}
		}
	}
	return nil
}

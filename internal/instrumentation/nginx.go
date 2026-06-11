// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-logr/logr"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	nginxDefaultConfigFile       = "/etc/nginx/nginx.conf"
	nginxAgentCloneContainerName = "otel-agent-source-container-clone"
	nginxAgentInitContainerName  = "otel-agent-attach-nginx"
	nginxAgentVolume             = "otel-nginx-agent"
	nginxAgentConfigVolume       = "otel-nginx-conf-dir"
	nginxAgentConfigFile         = "opentemetry_agent.conf"
	nginxAgentDirectory          = "/opt/opentelemetry-webserver"
	nginxAgentSubDirectory       = "/agent"
	nginxAgentDirFull            = nginxAgentDirectory + nginxAgentSubDirectory
	nginxAgentConfigDirectory    = "/source-conf"
	nginxAgentConfDirFull        = nginxAgentDirectory + nginxAgentConfigDirectory
	nginxAttributesEnvVar        = "OTEL_NGINX_AGENT_CONF"
	nginxServiceInstanceId       = "<<SID-PLACEHOLDER>>"
	nginxServiceInstanceIdEnvVar = "OTEL_NGINX_SERVICE_INSTANCE_ID"
	nginxLibraryPathEnv          = "LD_LIBRARY_PATH"
)

/*
	Nginx injection is different from other languages in:
	- OpenTelemetry parameters are not passed as environmental variables, but via a configuration file
	- OpenTelemetry module needs to be specified in the Nginx config file, but that is already specified by
	  an author of the original image configuration and the configuration must be preserved

	Therefore, following approach is taken:
	1) Inject an init container created as a *clone* of the application container and copy config file and referenced
	   configuration directory to an empty shared volume
	2) Inject a second init container with the OpenTelemetry module itself - i.e. instrumentation image
	3) Take the Nginx configuration file saved on volume and inject reference to OpenTelemetry module into the config
	4) On the same volume, inject a configuration file for OpenTelemetry module
	5) Copy OpenTelemetry module from second init container (instrumentation image) to another shared volume
	6) Inject mounting of volumes / files into appropriate directories in the application container
*/

func injectNginxSDK(_ logr.Logger, nginxSpec v1alpha1.Nginx, pod corev1.Pod, useLabelsForResourceAttributes bool, container *corev1.Container, otlpEndpoint string, resourceMap map[string]string, instSpec v1alpha1.InstrumentationSpec) corev1.Pod {
	// inject env vars
	container.Env = appendIfNotSet(container.Env, nginxSpec.Env...)

	// First make a clone of the instrumented container to take the existing Nginx configuration from
	// and create init container from it
	if isNginxInitContainerMissing(pod, nginxAgentCloneContainerName) {
		// Inject volume for original Nginx configuration
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: nginxAgentConfigVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})

		nginxConfDir := getNginxConfDir(nginxSpec.ConfigFile)

		// User-controlled value is passed as a positional arg (read as $1 in
		// the script) so it is never parsed by the shell.
		cloneContainer := corev1.Container{
			Name:    nginxAgentCloneContainerName,
			Image:   container.Image,
			Command: []string{"/bin/sh", "-c"},
			Args:    []string{nginxCloneScript, "--", nginxConfDir},
			Env:     container.Env,
			EnvFrom: container.EnvFrom,
			VolumeMounts: slices.Concat(container.VolumeMounts, []corev1.VolumeMount{{
				Name:      nginxAgentConfigVolume,
				MountPath: nginxAgentConfDirFull,
			}}),
			Resources:       nginxSpec.Resources,
			SecurityContext: resolveInitContainerSecurityContext(instSpec.InitContainerSecurityContext, container.SecurityContext),
			ImagePullPolicy: container.ImagePullPolicy,
		}

		pod.Spec.InitContainers = append(pod.Spec.InitContainers, cloneContainer)

		// drop volume mount with volume-provided Nginx config from original container
		// since it could over-write configuration provided by the injection
		idxFound := -1
		for idx, volume := range container.VolumeMounts {
			if strings.Contains(volume.MountPath, nginxConfDir) { // potentially passes config, which we want to pass to init copy only
				idxFound = idx
				break
			}
		}
		if idxFound >= 0 {
			volumeMounts := container.VolumeMounts
			volumeMounts = append(volumeMounts[:idxFound], volumeMounts[idxFound+1:]...)
			container.VolumeMounts = volumeMounts
		}

		// Inject volumes info instrumented container - Nginx config dir + Nginx agent
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      nginxAgentVolume,
			MountPath: nginxAgentDirFull,
		})
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      nginxAgentConfigVolume,
			MountPath: nginxConfDir,
		})
	}

	// Inject second init container with instrumentation image
	// Create / update config files
	// Copy OTEL module to a shared volume
	if isNginxInitContainerMissing(pod, nginxAgentInitContainerName) {
		// Inject volume for agent
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: nginxAgentVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})

		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:    nginxAgentInitContainerName,
			Image:   nginxSpec.Image,
			Command: []string{"/bin/sh", "-c"},
			// User-controlled value is passed as a positional arg (read as $1
			// in the script) so it is never parsed by the shell.
			Args: []string{nginxAgentScript, "--", getNginxConfFile(nginxSpec.ConfigFile)},
			Env: []corev1.EnvVar{
				{
					Name:  nginxAttributesEnvVar,
					Value: getNginxOtelConfig(pod, useLabelsForResourceAttributes, nginxSpec, container, otlpEndpoint, resourceMap),
				},
				{
					Name: nginxServiceInstanceIdEnvVar,
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.name",
						},
					},
				},
			},
			Resources: nginxSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      nginxAgentVolume,
					MountPath: nginxAgentDirFull,
				},
				{
					Name:      nginxAgentConfigVolume,
					MountPath: nginxAgentConfDirFull,
				},
			},
			SecurityContext: resolveInitContainerSecurityContext(instSpec.InitContainerSecurityContext, container.SecurityContext),
			ImagePullPolicy: instSpec.ImagePullPolicy,
		})

		found := false
		for i, e := range container.Env {
			if e.Name == nginxLibraryPathEnv {
				container.Env[i].Value = e.Value + ":" + nginxAgentDirFull + "/sdk_lib/lib"
				found = true
				break
			}
		}
		if !found {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  nginxLibraryPathEnv,
				Value: nginxAgentDirFull + "/sdk_lib/lib",
			})
		}
	}

	return pod
}

// Calculate if we already inject InitContainers.
func isNginxInitContainerMissing(pod corev1.Pod, containerName string) bool {
	for _, initContainer := range pod.Spec.InitContainers {
		if initContainer.Name == containerName {
			return false
		}
	}
	return true
}

// Calculate Nginx agent configuration file based on attributes provided by the injection rules
// and by the pod values.
func getNginxOtelConfig(pod corev1.Pod, useLabelsForResourceAttributes bool, nginxSpec v1alpha1.Nginx, container *corev1.Container, otelEndpoint string, resourceMap map[string]string) string {
	if otelEndpoint == "" {
		otelEndpoint = "http://localhost:4317/"
	}
	serviceName := chooseServiceName(pod, useLabelsForResourceAttributes, resourceMap, container)
	serviceNamespace := pod.GetNamespace()
	if serviceNamespace == "" {
		serviceNamespace = resourceMap[string(semconv.K8SNamespaceNameKey)]
		if serviceNamespace == "" {
			serviceNamespace = "nginx"
		}
	}

	// Namespace name override TBD

	attrMap := map[string]string{
		"NginxModuleEnabled":              "ON",
		"NginxModuleOtelSpanExporter":     "otlp",
		"NginxModuleOtelExporterEndpoint": otelEndpoint,
		"NginxModuleServiceName":          serviceName,
		"NginxModuleServiceNamespace":     serviceNamespace,
		"NginxModuleServiceInstanceId":    nginxServiceInstanceId,
		"NginxModuleResolveBackends":      "ON",
		"NginxModuleTraceAsError":         "ON",
	}
	for _, attr := range nginxSpec.Attrs {
		attrMap[attr.Name] = attr.Value
	}

	var configFileContent strings.Builder

	keys := make([]string, 0, len(attrMap))
	for key := range attrMap {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, key := range keys {
		fmt.Fprintf(&configFileContent, "%s %s;\n", key, attrMap[key])
	}

	return configFileContent.String()
}

func getNginxConfDir(configuredFile string) string {
	nginxConfFile := nginxDefaultConfigFile
	if configuredFile != "" {
		nginxConfFile = configuredFile
	}
	configDir := filepath.Dir(nginxConfFile)
	return configDir
}

func getNginxConfFile(configuredFile string) string {
	nginxConfFile := nginxDefaultConfigFile
	if configuredFile != "" {
		nginxConfFile = configuredFile
	}
	configFilenameOnly := filepath.Base(nginxConfFile)
	return configFilenameOnly
}

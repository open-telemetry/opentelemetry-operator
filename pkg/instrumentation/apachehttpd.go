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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	apacheConfigDirectory = "/usr/local/apache2/conf"
	apacheConfigFile      = "httpd.conf"
	apacheAgentConfigFile = "opentemetry_agent.conf"
	// apacheAgentDirectory          = "/otel-auto-instrumentation"
	apacheAgentDirectory          = "/opt/opentelemetry-webserver"
	apacheAgentSubDirectory       = "/agent"
	apacheAgentDirFull            = apacheAgentDirectory + apacheAgentSubDirectory
	apacheAgentConfigDirectory    = "/source-conf"
	apacheAgentConfDirFull        = apacheAgentDirectory + apacheAgentConfigDirectory
	apacheAgentInitContainerName  = "otel-agent-attach-apache"
	apacheAgentCloneContainerName = "otel-agent-source-container-clone"
	apacheAgentConfigVolume       = "otel-apache-conf-dir"
	apacheAgentVolume             = "otel-apache-agent"
	apacheAttributesEnvVar        = "OTEL_APACHE_AGENT_CONF"
	apacheServiceInstanceId       = "<<SID-PLACEHOLDER>>"
	apacheServiceInstanceIdEnvVar = "APACHE_SERVICE_INSTANCE_ID"
)

func injectApacheHttpdagent(logger logr.Logger, apacheSpec v1alpha1.ApacheHttpd, pod corev1.Pod, index int, otlpEndpoint string, resourceMap map[string]string) corev1.Pod {

	// caller checks if there is at least one container
	container := &pod.Spec.Containers[index]

	// inject env vars
	for _, env := range apacheSpec.Env {
		idx := getIndexOfEnv(container.Env, env.Name)
		if idx == -1 {
			container.Env = append(container.Env, env)
		}
	}

	// First make a clone of the instrumented container to take the existing Apache configuration from
	if isApacheInitContainerMissing(pod, apacheAgentCloneContainerName) {
		// Inject volume for original Apache configuration
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: apacheAgentConfigVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			}})

		cloneContainer := container.DeepCopy()
		cloneContainer.Name = apacheAgentCloneContainerName
		cloneContainer.Command = []string{"/bin/sh", "-c"}
		cloneContainer.Args = []string{"cp -r /usr/local/apache2/conf/* " + apacheAgentConfDirFull}
		cloneContainer.VolumeMounts = append(cloneContainer.VolumeMounts, corev1.VolumeMount{
			Name:      apacheAgentConfigVolume,
			MountPath: apacheAgentConfDirFull,
		})
		// remove resource requirements since those are then reserved for the lifetime of a pod
		// and we definitely do not need them for the init container for cp command
		cloneContainer.Resources = corev1.ResourceRequirements{}

		pod.Spec.InitContainers = append(pod.Spec.InitContainers, *cloneContainer)

		// drop volume mount with volume-provided Apache config from original container
		// since it could over-write configuration provided by the injection
		idxFound := -1
		for idx, volume := range container.VolumeMounts {
			if strings.Contains(volume.MountPath, apacheConfigDirectory) { // potentially passes config, which we want to pass to init copy only
				idxFound = idx
				break
			}
		}
		if idxFound >= 0 {
			volumeMounts := container.VolumeMounts
			volumeMounts = append(volumeMounts[:idxFound], volumeMounts[idxFound+1:]...)
			container.VolumeMounts = volumeMounts
		}

		// Inject volumes info instrumented container - Apache config dir + Apache agent
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      apacheAgentVolume,
			MountPath: apacheAgentDirFull,
		})
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      apacheAgentConfigVolume,
			MountPath: apacheConfigDirectory,
		})
	}

	// We just inject Volumes and init containers for the first processed container
	if isApacheInitContainerMissing(pod, apacheAgentInitContainerName) {
		// Inject volume for agent
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: apacheAgentVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			}})

		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:    apacheAgentInitContainerName,
			Image:   apacheSpec.Image,
			Command: []string{"/bin/sh", "-c"},
			Args: []string{
				// Copy agent binaries to shared volume
				"cp -ar /opt/opentelemetry/* " + apacheAgentDirFull + " && " +
					// setup logging configuration from template
					"export agentLogDir=$(echo \"" + apacheAgentDirFull + "/logs\" | sed 's,/,\\\\/,g') && " +
					"cat " + apacheAgentDirFull + "/conf/appdynamics_sdk_log4cxx.xml.template | sed 's/__agent_log_dir__/'${agentLogDir}'/g'  > " + apacheAgentDirFull + "/conf/appdynamics_sdk_log4cxx.xml &&" +
					// Create agent configuration file by pasting content of env var to a file
					"echo \"$" + apacheAttributesEnvVar + "\" > " + apacheAgentConfDirFull + "/" + apacheAgentConfigFile + " && " +
					"sed -i 's/" + apacheServiceInstanceId + "/'${" + apacheServiceInstanceIdEnvVar + "}'/g' " + apacheAgentConfDirFull + "/" + apacheAgentConfigFile + " && " +
					// Include a link to include Apache agent configuration file into httpd.conf
					"echo 'Include " + apacheConfigDirectory + "/" + apacheAgentConfigFile + "' >> " + apacheAgentConfDirFull + "/" + apacheConfigFile,
			},
			Env: []corev1.EnvVar{
				{
					Name:  apacheAttributesEnvVar,
					Value: getApacheOtelConfig(pod, apacheSpec, index, otlpEndpoint, resourceMap),
				},
				{Name: apacheServiceInstanceIdEnvVar,
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.name",
						},
					},
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      apacheAgentVolume,
					MountPath: apacheAgentDirFull,
				},
				{
					Name:      apacheAgentConfigVolume,
					MountPath: apacheAgentConfDirFull,
				},
			},
		})
	}

	return pod
}

// Calculate if we already inject InitContainers.
func isApacheInitContainerMissing(pod corev1.Pod, containerName string) bool {
	for _, initContainer := range pod.Spec.InitContainers {
		if initContainer.Name == containerName {
			return false
		}
	}
	return true
}

// Calculate Apache HTTPD agent configuration file based on attributes provided by the injection rules
// and by the pod values
func getApacheOtelConfig(pod corev1.Pod, apacheSpec v1alpha1.ApacheHttpd, index int, otelEndpoint string, resourceMap map[string]string) string {
	template := `
#Load the Otel Webserver SDK
LoadFile %[1]s/sdk_lib/lib/libopentelemetry_common.so
LoadFile %[1]s/sdk_lib/lib/libopentelemetry_resources.so
LoadFile %[1]s/sdk_lib/lib/libopentelemetry_trace.so
LoadFile %[1]s/sdk_lib/lib/libopentelemetry_otlp_recordable.so
LoadFile %[1]s/sdk_lib/lib/libopentelemetry_exporter_ostream_span.so
LoadFile %[1]s/sdk_lib/lib/libopentelemetry_exporter_otlp_grpc.so
#Load the Otel ApacheModule SDK
LoadFile %[1]s/sdk_lib/lib/libopentelemetry_webserver_sdk.so
#Load the Apache Module. In this example for Apache 2.4
#LoadModule otel_apache_module %[1]s/WebServerModule/Apache/libmod_apache_otel.so
#Load the Apache Module. In this example for Apache 2.2
#LoadModule otel_apache_module %[1]s/WebServerModule/Apache/libmod_apache_otel22.so
LoadModule otel_apache_module %[1]s/WebServerModule/Apache/libmod_apache_otel%[2]s.so
#Attributes
`
	if otelEndpoint == "" {
		otelEndpoint = "http://localhost:4317/"
	}
	serviceName := chooseServiceName(pod, resourceMap, index)
	serviceNamespace := pod.GetNamespace()
	if annotNamespace, found := pod.GetAnnotations()[annotationInjectOtelNamespace]; found {
		serviceNamespace = annotNamespace
	}

	versionSuffix := ""
	if apacheSpec.Version == "2.2" {
		versionSuffix = "22"
	}

	attrMap := map[string]string{
		"ApacheModuleEnabled": "ON",
		// ApacheModule Otel Exporter details
		"ApacheModuleOtelSpanExporter":     "otlp",
		"ApacheModuleOtelExporterEndpoint": otelEndpoint,
		// Service name and other IDs
		"ApacheModuleServiceName":       serviceName,
		"ApacheModuleServiceNamespace":  serviceNamespace,
		"ApacheModuleServiceInstanceId": apacheServiceInstanceId,

		"ApacheModuleResolveBackends": " ON",
		"ApacheModuleTraceAsError":    " ON",
	}
	for _, attr := range apacheSpec.Attrs {
		attrMap[attr.Name] = attr.Value
	}

	configFileContent := fmt.Sprintf(template,
		apacheAgentDirectory+apacheAgentSubDirectory,
		versionSuffix)

	keys := make([]string, 0, len(attrMap))
	for key := range attrMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		configFileContent += fmt.Sprintf("%s %s\n", key, attrMap[key])
	}

	return configFileContent
}

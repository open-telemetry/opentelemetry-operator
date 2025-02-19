// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	apacheDefaultConfigDirectory  = "/usr/local/apache2/conf"
	apacheConfigFile              = "httpd.conf"
	apacheAgentConfigFile         = "opentemetry_agent.conf"
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

/*
	Apache injection is different from other languages in:
	- OpenTelemetry parameters are not passed as environmental variables, but via a configuration file
	- OpenTelemetry module needs to be specified in the Apache HTTPD config file, but that is already specified by
	  an author of the application image and the configuration must be preserved

	Therefore, following approach is taken:
	1) Inject an init container created as a *clone* of the application container and copy config file to an empty shared volume
	2) Inject a second init container with the OpenTelemetry module itself - i.e. instrumentation image
	3) Take the Apache HTTPD configuration file saved on volume and inject reference to OpenTelemetry module into config
	4) Create on the same volume a configuration file for OpenTelemetry module
	5) Copy OpenTelemetry module from second init container (instrumentation image) to another shared volume
	6) Inject mounting of volumes / files into appropriate directories in application container
*/

func injectApacheHttpdagent(_ logr.Logger, apacheSpec v1alpha1.ApacheHttpd, pod corev1.Pod, useLabelsForResourceAttributes bool, index int, otlpEndpoint string, resourceMap map[string]string) corev1.Pod {

	volume := instrVolume(apacheSpec.VolumeClaimTemplate, apacheAgentVolume, apacheSpec.VolumeSizeLimit)

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
	// and create init container from it
	if isApacheInitContainerMissing(pod, apacheAgentCloneContainerName) {
		// Inject volume for original Apache configuration
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: apacheAgentConfigVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: volumeSize(apacheSpec.VolumeSizeLimit),
				},
			}})

		apacheConfDir := getApacheConfDir(apacheSpec.ConfigPath)

		cloneContainer := container.DeepCopy()
		cloneContainer.Name = apacheAgentCloneContainerName
		cloneContainer.Command = []string{"/bin/sh", "-c"}
		cloneContainer.Args = []string{"cp -r " + apacheConfDir + "/* " + apacheAgentConfDirFull}
		cloneContainer.VolumeMounts = append(cloneContainer.VolumeMounts, corev1.VolumeMount{
			Name:      apacheAgentConfigVolume,
			MountPath: apacheAgentConfDirFull,
		})
		// remove resource requirements since those are then reserved for the lifetime of a pod
		// and we definitely do not need them for the init container for cp command
		cloneContainer.Resources = apacheSpec.Resources
		// remove livenessProbe, readinessProbe, and startupProbe, since not supported on init containers
		cloneContainer.LivenessProbe = nil
		cloneContainer.ReadinessProbe = nil
		cloneContainer.StartupProbe = nil
		// remove lifecycle, since not supported on init containers
		cloneContainer.Lifecycle = nil

		pod.Spec.InitContainers = append(pod.Spec.InitContainers, *cloneContainer)

		// drop volume mount with volume-provided Apache config from original container
		// since it could over-write configuration provided by the injection
		idxFound := -1
		for idx, volume := range container.VolumeMounts {
			if strings.Contains(volume.MountPath, apacheConfDir) { // potentially passes config, which we want to pass to init copy only
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
			MountPath: apacheConfDir,
		})
	}

	// Inject second init container with instrumentation image
	// Create / update config files
	// Copy OTEL module to a shared volume
	if isApacheInitContainerMissing(pod, apacheAgentInitContainerName) {
		// Inject volume for agent
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:    apacheAgentInitContainerName,
			Image:   apacheSpec.Image,
			Command: []string{"/bin/sh", "-c"},
			Args: []string{
				// Copy agent binaries to shared volume
				"cp -r /opt/opentelemetry/* " + apacheAgentDirFull + " && " +
					// setup logging configuration from template
					"export agentLogDir=$(echo \"" + apacheAgentDirFull + "/logs\" | sed 's,/,\\\\/,g') && " +
					"cat " + apacheAgentDirFull + "/conf/opentelemetry_sdk_log4cxx.xml.template | sed 's/__agent_log_dir__/'${agentLogDir}'/g'  > " + apacheAgentDirFull + "/conf/opentelemetry_sdk_log4cxx.xml &&" +
					// Create agent configuration file by pasting content of env var to a file
					"echo \"$" + apacheAttributesEnvVar + "\" > " + apacheAgentConfDirFull + "/" + apacheAgentConfigFile + " && " +
					"sed -i 's/" + apacheServiceInstanceId + "/'${" + apacheServiceInstanceIdEnvVar + "}'/g' " + apacheAgentConfDirFull + "/" + apacheAgentConfigFile + " && " +
					// Include a link to include Apache agent configuration file into httpd.conf
					"echo -e '\nInclude " + getApacheConfDir(apacheSpec.ConfigPath) + "/" + apacheAgentConfigFile + "' >> " + apacheAgentConfDirFull + "/" + apacheConfigFile,
			},
			Env: []corev1.EnvVar{
				{
					Name:  apacheAttributesEnvVar,
					Value: getApacheOtelConfig(pod, useLabelsForResourceAttributes, apacheSpec, index, otlpEndpoint, resourceMap),
				},
				{Name: apacheServiceInstanceIdEnvVar,
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.name",
						},
					},
				},
			},
			Resources: apacheSpec.Resources,
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
// and by the pod values.
func getApacheOtelConfig(pod corev1.Pod, useLabelsForResourceAttributes bool, apacheSpec v1alpha1.ApacheHttpd, index int, otelEndpoint string, resourceMap map[string]string) string {
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
	serviceName := chooseServiceName(pod, useLabelsForResourceAttributes, resourceMap, index)
	serviceNamespace := pod.GetNamespace()
	if len(serviceNamespace) == 0 {
		serviceNamespace = resourceMap[string(semconv.K8SNamespaceNameKey)]
		if len(serviceNamespace) == 0 {
			serviceNamespace = "apache-httpd"
		}

	}
	// Namespace name override TBD

	// There are two versions of the OTEL modules - for Apache HTTPD 2.4 and 2.2.
	// 2.4 is default and the module does not have any version suffix
	// 2.2 has version suffix "22"
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

func getApacheConfDir(configuredDir string) string {
	apacheConfDir := apacheDefaultConfigDirectory
	if configuredDir != "" {
		apacheConfDir = configuredDir
		if apacheConfDir[len(apacheConfDir)-1] == '/' {
			apacheConfDir = apacheConfDir[:len(apacheConfDir)-1]
		}
	}
	return apacheConfDir
}

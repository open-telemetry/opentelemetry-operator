// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"
	"path/filepath"
	"sort"
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
	nginxVersionEnvVar           = "NGINX_VERSION"
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

func injectNginxSDK(_ logr.Logger, nginxSpec v1alpha1.Nginx, pod corev1.Pod, useLabelsForResourceAttributes bool, index int, otlpEndpoint string, resourceMap map[string]string) corev1.Pod {

	// caller checks if there is at least one container
	container := &pod.Spec.Containers[index]

	// inject env vars
	for _, env := range nginxSpec.Env {
		idx := getIndexOfEnv(container.Env, env.Name)
		if idx == -1 {
			container.Env = append(container.Env, env)
		}
	}

	// First make a clone of the instrumented container to take the existing Nginx configuration from
	// and create init container from it
	if isNginxInitContainerMissing(pod, nginxAgentCloneContainerName) {
		// Inject volume for original Nginx configuration
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: nginxAgentConfigVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			}})

		nginxConfFile := getNginxConfFile(nginxSpec.ConfigFile)
		nginxConfDir := getNginxConfDir(nginxSpec.ConfigFile)

		// from original Nginx container, we need
		// 1) original configuration files, which then get modified in the instrumentation process
		// 2) version of Nginx to select the proper version of OTel modules.
		//    - run Nginx with -v to get the version
		//    - store the version into a file where instrumentation initContainer can pick it up
		nginxCloneScriptTemplate :=
			`
cp -r %[2]s/* %[3]s &&
export %[4]s=$( { nginx -v ; } 2>&1 ) && echo ${%[4]s##*/} > %[3]s/version.txt
`
		nginxAgentCommands := prepareCommandFromTemplate(nginxCloneScriptTemplate,
			nginxConfFile,
			nginxConfDir,
			nginxAgentConfDirFull,
			nginxVersionEnvVar,
		)

		cloneContainer := container.DeepCopy()
		cloneContainer.Name = nginxAgentCloneContainerName
		cloneContainer.Command = []string{"/bin/sh", "-c"}
		cloneContainer.Args = []string{nginxAgentCommands}
		cloneContainer.VolumeMounts = append(cloneContainer.VolumeMounts, corev1.VolumeMount{
			Name:      nginxAgentConfigVolume,
			MountPath: nginxAgentConfDirFull,
		})
		// remove resource requirements since those are then reserved for the lifetime of a pod
		// and we definitely do not need them for the init container for cp command
		cloneContainer.Resources = nginxSpec.Resources
		// remove livenessProbe, readinessProbe, and startupProbe, since not supported on init containers
		cloneContainer.LivenessProbe = nil
		cloneContainer.ReadinessProbe = nil
		cloneContainer.StartupProbe = nil
		// remove lifecycle, since not supported on init containers
		cloneContainer.Lifecycle = nil

		pod.Spec.InitContainers = append(pod.Spec.InitContainers, *cloneContainer)

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
			}})

		// Following is the template for a shell script, which does the actual instrumentation
		// It does following:
		// 1) Copies Nginx OTel modules from the webserver agent image
		// 2) Picks-up the Nginx version stored by the clone of original container (see comment there)
		// 3) Finds out which directory to use for logs
		// 4) Configures the directory in logging configuration file of OTel modules
		// 5) Creates a configuration file for OTel modules
		// 6) In that configuration file, set SID parameter to pod name (in env var OTEL_NGINX_SERVICE_INSTANCE_ID)
		// 7) In Nginx config file, inject directive to load OTel module
		// 8) In Nginx config file, enable use of env var OTEL_RESOURCE_ATTRIBUTES in Nginx process
		//    (by default, env vars are hidden to Nginx process, they need to be enabled specifically)
		// 9) Move OTel module configuration file to Nginx configuration directory.

		// Each line of the script MUST end with \n !
		nginxAgentI13nScript :=
			`
NGINX_AGENT_DIR_FULL=$1	\n
NGINX_AGENT_CONF_DIR_FULL=$2 \n
NGINX_CONFIG_FILE=$3 \n
NGINX_SID_PLACEHOLDER=$4 \n
NGINX_SID_VALUE=$5 \n
echo "Input Parameters: $@" \n
set -x \n
\n
cp -r /opt/opentelemetry/* ${NGINX_AGENT_DIR_FULL} \n
\n
NGINX_VERSION=$(cat ${NGINX_AGENT_CONF_DIR_FULL}/version.txt) \n
NGINX_AGENT_LOG_DIR=$(echo "${NGINX_AGENT_DIR_FULL}/logs" | sed 's,/,\\/,g') \n
\n
cat ${NGINX_AGENT_DIR_FULL}/conf/opentelemetry_sdk_log4cxx.xml.template | sed 's,__agent_log_dir__,'${NGINX_AGENT_LOG_DIR}',g'  > ${NGINX_AGENT_DIR_FULL}/conf/opentelemetry_sdk_log4cxx.xml \n
echo -e $OTEL_NGINX_AGENT_CONF > ${NGINX_AGENT_CONF_DIR_FULL}/opentelemetry_agent.conf \n
sed -i "s,${NGINX_SID_PLACEHOLDER},${OTEL_NGINX_SERVICE_INSTANCE_ID},g" ${NGINX_AGENT_CONF_DIR_FULL}/opentelemetry_agent.conf \n
sed -i "1s,^,load_module ${NGINX_AGENT_DIR_FULL}/WebServerModule/Nginx/${NGINX_VERSION}/ngx_http_opentelemetry_module.so;\\n,g" ${NGINX_AGENT_CONF_DIR_FULL}/${NGINX_CONFIG_FILE} \n
sed -i "1s,^,env OTEL_RESOURCE_ATTRIBUTES;\\n,g" ${NGINX_AGENT_CONF_DIR_FULL}/${NGINX_CONFIG_FILE} \n
mv ${NGINX_AGENT_CONF_DIR_FULL}/opentelemetry_agent.conf  ${NGINX_AGENT_CONF_DIR_FULL}/conf.d \n
		`

		nginxAgentI13nCommand := "echo -e $OTEL_NGINX_I13N_SCRIPT > " + nginxAgentDirFull + "/nginx_instrumentation.sh && " +
			"chmod +x " + nginxAgentDirFull + "/nginx_instrumentation.sh && " +
			"cat " + nginxAgentDirFull + "/nginx_instrumentation.sh && " +
			fmt.Sprintf(nginxAgentDirFull+"/nginx_instrumentation.sh \"%s\" \"%s\" \"%s\" \"%s\"",
				nginxAgentDirFull,
				nginxAgentConfDirFull,
				getNginxConfFile(nginxSpec.ConfigFile),
				nginxServiceInstanceId,
			)

		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:    nginxAgentInitContainerName,
			Image:   nginxSpec.Image,
			Command: []string{"/bin/sh", "-c"},
			Args:    []string{nginxAgentI13nCommand},
			Env: []corev1.EnvVar{
				{
					Name:  nginxAttributesEnvVar,
					Value: getNginxOtelConfig(pod, useLabelsForResourceAttributes, nginxSpec, index, otlpEndpoint, resourceMap),
				},
				{
					Name:  "OTEL_NGINX_I13N_SCRIPT",
					Value: nginxAgentI13nScript,
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
			SecurityContext: pod.Spec.Containers[index].SecurityContext,
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
func getNginxOtelConfig(pod corev1.Pod, useLabelsForResourceAttributes bool, nginxSpec v1alpha1.Nginx, index int, otelEndpoint string, resourceMap map[string]string) string {

	if otelEndpoint == "" {
		otelEndpoint = "http://localhost:4317/"
	}
	serviceName := chooseServiceName(pod, useLabelsForResourceAttributes, resourceMap, index)
	serviceNamespace := pod.GetNamespace()
	if len(serviceNamespace) == 0 {
		serviceNamespace = resourceMap[string(semconv.K8SNamespaceNameKey)]
		if len(serviceNamespace) == 0 {
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

	configFileContent := ""

	keys := make([]string, 0, len(attrMap))
	for key := range attrMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		configFileContent += fmt.Sprintf("%s %s;\n", key, attrMap[key])
	}

	return configFileContent
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

func prepareCommandFromTemplate(template string, params ...any) string {
	command := fmt.Sprintf(template,
		params...,
	)

	command = strings.Replace(command, "\n", " ", -1)
	command = strings.Replace(command, "\t", " ", -1)
	command = strings.TrimLeft(command, " ")
	command = strings.TrimRight(command, " ")

	return command
}

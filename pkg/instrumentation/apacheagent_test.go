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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestInjectApacheagent(t *testing.T) {
	tests := []struct {
		name string
		v1alpha1.Apache
		pod      corev1.Pod
		expected corev1.Pod
	}{
		{
			name:   "Clone Container not present",
			Apache: v1alpha1.Apache{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: volumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    apacheAgentCloneContainerName,
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{"cp -r /usr/local/apache2/conf/* " + apacheAgentDirectory + apacheAgentConfigDirectory},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      apacheAgentConfigVolume,
								MountPath: apacheAgentDirectory + apacheAgentConfigDirectory,
							}},
						},
						{
							Name:    apacheAgentInitContainerName,
							Command: []string{"/bin/sh", "-c"},
							Args: []string{
								// Copy agent binaries to shared volume
								"cp -ar /opt/opentelemetry/* " + apacheAgentDirectory + apacheAgentSubDirectory + " && " +
									// Create agent configuration file by pasting content of env var to a file
									"echo \"$" + apacheAttributesEnvVar + "\" > " + apacheAgentDirectory + apacheAgentConfigDirectory + "/" + apacheAgentConfigFile + " && " +
									"sed -i 's/" + apacheServiceInstanceId + "/'${" + apacheServiceInstanceIdEnvVar + "}'/g' " + apacheAgentDirectory + apacheAgentConfigDirectory + "/" + apacheAgentConfigFile + " && " +
									// Include a link to include Apache agent configuration file into httpd.conf
									"echo 'Include " + apacheConfigDirectory + "/" + apacheAgentConfigFile + "' >> " + apacheAgentDirectory + apacheAgentConfigDirectory + "/" + apacheConfigFile,
							},
							Env: []corev1.EnvVar{
								{
									Name:  apacheAttributesEnvVar,
									Value: "",
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
									MountPath: apacheAgentDirectory + apacheAgentSubDirectory,
								},
								{
									Name:      apacheAgentConfigVolume,
									MountPath: apacheAgentDirectory + apacheAgentConfigDirectory,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      apacheAgentVolume,
									MountPath: apacheAgentDirectory + apacheAgentSubDirectory,
								},
								{
									Name:      apacheAgentConfigVolume,
									MountPath: apacheAgentDirectory + apacheAgentConfigDirectory,
								},
							},
						},
					},
				},
			},
		},
	}

	resourceMap := map[string]string{
		string(semconv.K8SDeploymentNameKey): "apache-service-name",
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pod := injectApacheagent(logr.Discard(), test.Apache, test.pod, 0, "http://otlp-endpoint:4317", resourceMap)
			assert.Equal(t, test.expected, pod)
		})
	}
}

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

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	envJavaToolsOptions   = "JAVA_TOOL_OPTIONS"
	javaAgent             = "-javaagent:/otel-auto-instrumentation-java/javaagent.jar"
	javaInitContainerName = initContainerName + "-java"
	javaVolumeName        = volumeName + "-java"
	javaInstrMountPath    = "/otel-auto-instrumentation-java"
)

func injectJavaagent(javaSpec v1alpha1.Java, pod corev1.Pod, container Container) (corev1.Pod, error) {
	volume := instrVolume(javaSpec.VolumeClaimTemplate, javaVolumeName, javaSpec.VolumeSizeLimit)

	if err := container.validate(&pod, envJavaToolsOptions); err != nil {
		return pod, err
	}

	// inject Java instrumentation spec env vars.
	for _, env := range javaSpec.Env {
		container.appendEnvVarIfNotExists(&pod, env)
	}

	javaJVMArgument := javaAgent
	if len(javaSpec.Extensions) > 0 {
		javaJVMArgument = javaAgent + fmt.Sprintf(" -Dotel.javaagent.extensions=%s/extensions", javaInstrMountPath)
	}

	if err := container.appendOrConcat(&pod, envJavaToolsOptions, javaJVMArgument, ConcatFunc(concatWithSpace)); err != nil {
		return pod, err
	}

	pod.Spec.Containers[container.index].VolumeMounts = append(pod.Spec.Containers[container.index].VolumeMounts, corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: javaInstrMountPath,
	})

	// We just inject Volumes and init containers for the first processed container.
	if isInitContainerMissing(pod, javaInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:      javaInitContainerName,
			Image:     javaSpec.Image,
			Command:   []string{"cp", "/javaagent.jar", javaInstrMountPath + "/javaagent.jar"},
			Resources: javaSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volume.Name,
				MountPath: javaInstrMountPath,
			}},
		})

		for i, extension := range javaSpec.Extensions {
			pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
				Name:      initContainerName + fmt.Sprintf("-extension-%d", i),
				Image:     extension.Image,
				Command:   []string{"cp", "-r", extension.Dir + "/.", javaInstrMountPath + "/extensions"},
				Resources: javaSpec.Resources,
				VolumeMounts: []corev1.VolumeMount{{
					Name:      volume.Name,
					MountPath: javaInstrMountPath,
				}},
			})
		}

	}
	return pod, nil
}

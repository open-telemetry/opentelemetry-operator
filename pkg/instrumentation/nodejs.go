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
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	envNodeOptions          = "NODE_OPTIONS"
	nodeRequireArgument     = "--require /otel-auto-instrumentation-nodejs/autoinstrumentation.js"
	nodejsInitContainerName = initContainerName + "-nodejs"
	nodejsVolumeName        = volumeName + "-nodejs"
	nodejsInstrMountPath    = "/otel-auto-instrumentation-nodejs"
)

func injectNodeJSSDK(nodeJSSpec v1alpha1.NodeJS, pod corev1.Pod, container Container) (corev1.Pod, error) {
	volume := instrVolume(nodeJSSpec.VolumeClaimTemplate, nodejsVolumeName, nodeJSSpec.VolumeSizeLimit)

	if err := container.validate(&pod, envNodeOptions); err != nil {
		return pod, err
	}

	// inject NodeJS instrumentation spec env vars.
	for _, env := range nodeJSSpec.Env {
		container.appendEnvVarIfNotExists(&pod, env)
	}

	if err := container.appendOrConcat(&pod, envNodeOptions, nodeRequireArgument, ConcatFunc(concatWithSpace)); err != nil {
		return pod, err
	}

	pod.Spec.Containers[container.index].VolumeMounts = append(pod.Spec.Containers[container.index].VolumeMounts, corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: nodejsInstrMountPath,
	})

	// We just inject Volumes and init containers for the first processed container
	if isInitContainerMissing(pod, nodejsInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:      nodejsInitContainerName,
			Image:     nodeJSSpec.Image,
			Command:   []string{"cp", "-r", "/autoinstrumentation/.", nodejsInstrMountPath},
			Resources: nodeJSSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volume.Name,
				MountPath: nodejsInstrMountPath,
			}},
		})
	}
	return pod, nil
}

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

package opampbridge

import (
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/proxy"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// Container builds a container for the given OpAMPBridge.
func Container(cfg config.Config, logger logr.Logger, opampBridge v1alpha1.OpAMPBridge) corev1.Container {
	image := opampBridge.Spec.Image
	if len(image) == 0 {
		image = cfg.OperatorOpAMPBridgeImage()
	}

	volumeMounts := []corev1.VolumeMount{{
		Name:      naming.OpAMPBridgeConfigMapVolume(),
		MountPath: "/conf",
	}}

	if len(opampBridge.Spec.VolumeMounts) > 0 {
		volumeMounts = append(volumeMounts, opampBridge.Spec.VolumeMounts...)
	}

	var envVars = opampBridge.Spec.Env
	if opampBridge.Spec.Env == nil {
		envVars = []corev1.EnvVar{}
	}

	idx := -1
	for i := range envVars {
		if envVars[i].Name == "OTELCOL_NAMESPACE" {
			idx = i
		}
	}
	if idx == -1 {
		envVars = append(envVars, corev1.EnvVar{
			Name: "OTELCOL_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		})
	}

	envVars = append(envVars, proxy.ReadProxyVarsFromEnv()...)

	return corev1.Container{
		Name:            naming.OpAMPBridgeContainer(),
		Image:           image,
		ImagePullPolicy: opampBridge.Spec.ImagePullPolicy,
		Env:             envVars,
		VolumeMounts:    volumeMounts,
		EnvFrom:         opampBridge.Spec.EnvFrom,
		Resources:       opampBridge.Spec.Resources,
		SecurityContext: opampBridge.Spec.SecurityContext,
	}
}

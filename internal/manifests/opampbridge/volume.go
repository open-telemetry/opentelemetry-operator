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
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// Volumes builds the volumes for the given instance, including the config map volume.
func Volumes(cfg config.Config, opampBridge v1alpha1.OpAMPBridge) []corev1.Volume {
	volumes := []corev1.Volume{{
		Name: naming.OpAMPBridgeConfigMapVolume(),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: naming.OpAMPBridgeConfigMap(opampBridge.Name)},
				Items: []corev1.KeyToPath{
					{
						Key:  cfg.OperatorOpAMPBridgeConfigMapEntry(),
						Path: cfg.OperatorOpAMPBridgeConfigMapEntry(),
					},
				},
			},
		},
	}}

	return volumes
}

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

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func TestInitContainers(t *testing.T) {
	params := deploymentParams()
	type args struct {
		flagState bool
		otelcol   v1alpha1.OpenTelemetryCollector
	}
	tests := []struct {
		name string
		args args
		want []corev1.Container
	}{
		{
			name: "flag disabled",
			args: args{
				flagState: false,
				otelcol:   params.Instance,
			},
			want: params.Instance.Spec.InitContainers,
		},
		{
			name: "flag enabled",
			args: args{
				flagState: true,
				otelcol:   params.Instance,
			},
			want: append(params.Instance.Spec.InitContainers, corev1.Container{
				Name:  "init-otc-container",
				Image: params.Instance.Spec.Image,
				Args:  []string{"validate", "--config=/conf/collector.yaml"},
				Env: []corev1.EnvVar{
					{
						Name: "POD_NAME",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "metadata.name",
							},
						},
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      naming.ConfigMapVolume(),
						MountPath: "/conf",
					},
				},
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.otelcol.Spec.RunValidation = tt.args.flagState
			assert.Equalf(t, tt.want, InitContainers(params.Config, params.Log, tt.args.otelcol), "InitContainers(%v)", tt.args.otelcol)
		})
	}
}

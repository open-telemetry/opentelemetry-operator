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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/api/convert"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func ConfigMap(params manifests.Params) (*corev1.ConfigMap, error) {
	otelCol, err := convert.V1Alpha1to2(params.OtelCol)
	if err != nil {
		return nil, err
	}
	name := naming.ConfigMap(otelCol.Name)
	labels := manifestutils.Labels(otelCol.ObjectMeta, name, otelCol.Spec.Image, ComponentOpenTelemetryCollector, []string{})

	replacedConf, err := ReplaceConfig(otelCol)
	if err != nil {
		params.Log.V(2).Info("failed to update prometheus config to use sharded targets: ", "err", err)
		return nil, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   otelCol.Namespace,
			Labels:      labels,
			Annotations: otelCol.Annotations,
		},
		Data: map[string]string{
			"collector.yaml": replacedConf,
		},
	}, nil
}

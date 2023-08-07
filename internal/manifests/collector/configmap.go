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
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func ConfigMap(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) *corev1.ConfigMap {
	name := naming.ConfigMap(otelcol.Name)
	labels := Labels(otelcol, name, []string{})

	replacedConf, err := ReplaceConfig(otelcol)
	if err != nil {
		logger.V(2).Info("failed to update prometheus config to use sharded targets: ", "err", err)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   otelcol.Namespace,
			Labels:      labels,
			Annotations: otelcol.Annotations,
		},
		Data: map[string]string{
			"collector.yaml": replacedConf,
		},
	}
}

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
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func InitContainers(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) []corev1.Container {
	if !otelcol.Spec.RunValidation {
		return otelcol.Spec.InitContainers
	}
	c := Container(cfg, logger, otelcol, true)
	c.Name = fmt.Sprintf("init-%s", c.Name)
	c.Args = append([]string{"validate"}, c.Args...)
	// Manually disable any unsupported init container fields
	c.LivenessProbe = nil
	c.ReadinessProbe = nil
	c.Lifecycle = nil
	c.StartupProbe = nil
	c.Ports = nil
	return append(otelcol.Spec.InitContainers, c)
}

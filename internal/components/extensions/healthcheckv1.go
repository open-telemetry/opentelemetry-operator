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

package extensions

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

const (
	DefaultHealthcheckV1Path = "/"
	DefaultHealthcheckV1Port = 13133
)

type healthcheckV1Config struct {
	components.SingleEndpointConfig `mapstructure:",squash"`
	Path                            string `mapstructure:"path"`
}

// HealthCheckV1Probe returns the probe configuration for the healthcheck v1 extension.
// Right now no TLS config is parsed.
func HealthCheckV1Probe(logger logr.Logger, config healthcheckV1Config) (*corev1.Probe, error) {
	path := config.Path
	if len(path) == 0 {
		path = DefaultHealthcheckV1Path
	}
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: path,
				Port: intstr.FromInt32(config.GetPortNumOrDefault(logger, DefaultHealthcheckV1Port)),
			},
		},
	}, nil
}

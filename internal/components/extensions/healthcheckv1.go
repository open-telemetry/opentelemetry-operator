// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

const (
	defaultHealthcheckV1Path = "/"
	defaultHealthcheckV1Port = 13133
)

type healthcheckV1Config struct {
	components.SingleEndpointConfig `mapstructure:",squash"`
	Path                            string `mapstructure:"path"`
}

// healthCheckV1Probe returns the probe configuration for the healthcheck v1 extension.
// Right now no TLS config is parsed.
func healthCheckV1Probe(logger logr.Logger, config healthcheckV1Config) (*corev1.Probe, error) {
	path := config.Path
	if len(path) == 0 {
		path = defaultHealthcheckV1Path
	}
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: path,
				Port: intstr.FromInt32(config.GetPortNumOrDefault(logger, defaultHealthcheckV1Port)),
			},
		},
	}, nil
}

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"fmt"
	"net"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
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

func healthCheckV1AddressDefaulter(logger logr.Logger, defaultRecAddr string, port int32, config healthcheckV1Config) (map[string]interface{}, error) {
	if config.Endpoint == "" {
		config.Endpoint = fmt.Sprintf("%s:%d", defaultRecAddr, port)
	} else {
		h, p, err := net.SplitHostPort(config.Endpoint)
		if err == nil && h == "" && p != "" {
			config.Endpoint = fmt.Sprintf("%s:%s", defaultRecAddr, p)
		}
	}

	if config.Path == "" {
		config.Path = defaultHealthcheckV1Path
	}

	res := make(map[string]interface{})
	err := mapstructure.Decode(config, &res)
	return res, err
}

// healthCheckV1Probe returns the probe configuration for the healthcheck v1 extension.
// Right now no TLS config is parsed.
func healthCheckV1Probe(logger logr.Logger, config healthcheckV1Config) (*corev1.Probe, error) {
	// These defaults shouldn't be needed if healthCheckV1AddressDefaulter is applied,
	// but since the function runs only when manifests are deployed,
	// we must keep these runtime defaults for backward compatibility.
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

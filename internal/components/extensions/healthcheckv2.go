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
	defaultHealthcheckV2Path       = "/"
	defaultHealthcheckV2StatusPath = "/status"
	defaultHealthcheckV2ConfigPath = "/config"
	defaultHealthcheckV2Port       = 13133
)

type healthcheckV2PathConfig struct {
	Enabled *bool  `mapstructure:"enabled,omitempty"`
	Path    string `mapstructure:"path,omitempty"`
}

type healthcheckV2HTTPConfig struct {
	components.SingleEndpointConfig `mapstructure:",squash"`
	Status                          healthcheckV2PathConfig `mapstructure:"status"`
	Config                          healthcheckV2PathConfig `mapstructure:"config"`
}

type healthcheckV2Config struct {
	components.SingleEndpointConfig `mapstructure:",squash"`
	Path                            string                   `mapstructure:"path,omitempty"`
	UseV2                           bool                     `mapstructure:"use_v2,omitempty"`
	HTTP                            *healthcheckV2HTTPConfig `mapstructure:"http,omitempty"`
}

func healthCheckV2AddressDefaulter(_ logr.Logger, defaultConfig *components.DefaultConfig, defaultRecAddr string, port int32, config healthcheckV2Config) (map[string]any, error) {
	if config.UseV2 {
		if config.HTTP != nil {
			if config.HTTP.Endpoint == "" {
				config.HTTP.Endpoint = fmt.Sprintf("%s:%d", defaultRecAddr, port)
			} else {
				h, p, err := net.SplitHostPort(config.HTTP.Endpoint)
				if err == nil && h == "" && p != "" {
					config.HTTP.Endpoint = fmt.Sprintf("%s:%s", defaultRecAddr, p)
				}
			}
			if config.HTTP.Status.Path == "" {
				config.HTTP.Status.Path = defaultHealthcheckV2StatusPath
			}
			if config.HTTP.Config.Path == "" {
				config.HTTP.Config.Path = defaultHealthcheckV2ConfigPath
			}
			config.HTTP.TLS.ApplyTLSProfileDefaults(defaultConfig.TLSProfile)
		}
		res := make(map[string]any)
		err := mapstructure.Decode(config, &res)
		return res, err
	}

	if config.Endpoint == "" {
		config.Endpoint = fmt.Sprintf("%s:%d", defaultRecAddr, port)
	} else {
		h, p, err := net.SplitHostPort(config.Endpoint)
		if err == nil && h == "" && p != "" {
			config.Endpoint = fmt.Sprintf("%s:%s", defaultRecAddr, p)
		}
	}

	if config.Path == "" {
		config.Path = defaultHealthcheckV2Path
	}

	config.TLS.ApplyTLSProfileDefaults(defaultConfig.TLSProfile)

	res := make(map[string]any)
	err := mapstructure.Decode(config, &res)
	return res, err
}

// healthCheckV2HTTPProbeTarget returns the path and port to probe on the v2 http
// service, or an empty path when no http endpoint is exposed.
func healthCheckV2HTTPProbeTarget(logger logr.Logger, http *healthcheckV2HTTPConfig) (path string, port int32) {
	port = http.GetPortNumOrDefault(logger, defaultHealthcheckV2Port)
	// Status.Enabled defaults to true when the http key is set.
	// See opentelemetry-collector-contrib/internal/healthcheck Config.Unmarshal.
	if http.Status.Enabled == nil || *http.Status.Enabled {
		path := http.Status.Path
		if path == "" {
			path = defaultHealthcheckV2StatusPath
		}
		return path, port
	}
	if http.Config.Enabled != nil && *http.Config.Enabled {
		path := http.Config.Path
		if path == "" {
			path = defaultHealthcheckV2ConfigPath
		}
		return path, port
	}
	return "", port
}

// healthCheckV2Probe returns the probe configuration for the healthcheck v2 extension.
func healthCheckV2Probe(logger logr.Logger, config healthcheckV2Config) (*corev1.Probe, error) {
	// These defaults shouldn't be needed if healthCheckV2AddressDefaulter is applied,
	// but since the function runs only when manifests are deployed,
	// we must keep these runtime defaults for backward compatibility.
	path := config.Path
	port := config.GetPortNumOrDefault(logger, defaultHealthcheckV2Port)
	if path == "" {
		path = defaultHealthcheckV2Path
	}
	if config.UseV2 {
		if config.HTTP == nil {
			// A v2 config without an http block serves no http endpoint to probe.
			return nil, nil
		}
		path, port = healthCheckV2HTTPProbeTarget(logger, config.HTTP)
		if path == "" {
			return nil, nil
		}
	}
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: path,
				Port: intstr.FromInt32(port),
			},
		},
	}, nil
}

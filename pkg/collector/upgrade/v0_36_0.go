// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
)

func upgrade0_36_0(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	if len(otelcol.Spec.Config) == 0 {
		return otelcol, nil
	}

	cfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.36.0, failed to parse configuration: %w", err)
	}

	// upgrading the receivers
	receivers, ok := cfg["receivers"].(map[interface{}]interface{})
	if !ok {
		// no receivers? no need to fail because of that
		return otelcol, nil
	}

	for k1, v1 := range receivers {
		// from the changelog https://github.com/open-telemetry/opentelemetry-collector/blob/main/CHANGELOG.md#-breaking-changes--2
		// Here is the upstream PR https://github.com/open-telemetry/opentelemetry-collector/pull/4063

		// Change tls config key from tls_settings to tls in otlp.protocols.grpc
		if strings.HasPrefix(k1.(string), "otlp") {
			otlpConfig, withOTLP := v1.(map[interface{}]interface{})
			if !withOTLP {
				// no otlpConfig? no need to fail because of that
				return otelcol, nil
			}
			for k2, v2 := range otlpConfig {
				// protocols config
				if k2 == "protocols" {
					protocConfig, withProtocConfig := v2.(map[interface{}]interface{})
					if !withProtocConfig {
						// no protocolConfig? no need to fail because of that
						return otelcol, nil
					}
					for k3, v3 := range protocConfig {
						// grpc config
						if k3 == "grpc" || k3 == "http" {
							grpcHTTPConfig, withHTTPConfig := v3.(map[interface{}]interface{})
							if !withHTTPConfig {
								// no grpcHTTPConfig? no need to fail because of that
								return otelcol, nil
							}
							for k4, v4 := range grpcHTTPConfig {
								// change tls_settings to tls
								if k4.(string) == "tls_settings" {
									grpcHTTPConfig["tls"] = v4
									delete(grpcHTTPConfig, "tls_settings")
									existing := &corev1.ConfigMap{}
									updated := existing.DeepCopy()
									u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.36.0 has changed the tls_settings field name to tls in %s protocol of %s receiver", k3, k1))
								}
							}
						}
					}
				}
			}
		}
	}
	cfg["receivers"] = receivers

	// upgrading the exporters
	exporters, ok := cfg["exporters"].(map[interface{}]interface{})
	if !ok {
		// no exporters? no need to fail because of that
		return otelcol, nil
	}

	for k1, v1 := range exporters {
		// from the changelog https://github.com/open-telemetry/opentelemetry-collector/blob/main/CHANGELOG.md#-breaking-changes--2
		// Here is the upstream PR https://github.com/open-telemetry/opentelemetry-collector/pull/4063

		// Move all tls config into separate field i,e, tls.*
		if strings.HasPrefix(k1.(string), "otlp") {
			otlpConfig, ok := v1.(map[interface{}]interface{})
			if !ok {
				// no otlpConfig? no need to fail because of that
				return otelcol, nil
			}
			tlsConfig := make(map[interface{}]interface{}, 5)
			for key, value := range otlpConfig {
				if key == "ca_file" || key == "cert_file" || key == "key_file" || key == "min_version" || key == "max_version" ||
					key == "insecure" || key == "insecure_skip_verify" || key == "server_name_override" {
					tlsConfig[key] = value
					delete(otlpConfig, key)
				}
				otlpConfig["tls"] = tlsConfig
				existing := &corev1.ConfigMap{}
				updated := existing.DeepCopy()
				u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.36.0 move tls config i.e. ca_file, key_file, cert_file, min_version, max_version to tls.* in %s exporter", k1))
			}
		}
	}
	cfg["exporters"] = exporters

	res, err := yaml.Marshal(cfg)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.36.0, failed to marshall back configuration: %w", err)
	}
	otelcol.Spec.Config = string(res)
	return otelcol, nil
}

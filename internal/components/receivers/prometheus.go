// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivers

import (
	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

var _ components.Parser = &PrometheusParser{}

// PrometheusParser handles the prometheus receiver, which has a unique config
// structure (config.scrape_configs[].tls_config) that doesn't fit the standard
// OTel tls: block pattern. It overrides GetDefaultConfig to walk the
// Prometheus-specific config.scrape_configs[].tls_config path.
type PrometheusParser struct {
	*ScraperParser
}

// NewPrometheusParser returns a parser for the prometheus receiver that applies
// TLS profile defaults to scrape_configs[].tls_config blocks.
func NewPrometheusParser() *PrometheusParser {
	return &PrometheusParser{
		ScraperParser: NewScraperParser("prometheus"),
	}
}

func (*PrometheusParser) GetDefaultConfig(_ logr.Logger, config any, opts ...components.DefaultOption) (any, error) {
	if config == nil {
		return nil, nil
	}
	configMap, ok := config.(map[string]any)
	if !ok {
		return config, nil
	}

	defaultCfg := &components.DefaultConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(defaultCfg)
		}
	}

	if defaultCfg.TLSProfile != nil {
		applyTLSProfileToScrapeConfigs(configMap, defaultCfg.TLSProfile)
	}

	return configMap, nil
}

// applyTLSProfileToScrapeConfigs walks config.scrape_configs and applies TLS profile
// defaults to each scrape config's tls_config block.
func applyTLSProfileToScrapeConfigs(config map[string]any, profile components.TLSProfile) {
	promConfig, ok := config["config"].(map[string]any)
	if !ok {
		return
	}

	scrapeConfigsList, ok := promConfig["scrape_configs"].([]any)
	if !ok {
		return
	}

	for _, sc := range scrapeConfigsList {
		scrapeConfig, ok := sc.(map[string]any)
		if !ok {
			continue
		}

		tlsConfig, ok := scrapeConfig["tls_config"].(map[string]any)
		if !ok {
			continue
		}

		if _, exists := tlsConfig["min_version"]; !exists {
			if minVersion := profile.MinTLSVersionPrometheus(); minVersion != "" {
				tlsConfig["min_version"] = minVersion
			}
		}
	}
}

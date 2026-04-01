// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivers

import (
	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

var _ components.Parser = &ScraperParser{}

// ScraperConfig represents the minimal config for scraper-style receivers.
// Only the tls: block is relevant for defaulting; other fields are preserved
// by the caller's mergo.Merge.
type ScraperConfig struct {
	TLS *components.TLSConfig `mapstructure:"tls,omitempty"`
}

// ScraperParser is a parser for scraper-style receivers (outbound HTTP clients).
// It applies TLS profile defaults to a top-level tls: block using OTel format
// (min_version: "1.2", cipher_suites: [...]).
type ScraperParser struct {
	*components.GenericParser[*ScraperConfig]
}

// NewScraperParser returns a scraper parser that applies TLS profile defaults
// to the top-level tls: block if present.
func NewScraperParser(name string) *ScraperParser {
	return &ScraperParser{
		GenericParser: components.NewBuilder[*ScraperConfig]().WithName(name).WithPort(components.UnsetPort).MustBuild(),
	}
}

func (*ScraperParser) GetDefaultConfig(_ logr.Logger, config any, opts ...components.DefaultOption) (any, error) {
	if config == nil {
		return nil, nil
	}

	defaultCfg := &components.DefaultConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(defaultCfg)
		}
	}

	var parsed ScraperConfig
	if err := mapstructure.Decode(config, &parsed); err != nil {
		return nil, err
	}

	conf, err := scraperDefaulter(defaultCfg, &parsed)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func scraperDefaulter(defaultCfg *components.DefaultConfig, config *ScraperConfig) (map[string]any, error) {
	if config == nil {
		config = &ScraperConfig{}
	}

	config.TLS.ApplyTLSProfileDefaults(defaultCfg.TLSProfile)

	res := make(map[string]any)
	err := mapstructure.Decode(config, &res)
	return res, err
}

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporters

import (
	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

var _ components.Parser = &ExporterParser{}

type ExporterConfig struct {
	TLS *components.TLSConfig `mapstructure:"tls,omitempty"`
}

// ExporterParser is a parser for push-based exporters (outbound connections).
// It applies TLS profile defaults to a top-level tls: block using OTel format
// (min_version: "1.2", cipher_suites: [...]).
type ExporterParser struct {
	*components.GenericParser[*ExporterConfig]
}

// NewExporterParser returns an exporter parser that applies TLS profile defaults
// to the top-level tls: block if present.
func NewExporterParser(name string) *ExporterParser {
	return &ExporterParser{
		GenericParser: components.NewBuilder[*ExporterConfig]().WithName(name).MustBuild(),
	}
}

func (*ExporterParser) GetDefaultConfig(_ logr.Logger, config any, opts ...components.DefaultOption) (any, error) {
	if config == nil {
		return nil, nil
	}

	defaultCfg := &components.DefaultConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(defaultCfg)
		}
	}

	var parsed ExporterConfig
	if err := mapstructure.Decode(config, &parsed); err != nil {
		return nil, err
	}

	conf, err := exporterDefaulter(defaultCfg, &parsed)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func exporterDefaulter(defaultCfg *components.DefaultConfig, config *ExporterConfig) (map[string]any, error) {
	if config == nil {
		config = &ExporterConfig{}
	}

	config.TLS.ApplyTLSProfileDefaults(defaultCfg.TLSProfile)

	res := make(map[string]any)
	err := mapstructure.Decode(config, &res)
	return res, err
}

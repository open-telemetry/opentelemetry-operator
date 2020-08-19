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

package config

import (
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/spf13/pflag"
)

// Config holds the static configuration for this operator
type Config struct {
	v                       version.Version
	collectorImage          string
	collectorConfigMapEntry string
}

// DefaultConfig builds a new configuration populated with the default, immutable values
func DefaultConfig() Config {
	return WithVersion(version.Get())
}

// WithVersion builds a new configuration using the provided version object
func WithVersion(v version.Version) Config {
	return Config{
		v:                       v,
		collectorConfigMapEntry: "collector.yaml",
	}
}

// FlagSet binds the flags to the user-modifiable values of the operator's configuration
func (c *Config) FlagSet() *pflag.FlagSet {
	fs := pflag.NewFlagSet("opentelemetry-operator", pflag.ExitOnError)
	pflag.StringVar(&c.collectorImage,
		"otelcol-image",
		fmt.Sprintf("quay.io/opentelemetry/opentelemetry-collector:v%s", c.v.OpenTelemetryCollector),
		"The default image to use for OpenTelemetry Collector when not specified in the individual custom resource (CR)",
	)

	return fs
}

// CollectorImage represents the flag to override the OpenTelemetry Collector container image.
func (c *Config) CollectorImage() string {
	return c.collectorImage
}

// CollectorConfigMapEntry represents the configuration file name for the collector. Immutable.
func (c *Config) CollectorConfigMapEntry() string {
	return c.collectorConfigMapEntry
}

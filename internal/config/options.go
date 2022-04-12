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
	"regexp"
	"strings"
	"time"

	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/pkg/platform"
)

// Option represents one specific configuration option.
type Option func(c *options)

type options struct {
	autoDetect                     autodetect.AutoDetect
	autoDetectFrequency            time.Duration
	targetAllocatorImage           string
	collectorImage                 string
	autoInstrumentationJavaImage   string
	autoInstrumentationNodeJSImage string
	autoInstrumentationPythonImage string
	autoInstrumentationDotNetImage string
	collectorConfigMapEntry        string
	targetAllocatorConfigMapEntry  string
	logger                         logr.Logger
	onChange                       []func() error
	platform                       platform.Platform
	version                        version.Version
	labelsFilter                   []string
}

func WithAutoDetect(a autodetect.AutoDetect) Option {
	return func(o *options) {
		o.autoDetect = a
	}
}
func WithAutoDetectFrequency(t time.Duration) Option {
	return func(o *options) {
		o.autoDetectFrequency = t
	}
}
func WithTargetAllocatorImage(s string) Option {
	return func(o *options) {
		o.targetAllocatorImage = s
	}
}

func WithCollectorImage(s string) Option {
	return func(o *options) {
		o.collectorImage = s
	}
}
func WithCollectorConfigMapEntry(s string) Option {
	return func(o *options) {
		o.collectorConfigMapEntry = s
	}
}
func WithTargetAllocatorConfigMapEntry(s string) Option {
	return func(o *options) {
		o.targetAllocatorConfigMapEntry = s
	}
}
func WithLogger(logger logr.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}
func WithOnChange(f func() error) Option {
	return func(o *options) {
		if o.onChange == nil {
			o.onChange = []func() error{}
		}
		o.onChange = append(o.onChange, f)
	}
}
func WithPlatform(plt platform.Platform) Option {
	return func(o *options) {
		o.platform = plt
	}
}
func WithVersion(v version.Version) Option {
	return func(o *options) {
		o.version = v
	}
}

func WithAutoInstrumentationJavaImage(s string) Option {
	return func(o *options) {
		o.autoInstrumentationJavaImage = s
	}
}

func WithAutoInstrumentationNodeJSImage(s string) Option {
	return func(o *options) {
		o.autoInstrumentationNodeJSImage = s
	}
}

func WithAutoInstrumentationPythonImage(s string) Option {
	return func(o *options) {
		o.autoInstrumentationPythonImage = s
	}
}

func WithAutoInstrumentationDotNetImage(s string) Option {
	return func(o *options) {
		o.autoInstrumentationDotNetImage = s
	}
}

func WithLabelFilters(labelFilters []string) Option {
	return func(o *options) {

		filters := []string{}
		for _, pattern := range labelFilters {
			var result strings.Builder

			for i, literal := range strings.Split(pattern, "*") {

				// Replace * with .*
				if i > 0 {
					result.WriteString(".*")
				}

				// Quote any regular expression meta characters in the
				// literal text.
				result.WriteString(regexp.QuoteMeta(literal))
			}
			filters = append(filters, result.String())
		}

		o.labelsFilter = filters
	}
}

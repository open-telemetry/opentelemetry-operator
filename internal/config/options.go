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
	"time"

	"github.com/go-logr/logr"

	"github.com/signalfx/splunk-otel-operator/internal/version"
	"github.com/signalfx/splunk-otel-operator/pkg/autodetect"
	"github.com/signalfx/splunk-otel-operator/pkg/platform"
)

// Option represents one specific configuration option.
type Option func(c *options)

type options struct {
	autoDetect                    autodetect.AutoDetect
	autoDetectFrequency           time.Duration
	targetAllocatorImage          string
	collectorImage                string
	collectorConfigMapEntry       string
	targetAllocatorConfigMapEntry string
	logger                        logr.Logger
	onChange                      []func() error
	platform                      platform.Platform
	version                       version.Version
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

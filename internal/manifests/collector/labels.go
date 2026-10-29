// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
)

// Labels returns the common labels to all objects that are part of a managed Collector CR.
// It resolves the collector image from the spec, falling back to the configured default collector image.
func Labels(instance v1beta1.OpenTelemetryCollector, name string, cfg config.Config, filterLabels []string) map[string]string {
	image := instance.Spec.Image
	if image == "" {
		image = cfg.CollectorImage
	}
	return manifestutils.Labels(instance.ObjectMeta, name, image, ComponentOpenTelemetryCollector, filterLabels)
}

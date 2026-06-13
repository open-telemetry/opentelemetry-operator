// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifestutils

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

// PrometheusAnnotationsAddedKey is the pod-template annotation the operator
// stamps when it adds the default prometheus.io/* scrape annotations. The
// mutate path uses its presence on an existing resource to decide whether the
// operator is the owner of the prometheus.io/* annotations and may therefore
// remove them when DisablePrometheusAnnotations is toggled to true. This
// avoids clobbering prometheus.io/* annotations that the user set out of band.
const PrometheusAnnotationsAddedKey = "collector.opentelemetry.io/prometheus-annotations-added"

// Annotations return the annotations for OpenTelemetryCollector resources.
func Annotations(instance v1beta1.OpenTelemetryCollector, filterAnnotations []string) (map[string]string, error) {
	// new map every time, so that we don't touch the instance's annotations
	annotations := map[string]string{}

	if instance.Annotations != nil {
		for k, v := range instance.Annotations {
			if !IsFilteredSet(k, filterAnnotations) {
				annotations[k] = v
			}
		}
	}

	return annotations, nil
}

// PodAnnotations return the spec annotations for OpenTelemetryCollector pod.
func PodAnnotations(instance v1beta1.OpenTelemetryCollector, filterAnnotations []string) (map[string]string, error) {
	// new map every time, so that we don't touch the instance's annotations
	podAnnotations := map[string]string{}
	if instance.Spec.PodAnnotations != nil {
		for k, v := range instance.Spec.PodAnnotations {
			if !IsFilteredSet(k, filterAnnotations) {
				podAnnotations[k] = v
			}
		}
	}

	annotations, err := Annotations(instance, filterAnnotations)
	if err != nil {
		return nil, err
	}
	// propagating annotations from metadata.annotations
	for kMeta, vMeta := range annotations {
		if _, found := podAnnotations[kMeta]; !found {
			podAnnotations[kMeta] = vMeta
		}
	}

	// Enable Prometheus annotations by default if DisablePrometheusAnnotations is nil or true
	if !instance.Spec.Observability.Metrics.DisablePrometheusAnnotations {
		// Set default Prometheus annotations
		prometheusAnnotations := map[string]string{
			"prometheus.io/scrape": "true",
			"prometheus.io/port":   "8888",
			"prometheus.io/path":   "/metrics",
		}
		// Default Prometheus annotations do not override existing
		stamped := false
		for kMeta, vMeta := range prometheusAnnotations {
			if _, ok := podAnnotations[kMeta]; !ok {
				podAnnotations[kMeta] = vMeta
				stamped = true
			}
		}
		// Stamp a marker only when the operator actually added at least one
		// default prometheus.io/* annotation. The mutate path uses the marker
		// to distinguish operator-stamped prom annotations from prom
		// annotations the user supplied out of band. Both are removed together
		// when DisablePrometheusAnnotations is toggled to true.
		if stamped {
			podAnnotations[PrometheusAnnotationsAddedKey] = "true"
		}
	}

	// make sure sha256 for configMap is always calculated
	hash, err := GetConfigMapSHA(instance.Spec.Config)
	if err != nil {
		return nil, err
	}

	// Adding the ConfigMap Hash only to PodAnnotations
	podAnnotations["opentelemetry-operator-config/sha256"] = hash

	return podAnnotations, nil
}

func GetConfigMapSHA(config v1beta1.Config) (string, error) {
	b, err := json.Marshal(&config)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h), nil
}

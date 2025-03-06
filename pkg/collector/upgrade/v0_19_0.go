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

func upgrade0_19_0(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	if len(otelcol.Spec.Config) == 0 {
		return otelcol, nil
	}

	cfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.19.0, failed to parse configuration: %w", err)
	}

	processors, ok := cfg["processors"].(map[interface{}]interface{})
	if !ok {
		// no processors? no need to fail because of that
		return otelcol, nil
	}

	for k, v := range processors {
		// from the changelog https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v0.19.0

		// Remove deprecated queued_retry processor
		if strings.HasPrefix(k.(string), "queued_retry") {
			delete(processors, k)
			existing := &corev1.ConfigMap{}
			updated := existing.DeepCopy()
			u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.19.0 removed the processor %q", k))
			continue
		}

		// Remove deprecated configs from resource processor: type (set "opencensus.type" key in "attributes.upsert" map instead) and labels (use "attributes.upsert" instead).
		if strings.HasPrefix(k.(string), "resource") {
			switch processor := v.(type) {
			case map[interface{}]interface{}:
				// type becomes an attribute.upsert with key opencensus.type
				if typ, found := processor["type"]; found {
					var attributes []map[string]string
					if attrs, found := processor["attributes"]; found {
						if attributes, ok = attrs.([]map[string]string); !ok {
							return otelcol, fmt.Errorf("couldn't upgrade to v0.19.0, the attributes list for processors %q couldn't be parsed based on the previous value. Type: %t, value: %v", k, attrs, attrs)
						}
					}
					attr := map[string]string{}
					attr["key"] = "opencensus.type"
					attr["value"] = typ.(string)
					attr["action"] = "upsert"
					attributes = append(attributes, attr)

					processor["attributes"] = attributes
					delete(processor, "type")
					existing := &corev1.ConfigMap{}
					updated := existing.DeepCopy()
					u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.19.0 migrated the property 'type' for processor %q", k))
				}

				// handle labels
				if labels, found := processor["labels"]; found {
					var attributes []map[string]string
					if attrs, found := processor["attributes"]; found {
						if attributes, ok = attrs.([]map[string]string); !ok {
							return otelcol, fmt.Errorf("couldn't upgrade to v0.19.0, the attributes list for processors %q couldn't be parsed based on the previous value. Type: %t, value: %v", k, attrs, attrs)
						}
					}

					if ls, ok := labels.(map[interface{}]interface{}); ok {
						for labelK, labelV := range ls {
							attr := map[string]string{}
							attr["key"] = labelK.(string)
							attr["value"] = labelV.(string)
							attr["action"] = "upsert"
							attributes = append(attributes, attr)
						}
					}

					processor["attributes"] = attributes
					delete(processor, "labels")
					existing := &corev1.ConfigMap{}
					updated := existing.DeepCopy()
					u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.19.0 migrated the property 'labels' for processor %q", k))
				}

				processors[k] = processor
			case string:
				if len(processor) == 0 {
					// this processor is using the default configuration
					continue
				}
			default:
				return otelcol, fmt.Errorf("couldn't upgrade to v0.19.0, the processor %q is invalid (neither a string nor map)", k)
			}
		}
	}

	cfg["processors"] = processors
	res, err := yaml.Marshal(cfg)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.19.0, failed to marshall back configuration: %w", err)
	}

	otelcol.Spec.Config = string(res)
	return otelcol, nil
}

package upgrade

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
)

func upgrade0_9_0(cl client.Client, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	if len(otelcol.Spec.Config) == 0 {
		return otelcol, nil
	}

	cfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.9.0, failed to parse configuration: %w", err)
	}

	exporters, ok := cfg["exporters"].(map[interface{}]interface{})
	if !ok {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.9.0, failed to extract list of exporters from the configuration: %q", cfg["exporters"])
	}

	for k, v := range exporters {
		if strings.HasPrefix("opencensus", k.(string)) {
			switch exporter := v.(type) {
			case map[interface{}]interface{}:
				// delete is a noop if there's no such entry
				delete(exporter, "reconnection_delay")
				exporters[k] = exporter
			case string:
				if len(exporter) == 0 {
					// this exporter is using the default configuration
					continue
				}
			default:
				return otelcol, fmt.Errorf("couldn't upgrade to v0.9.0, the exporter %q is invalid (neither a string nor map)", k)
			}
		}
	}

	cfg["exporters"] = exporters
	res, err := yaml.Marshal(cfg)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.9.0, failed to marshall back configuration: %w", err)
	}

	otelcol.Spec.Config = string(res)
	return otelcol, nil
}

package collector

import (
	"crypto/sha256"
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
)

// Annotations return the annotations for OpenTelemetryCollector pod.
func Annotations(instance v1alpha1.OpenTelemetryCollector) map[string]string {
	// new map every time, so that we don't touch the instance's annotations
	annotations := map[string]string{}
	if nil != instance.Annotations {
		for k, v := range instance.Annotations {
			annotations[k] = v
		}
	}

	annotations["prometheus.io/scrape"] = "true"
	annotations["prometheus.io/port"] = "8888"
	annotations["prometheus.io/path"] = "/metrics"
	annotations["opentelemetry-operator-config/sha256"] = getConfigMapSHA(instance.Spec.Config)

	return annotations
}

func getConfigMapSHA(config string) string {
	h := sha256.Sum256([]byte(config))
	return fmt.Sprintf("%x", h)
}

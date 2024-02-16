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

package collector

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

// Annotations return the annotations for OpenTelemetryCollector pod.
func Annotations(instance v1beta1.OpenTelemetryCollector) (map[string]string, error) {
	// new map every time, so that we don't touch the instance's annotations
	annotations := map[string]string{}

	// Enable Prometheus annotations by default if DisablePrometheusAnnotations is nil or true
	if !instance.Spec.Observability.Metrics.DisablePrometheusAnnotations {
		// Set default Prometheus annotations
		annotations["prometheus.io/scrape"] = "true"
		annotations["prometheus.io/port"] = "8888"
		annotations["prometheus.io/path"] = "/metrics"
	}

	// allow override of prometheus annotations
	if nil != instance.Annotations {
		for k, v := range instance.Annotations {
			annotations[k] = v
		}
	}

	hash, err := getConfigMapSHA(instance.Spec.Config)
	if err != nil {
		return nil, err
	}

	// make sure sha256 for configMap is always calculated
	annotations["opentelemetry-operator-config/sha256"] = hash

	return annotations, nil
}

// PodAnnotations return the spec annotations for OpenTelemetryCollector pod.
func PodAnnotations(instance v1beta1.OpenTelemetryCollector) (map[string]string, error) {
	// new map every time, so that we don't touch the instance's annotations
	podAnnotations := map[string]string{}

	// allow override of pod annotations
	for k, v := range instance.Spec.PodAnnotations {
		podAnnotations[k] = v
	}

	annotations, err := Annotations(instance)
	if err != nil {
		return nil, err
	}
	// propagating annotations from metadata.annotations
	for kMeta, vMeta := range annotations {
		if _, found := podAnnotations[kMeta]; !found {
			podAnnotations[kMeta] = vMeta
		}
	}

	hash, err := getConfigMapSHA(instance.Spec.Config)
	if err != nil {
		return nil, err
	}
	// make sure sha256 for configMap is always calculated
	podAnnotations["opentelemetry-operator-config/sha256"] = hash

	return podAnnotations, nil
}

func getConfigMapSHA(config v1beta1.Config) (string, error) {
	b, err := json.Marshal(&config)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h), nil
}

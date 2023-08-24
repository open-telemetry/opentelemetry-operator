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
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

// Annotations return the annotations for OpenTelemetryCollector pod.
func Annotations(instance v1alpha1.OpenTelemetryCollector) map[string]string {
	// new map every time, so that we don't touch the instance's annotations
	annotations := map[string]string{}

	// set default prometheus annotations
	annotations["prometheus.io/scrape"] = "true"
	annotations["prometheus.io/port"] = "8888"
	annotations["prometheus.io/path"] = "/metrics"

	// allow override of prometheus annotations
	if nil != instance.Annotations {
		for k, v := range instance.Annotations {
			annotations[k] = v
		}
	}
	// make sure sha256 for configMap is always calculated
	annotations["opentelemetry-operator-config/sha256"] = getConfigMapSHA(instance.Spec.Config)

	return annotations
}

// PodAnnotations return the spec annotations for OpenTelemetryCollector pod.
func PodAnnotations(instance v1alpha1.OpenTelemetryCollector) map[string]string {
	// new map every time, so that we don't touch the instance's annotations
	podAnnotations := map[string]string{}

	// allow override of pod annotations
	for k, v := range instance.Spec.PodAnnotations {
		podAnnotations[k] = v
	}

	// propagating annotations from metadata.annotations
	for kMeta, vMeta := range Annotations(instance) {
		if _, found := podAnnotations[kMeta]; !found {
			podAnnotations[kMeta] = vMeta
		}
	}

	// make sure sha256 for configMap is always calculated
	podAnnotations["opentelemetry-operator-config/sha256"] = getConfigMapSHA(instance.Spec.Config)

	return podAnnotations
}

func getConfigMapSHA(config string) string {
	h := sha256.Sum256([]byte(config))
	return fmt.Sprintf("%x", h)
}

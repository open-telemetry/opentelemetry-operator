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

	"github.com/signalfx/splunk-otel-operator/api/v1alpha1"
)

// Annotations return the annotations for SplunkOtelAgent pod.
func Annotations(instance v1alpha1.SplunkOtelAgent) map[string]string {
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
	annotations["splunk-otel-operator-config/sha256"] = getConfigMapSHA(instance.Spec.Config)

	return annotations
}

func getConfigMapSHA(config string) string {
	h := sha256.Sum256([]byte(config))
	return fmt.Sprintf("%x", h)
}

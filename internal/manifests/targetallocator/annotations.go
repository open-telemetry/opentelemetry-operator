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

package targetallocator

import (
	"crypto/sha256"
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const configMapHashAnnotationKey = "opentelemetry-targetallocator-config/hash"

// Annotations returns the annotations for the TargetAllocator Pod.
func Annotations(instance v1alpha1.OpenTelemetryCollector, configMap *v1.ConfigMap) map[string]string {
	// Make a copy of PodAnnotations to be safe
	annotations := make(map[string]string, len(instance.Spec.PodAnnotations))
	for key, value := range instance.Spec.PodAnnotations {
		annotations[key] = value
	}

	if configMap != nil {
		cmHash := getConfigMapSHA(configMap)
		if cmHash != "" {
			annotations[configMapHashAnnotationKey] = getConfigMapSHA(configMap)
		}
	}

	return annotations
}

// getConfigMapSHA returns the hash of the content of the TA ConfigMap.
func getConfigMapSHA(configMap *v1.ConfigMap) string {
	configString, ok := configMap.Data[targetAllocatorFilename]
	if !ok {
		return ""
	}
	h := sha256.Sum256([]byte(configString))
	return fmt.Sprintf("%x", h)
}

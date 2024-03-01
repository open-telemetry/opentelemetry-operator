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

package opampbridge

import (
	"crypto/sha256"
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
)

const configMapHashAnnotationKey = "opentelemetry-opampbridge-config/hash"

// Annotations returns the annotations for the OPAmpBridge Pod.
func Annotations(instance v1alpha1.OpAMPBridge, configMap *v1.ConfigMap, filterAnnotations []string) map[string]string {
	// Make a copy of PodAnnotations to be safe
	annotations := make(map[string]string, len(instance.Spec.PodAnnotations))
	for key, value := range instance.Spec.PodAnnotations {
		annotations[key] = value
	}
	if nil != instance.ObjectMeta.Annotations {
		for k, v := range instance.ObjectMeta.Annotations {
			if !manifestutils.IsFilteredSet(k, filterAnnotations) {
				annotations[k] = v
			}
		}
	}
	if configMap != nil {
		cmHash := getConfigMapSHA(configMap)
		if cmHash != "" {
			annotations[configMapHashAnnotationKey] = getConfigMapSHA(configMap)
		}
	}

	return annotations
}

// getConfigMapSHA returns the hash of the content of the OpAMPBridge ConfigMap.
func getConfigMapSHA(configMap *v1.ConfigMap) string {
	configString, ok := configMap.Data[OpAMPBridgeFilename]
	if !ok {
		return ""
	}
	h := sha256.Sum256([]byte(configString))
	return fmt.Sprintf("%x", h)
}

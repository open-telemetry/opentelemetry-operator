// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"crypto/sha256"
	"fmt"
	"maps"

	v1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
)

const configMapHashAnnotationKey = "opentelemetry-targetallocator-config/hash"

// ResourceAnnotations returns the TargetAllocator CR's metadata annotations,
// filtered, for propagation to the resources the operator creates for it,
// mirroring what manifestutils.Annotations does for the OpenTelemetryCollector.
func ResourceAnnotations(instance v1alpha1.TargetAllocator, filterAnnotations []string) map[string]string {
	annotations := make(map[string]string, len(instance.Annotations))
	for k, v := range instance.Annotations {
		if !manifestutils.IsFilteredSet(k, filterAnnotations) {
			annotations[k] = v
		}
	}
	return annotations
}

// Annotations returns the annotations for the TargetAllocator Pod.
func Annotations(instance v1alpha1.TargetAllocator, configMap *v1.ConfigMap, filterAnnotations []string) map[string]string {
	// Make a copy of PodAnnotations to be safe
	annotations := make(map[string]string, len(instance.Spec.PodAnnotations))
	maps.Copy(annotations, instance.Spec.PodAnnotations)
	if instance.Annotations != nil {
		for k, v := range instance.Annotations {
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

// getConfigMapSHA returns the hash of the content of the TA ConfigMap.
func getConfigMapSHA(configMap *v1.ConfigMap) string {
	configString, ok := configMap.Data[targetAllocatorFilename]
	if !ok {
		return ""
	}
	h := sha256.Sum256([]byte(configString))
	return fmt.Sprintf("%x", h)
}

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestPodAnnotations(t *testing.T) {
	instance := targetAllocatorInstance()
	instance.Spec.PodAnnotations = map[string]string{
		"key": "value",
	}
	annotations := Annotations(instance, nil, []string{".*\\.bar\\.io"})
	assert.Subset(t, annotations, instance.Spec.PodAnnotations)
}

func TestResourceAnnotations(t *testing.T) {
	t.Run("propagates CR metadata annotations", func(t *testing.T) {
		instance := targetAllocatorInstance()
		instance.Annotations = map[string]string{
			"key":        "value",
			"foo.bar.io": "filtered",
		}
		annotations := ResourceAnnotations(instance, []string{".*\\.bar\\.io"})
		assert.Equal(t, map[string]string{"key": "value"}, annotations)
	})

	t.Run("does not include pod annotations", func(t *testing.T) {
		instance := targetAllocatorInstance()
		instance.Spec.PodAnnotations = map[string]string{"key": "value"}
		instance.Annotations = map[string]string{"meta": "value"}
		annotations := ResourceAnnotations(instance, nil)
		assert.Equal(t, map[string]string{"meta": "value"}, annotations)
	})

	t.Run("returns an empty map when the CR has no annotations", func(t *testing.T) {
		instance := targetAllocatorInstance()
		instance.Annotations = nil
		annotations := ResourceAnnotations(instance, nil)
		assert.NotNil(t, annotations)
		assert.Empty(t, annotations)
	})

	t.Run("does not mutate the instance annotations", func(t *testing.T) {
		instance := targetAllocatorInstance()
		instance.Annotations = map[string]string{"key": "value"}
		annotations := ResourceAnnotations(instance, nil)
		annotations["other"] = "other"
		assert.Equal(t, map[string]string{"key": "value"}, instance.Annotations)
	})
}

func TestConfigMapHash(t *testing.T) {
	cfg := config.New()
	collector := collectorInstance()
	targetAllocator := targetAllocatorInstance()
	params := Params{
		Collector:       collector,
		TargetAllocator: targetAllocator,
		Config:          cfg,
		Log:             logr.Discard(),
	}
	expectedConfigMap, err := ConfigMap(params)
	require.NoError(t, err)
	expectedConfig := expectedConfigMap.Data[targetAllocatorFilename]
	require.NotEmpty(t, expectedConfig)
	expectedHash := sha256.Sum256([]byte(expectedConfig))
	annotations := Annotations(targetAllocator, expectedConfigMap, []string{".*\\.bar\\.io"})
	require.Contains(t, annotations, configMapHashAnnotationKey)
	cmHash := annotations[configMapHashAnnotationKey]
	assert.Equal(t, fmt.Sprintf("%x", expectedHash), cmHash)
}

func TestInvalidConfigNoHash(t *testing.T) {
	instance := targetAllocatorInstance()
	annotations := Annotations(instance, nil, []string{".*\\.bar\\.io"})
	require.NotContains(t, annotations, configMapHashAnnotationKey)
}

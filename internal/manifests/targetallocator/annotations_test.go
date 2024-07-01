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

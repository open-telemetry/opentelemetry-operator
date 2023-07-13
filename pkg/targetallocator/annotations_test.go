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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPodAnnotations(t *testing.T) {
	instance := collectorInstance()
	instance.Spec.PodAnnotations = map[string]string{
		"key": "value",
	}
	annotations := Annotations(instance)
	assert.Subset(t, annotations, instance.Spec.PodAnnotations)
}

func TestConfigMapHash(t *testing.T) {
	instance := collectorInstance()
	annotations := Annotations(instance)
	require.Contains(t, annotations, configMapHashAnnotationKey)
	cmHash := annotations[configMapHashAnnotationKey]
	assert.Len(t, cmHash, 64)
}

func TestInvalidConfigNoHash(t *testing.T) {
	instance := collectorInstance()
	instance.Spec.Config = ""
	annotations := Annotations(instance)
	require.NotContains(t, annotations, configMapHashAnnotationKey)
}

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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

func TestConfigMapHash(t *testing.T) {
	cfg := config.New()
	excludedAnnotations := map[string]string{
		"foo":         "1",
		"app.foo.bar": "1",
		"opampbridge": "true",
	}
	opampBridge := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-instance",
			Annotations: excludedAnnotations,
		},
		Spec: v1alpha1.OpAMPBridgeSpec{},
	}
	params := manifests.Params{
		OpAMPBridge: opampBridge,
		Config:      cfg,
		Log:         logr.Discard(),
	}
	expectedConfigMap, err := ConfigMap(params)
	require.NoError(t, err)
	expectedConfig := expectedConfigMap.Data[OpAMPBridgeFilename]
	require.NotEmpty(t, expectedConfig)
	expectedHash := sha256.Sum256([]byte(expectedConfig))
	annotations := Annotations(opampBridge, expectedConfigMap, []string{".*\\.bar\\.io"})
	require.Contains(t, annotations, configMapHashAnnotationKey)
	cmHash := annotations[configMapHashAnnotationKey]
	assert.Equal(t, fmt.Sprintf("%x", expectedHash), cmHash)
}

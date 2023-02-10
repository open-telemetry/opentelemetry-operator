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

package autodetect_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
)

func TestDetectPlatformBasedOnAvailableAPIGroups(t *testing.T) {
	for _, tt := range []struct {
		apiGroupList *metav1.APIGroupList
		expected     autodetect.OpenShiftRoutesAvailability
	}{
		{
			&metav1.APIGroupList{},
			autodetect.OpenShiftRoutesNotAvailable,
		},
		{
			&metav1.APIGroupList{
				Groups: []metav1.APIGroup{
					{
						Name: "route.openshift.io",
					},
				},
			},
			autodetect.OpenShiftRoutesAvailable,
		},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			output, err := json.Marshal(tt.apiGroupList)
			require.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err = w.Write(output)
			require.NoError(t, err)
		}))
		defer server.Close()

		autoDetect, err := autodetect.New(&rest.Config{Host: server.URL})
		require.NoError(t, err)

		// test
		ora, err := autoDetect.OpenShiftRoutesAvailability()

		// verify
		assert.NoError(t, err)
		assert.Equal(t, tt.expected, ora)
	}
}

func TestAutoscalingVersionToString(t *testing.T) {
	assert.Equal(t, "v2", autodetect.AutoscalingVersionV2.String())
	assert.Equal(t, "v2beta2", autodetect.AutoscalingVersionV2Beta2.String())
	assert.Equal(t, "unknown", autodetect.AutoscalingVersionUnknown.String())
}

func TestToAutoScalingVersion(t *testing.T) {
	assert.Equal(t, autodetect.AutoscalingVersionV2, autodetect.ToAutoScalingVersion("v2"))
	assert.Equal(t, autodetect.AutoscalingVersionV2Beta2, autodetect.ToAutoScalingVersion("v2beta2"))
	assert.Equal(t, autodetect.AutoscalingVersionUnknown, autodetect.ToAutoScalingVersion("fred"))
}

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

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
)

func TestDetectPlatformBasedOnAvailableAPIGroups(t *testing.T) {
	for _, tt := range []struct {
		apiGroupList *metav1.APIGroupList
		expected     openshift.RoutesAvailability
	}{
		{
			&metav1.APIGroupList{},
			openshift.RoutesNotAvailable,
		},
		{
			&metav1.APIGroupList{
				Groups: []metav1.APIGroup{
					{
						Name: "route.openshift.io",
					},
				},
			},
			openshift.RoutesAvailable,
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

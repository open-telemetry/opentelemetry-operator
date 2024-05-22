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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

func TestDesiredServiceMonitors(t *testing.T) {
	ta := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Tolerations: testTolerationValues,
			},
		},
	}
	cfg := config.New()

	params := manifests.Params{
		TargetAllocator: ta,
		Config:          cfg,
		Log:             logger,
	}

	actual := ServiceMonitor(params)
	assert.NotNil(t, actual)
	assert.Equal(t, fmt.Sprintf("%s-targetallocator", params.TargetAllocator.Name), actual.Name)
	assert.Equal(t, params.TargetAllocator.Namespace, actual.Namespace)
	assert.Equal(t, "targetallocation", actual.Spec.Endpoints[0].Port)

}

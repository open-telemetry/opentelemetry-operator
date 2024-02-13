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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha2"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

func TestServiceAccountDefaultName(t *testing.T) {
	// prepare
	otelcol := v1alpha2.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	// test
	saName := ServiceAccountName(otelcol)

	// verify
	assert.Equal(t, "my-instance-targetallocator", saName)
}

func TestServiceAccountOverrideName(t *testing.T) {
	// prepare
	otelcol := v1alpha2.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha2.OpenTelemetryCollectorSpec{
			TargetAllocator: v1alpha2.TargetAllocatorEmbedded{
				ServiceAccount: "my-special-sa",
			},
		},
	}

	// test
	sa := ServiceAccountName(otelcol)

	// verify
	assert.Equal(t, "my-special-sa", sa)
}

func TestServiceAccountDefault(t *testing.T) {
	params := manifests.Params{
		OtelCol: v1alpha2.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-instance",
			},
		},
	}
	expected := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-instance-targetallocator",
			Namespace:   params.OtelCol.Namespace,
			Labels:      Labels(params.OtelCol, "my-instance-targetallocator"),
			Annotations: params.OtelCol.Annotations,
		},
	}

	saName := ServiceAccountName(params.OtelCol)
	sa := ServiceAccount(params)

	assert.Equal(t, sa.Name, saName)
	assert.Equal(t, expected, sa)
}

func TestServiceAccountOverride(t *testing.T) {
	params := manifests.Params{
		OtelCol: v1alpha2.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-instance",
			},
			Spec: v1alpha2.OpenTelemetryCollectorSpec{
				TargetAllocator: v1alpha2.TargetAllocatorEmbedded{
					ServiceAccount: "my-special-sa",
				},
			},
		},
	}
	sa := ServiceAccount(params)

	assert.Nil(t, sa)
}

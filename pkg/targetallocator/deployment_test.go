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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestDeploymentNewDefault(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}
	cfg := config.New()

	// test
	d := Deployment(cfg, logger, otelcol)

	// verify
	assert.Equal(t, "my-instance-targetallocator", d.Name)
	assert.Equal(t, "my-instance-targetallocator", d.Labels["app.kubernetes.io/name"])

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)

	// none of the default annotations should propagate down to the pod
	assert.Empty(t, d.Spec.Template.Annotations)

	// the pod selector should match the pod spec's labels
	assert.Equal(t, d.Spec.Template.Labels, d.Spec.Selector.MatchLabels)
}

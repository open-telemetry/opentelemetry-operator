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
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	opampBridgeName      = "my-instance"
	opampBridgeNamespace = "my-ns"
)

func TestLabelsCommonSet(t *testing.T) {
	// prepare
	opampBridge := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opampBridgeName,
			Namespace: opampBridgeNamespace,
		},
		Spec: v1alpha1.OpAMPBridgeSpec{
			Image: "ghcr.io/open-telemetry/opentelemetry-operator/operator-opamp-bridge:0.69.0",
		},
	}

	// test
	labels := Labels(opampBridge, opampBridgeName, []string{})
	assert.Equal(t, "opentelemetry-operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-ns.my-instance", labels["app.kubernetes.io/instance"])
	assert.Equal(t, "0.69.0", labels["app.kubernetes.io/version"])
	assert.Equal(t, "opentelemetry", labels["app.kubernetes.io/part-of"])
	assert.Equal(t, "opentelemetry-opamp-bridge", labels["app.kubernetes.io/component"])
}

func TestLabelsTagUnset(t *testing.T) {
	// prepare
	opampBridge := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opampBridgeName,
			Namespace: opampBridgeNamespace,
		},
		Spec: v1alpha1.OpAMPBridgeSpec{
			Image: "ghcr.io/open-telemetry/opentelemetry-operator/operator-opamp-bridge",
		},
	}

	// test
	labels := Labels(opampBridge, opampBridgeName, []string{})
	assert.Equal(t, "opentelemetry-operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-ns.my-instance", labels["app.kubernetes.io/instance"])
	assert.Equal(t, "latest", labels["app.kubernetes.io/version"])
	assert.Equal(t, "opentelemetry", labels["app.kubernetes.io/part-of"])
	assert.Equal(t, "opentelemetry-opamp-bridge", labels["app.kubernetes.io/component"])
}

func TestLabelsPropagateDown(t *testing.T) {
	// prepare
	opampBridge := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"myapp":                  "mycomponent",
				"app.kubernetes.io/name": "test",
			},
		},
	}

	// test
	labels := Labels(opampBridge, opampBridgeName, []string{})

	// verify
	assert.Len(t, labels, 7)
	assert.Equal(t, "mycomponent", labels["myapp"])
	assert.Equal(t, "test", labels["app.kubernetes.io/name"])
}

func TestSelectorLabels(t *testing.T) {
	// prepare
	expected := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-opamp-bridge",
		"app.kubernetes.io/instance":   "my-namespace.my-opamp-bridge",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
	}
	opampBridge := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{Name: "my-opamp-bridge", Namespace: "my-namespace"},
	}

	// test
	result := SelectorLabels(opampBridge)

	// verify
	assert.Equal(t, expected, result)
}

func TestLabelsFilter(t *testing.T) {
	opampBridge := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"test.bar.io": "foo", "test.foo.io": "bar"},
		},
	}

	// This requires the filter to be in regex match form and not the other simpler wildcard one.
	labels := Labels(opampBridge, opampBridgeName, []string{".*.bar.io"})

	// verify
	assert.Len(t, labels, 6)
	assert.NotContains(t, labels, "test.bar.io")
	assert.Equal(t, "bar", labels["test.foo.io"])
}

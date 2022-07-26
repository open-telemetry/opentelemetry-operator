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

package upgrade

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestUpgrade(t *testing.T) {
	nsName := strings.ToLower(t.Name())
	err := k8sClient.Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	})
	require.NoError(t, err)

	inst := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-inst",
			Namespace: nsName,
			Annotations: map[string]string{
				v1alpha1.AnnotationDefaultAutoInstrumentationJava:   "java:1",
				v1alpha1.AnnotationDefaultAutoInstrumentationNodeJS: "nodejs:1",
				v1alpha1.AnnotationDefaultAutoInstrumentationPython: "python:1",
				v1alpha1.AnnotationDefaultAutoInstrumentationDotNet: "dotnet:1",
			},
		},
		Spec: v1alpha1.InstrumentationSpec{
			Sampler: v1alpha1.Sampler{
				Type: v1alpha1.ParentBasedAlwaysOff,
			},
		},
	}
	inst.Default()
	assert.Equal(t, "java:1", inst.Spec.Java.Image)
	assert.Equal(t, "nodejs:1", inst.Spec.NodeJS.Image)
	assert.Equal(t, "python:1", inst.Spec.Python.Image)
	assert.Equal(t, "dotnet:1", inst.Spec.DotNet.Image)
	err = k8sClient.Create(context.Background(), inst)
	require.NoError(t, err)

	up := &InstrumentationUpgrade{
		Logger:                logr.Discard(),
		DefaultAutoInstJava:   "java:2",
		DefaultAutoInstNodeJS: "nodejs:2",
		DefaultAutoInstPython: "python:2",
		DefaultAutoInstDotNet: "dotnet:2",
		Client:                k8sClient,
	}
	err = up.ManagedInstances(context.Background())
	require.NoError(t, err)

	updated := v1alpha1.Instrumentation{}
	err = k8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: nsName,
		Name:      "my-inst",
	}, &updated)
	require.NoError(t, err)
	assert.Equal(t, "java:2", updated.Annotations[v1alpha1.AnnotationDefaultAutoInstrumentationJava])
	assert.Equal(t, "java:2", updated.Spec.Java.Image)
	assert.Equal(t, "nodejs:2", updated.Annotations[v1alpha1.AnnotationDefaultAutoInstrumentationNodeJS])
	assert.Equal(t, "nodejs:2", updated.Spec.NodeJS.Image)
	assert.Equal(t, "python:2", updated.Annotations[v1alpha1.AnnotationDefaultAutoInstrumentationPython])
	assert.Equal(t, "python:2", updated.Spec.Python.Image)
	assert.Equal(t, "dotnet:2", updated.Annotations[v1alpha1.AnnotationDefaultAutoInstrumentationDotNet])
	assert.Equal(t, "dotnet:2", updated.Spec.DotNet.Image)
}

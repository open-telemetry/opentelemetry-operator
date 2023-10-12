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
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

func TestUpgrade(t *testing.T) {
	originalVal := featuregate.EnableGoAutoInstrumentationSupport.IsEnabled()
	require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableGoAutoInstrumentationSupport.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableGoAutoInstrumentationSupport.ID(), originalVal))
	})

	originalVal = featuregate.EnableApacheHTTPAutoInstrumentationSupport.IsEnabled()
	require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableApacheHTTPAutoInstrumentationSupport.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableApacheHTTPAutoInstrumentationSupport.ID(), originalVal))
	})

	originalVal = featuregate.EnableNginxAutoInstrumentationSupport.IsEnabled()
	require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableNginxAutoInstrumentationSupport.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableNginxAutoInstrumentationSupport.ID(), originalVal))
	})

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
		},
		Spec: v1alpha1.InstrumentationSpec{
			Sampler: v1alpha1.Sampler{
				Type: v1alpha1.ParentBasedAlwaysOff,
			},
		},
	}
	err = v1alpha1.NewInstrumentationWebhook(
		logr.Discard(),
		testScheme,
		config.New(
			config.WithAutoInstrumentationJavaImage("java:1"),
			config.WithAutoInstrumentationNodeJSImage("nodejs:1"),
			config.WithAutoInstrumentationPythonImage("python:1"),
			config.WithAutoInstrumentationDotNetImage("dotnet:1"),
			config.WithAutoInstrumentationGoImage("go:1"),
			config.WithAutoInstrumentationApacheHttpdImage("apache-httpd:1"),
			config.WithAutoInstrumentationNginxImage("nginx:1"),
		),
	).Default(context.Background(), inst)
	assert.Nil(t, err)
	assert.Equal(t, "java:1", inst.Spec.Java.Image)
	assert.Equal(t, "nodejs:1", inst.Spec.NodeJS.Image)
	assert.Equal(t, "python:1", inst.Spec.Python.Image)
	assert.Equal(t, "dotnet:1", inst.Spec.DotNet.Image)
	assert.Equal(t, "go:1", inst.Spec.Go.Image)
	assert.Equal(t, "apache-httpd:1", inst.Spec.ApacheHttpd.Image)
	assert.Equal(t, "nginx:1", inst.Spec.Nginx.Image)
	err = k8sClient.Create(context.Background(), inst)
	require.NoError(t, err)

	up := &InstrumentationUpgrade{
		Logger:                     logr.Discard(),
		DefaultAutoInstJava:        "java:2",
		DefaultAutoInstNodeJS:      "nodejs:2",
		DefaultAutoInstPython:      "python:2",
		DefaultAutoInstDotNet:      "dotnet:2",
		DefaultAutoInstGo:          "go:2",
		DefaultAutoInstApacheHttpd: "apache-httpd:2",
		DefaultAutoInstNginx:       "nginx:2",
		Client:                     k8sClient,
	}
	err = up.ManagedInstances(context.Background())
	require.NoError(t, err)

	updated := v1alpha1.Instrumentation{}
	err = k8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: nsName,
		Name:      "my-inst",
	}, &updated)
	require.NoError(t, err)
	assert.Equal(t, "java:2", updated.Annotations[constants.AnnotationDefaultAutoInstrumentationJava])
	assert.Equal(t, "java:2", updated.Spec.Java.Image)
	assert.Equal(t, "nodejs:2", updated.Annotations[constants.AnnotationDefaultAutoInstrumentationNodeJS])
	assert.Equal(t, "nodejs:2", updated.Spec.NodeJS.Image)
	assert.Equal(t, "python:2", updated.Annotations[constants.AnnotationDefaultAutoInstrumentationPython])
	assert.Equal(t, "python:2", updated.Spec.Python.Image)
	assert.Equal(t, "dotnet:2", updated.Annotations[constants.AnnotationDefaultAutoInstrumentationDotNet])
	assert.Equal(t, "dotnet:2", updated.Spec.DotNet.Image)
	assert.Equal(t, "go:2", updated.Annotations[constants.AnnotationDefaultAutoInstrumentationGo])
	assert.Equal(t, "go:2", updated.Spec.Go.Image)
	assert.Equal(t, "apache-httpd:2", updated.Annotations[constants.AnnotationDefaultAutoInstrumentationApacheHttpd])
	assert.Equal(t, "apache-httpd:2", updated.Spec.ApacheHttpd.Image)
	assert.Equal(t, "nginx:2", updated.Annotations[constants.AnnotationDefaultAutoInstrumentationNginx])
	assert.Equal(t, "nginx:2", updated.Spec.Nginx.Image)
}

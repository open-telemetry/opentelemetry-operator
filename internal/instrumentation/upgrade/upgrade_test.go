// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
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
		},
		Spec: v1alpha1.InstrumentationSpec{
			Sampler: v1alpha1.Sampler{
				Type: v1alpha1.ParentBasedAlwaysOff,
			},
		},
	}

	cfg := config.Config{
		AutoInstrumentationJavaImage:        "java:1",
		AutoInstrumentationNodeJSImage:      "nodejs:1",
		AutoInstrumentationPythonImage:      "python:1",
		AutoInstrumentationDotNetImage:      "dotnet:1",
		AutoInstrumentationGoImage:          "go:1",
		AutoInstrumentationApacheHttpdImage: "apache-httpd:1",
		AutoInstrumentationNginxImage:       "nginx:1",
		EnableApacheHttpdInstrumentation:    true,
		EnableDotNetAutoInstrumentation:     true,
		EnableGoAutoInstrumentation:         true,
		EnableNginxAutoInstrumentation:      true,
		EnablePythonAutoInstrumentation:     true,
		EnableNodeJSAutoInstrumentation:     true,
		EnableJavaAutoInstrumentation:       true,
	}
	err = v1alpha1.NewInstrumentationWebhook(
		logr.Discard(),
		testScheme,
		cfg,
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

	cfg = config.Config{
		AutoInstrumentationJavaImage:        "java:2",
		AutoInstrumentationNodeJSImage:      "nodejs:2",
		AutoInstrumentationPythonImage:      "python:2",
		AutoInstrumentationDotNetImage:      "dotnet:2",
		AutoInstrumentationGoImage:          "go:2",
		AutoInstrumentationApacheHttpdImage: "apache-httpd:2",
		AutoInstrumentationNginxImage:       "nginx:2",
		EnableApacheHttpdInstrumentation:    true,
		EnableDotNetAutoInstrumentation:     true,
		EnableGoAutoInstrumentation:         true,
		EnableNginxAutoInstrumentation:      true,
		EnablePythonAutoInstrumentation:     true,
		EnableNodeJSAutoInstrumentation:     true,
		EnableJavaAutoInstrumentation:       true,
	}
	up := NewInstrumentationUpgrade(k8sClient, ctrl.Log.WithName("instrumentation-upgrade"), &record.FakeRecorder{}, cfg)

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

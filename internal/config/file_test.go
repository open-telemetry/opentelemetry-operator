// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
)

func TestDefaultConfig(t *testing.T) {
	cfg := New()
	f, err := os.ReadFile("testdata/config.yaml")
	require.NoError(t, err)
	actual := Config{}
	require.NoError(t, yaml.Unmarshal(f, &actual))
	assert.Equal(t, cfg, actual)
}

func TestSomeChanges(t *testing.T) {
	f, err := os.ReadFile("testdata/config2.yaml")
	require.NoError(t, err)
	actual := Config{}
	require.NoError(t, yaml.Unmarshal(f, &actual))
	assert.Equal(t, "foo:1", actual.AutoInstrumentationDotNetImage)
	assert.Equal(t, "foobar:1", actual.AutoInstrumentationGoImage)
	assert.Equal(t, "bar:1", actual.AutoInstrumentationApacheHttpdImage)
}

func TestInstrumentationInConfigFile(t *testing.T) {
	actual := Config{}
	require.NoError(t, ApplyConfigFile("testdata/config_instrumentation.yaml", &actual))
	assert.Equal(t, "foo:1", actual.AutoInstrumentationDotNetImage)
	assert.Equal(t, "foobar:1", actual.AutoInstrumentationGoImage)
	assert.Equal(t, "bar:1", actual.AutoInstrumentationApacheHttpdImage)
	assert.Equal(t, "myjavainstrumentation:latest", actual.Instrumentation.Spec.Java.Image)
	assert.Equal(t, "myapacheinstrumentation:latest", actual.Instrumentation.Spec.ApacheHttpd.Image)
	assert.Equal(t, "bar", actual.Instrumentation.Spec.Resource.Attributes["foo"])
}

func TestInstrumentationConfigFileHonorsJSONTags(t *testing.T) {
	actual := Config{}
	require.NoError(t, ApplyConfigFile("testdata/config_instrumentation.yaml", &actual))
	spec := actual.Instrumentation.Spec

	assert.Equal(t, "foo:1", actual.AutoInstrumentationDotNetImage)

	assert.Equal(t, "http://$(OTEL_NODE_IP):4317", spec.Endpoint)
	require.NotNil(t, spec.TLS)
	assert.Equal(t, "exporter-tls", spec.TLS.SecretName)
	assert.Equal(t, "exporter-ca", spec.TLS.ConfigMapName)

	assert.Equal(t, "bar", spec.Resource.Attributes["foo"])
	assert.True(t, spec.Resource.AddK8sUIDAttributes)
	assert.True(t, spec.Defaults.UseLabelsForResourceAttributes)
	assert.Equal(t, corev1.PullIfNotPresent, spec.ImagePullPolicy)

	require.NotNil(t, spec.InitContainerSecurityContext)
	require.NotNil(t, spec.InitContainerSecurityContext.RunAsNonRoot)
	assert.True(t, *spec.InitContainerSecurityContext.RunAsNonRoot)
	require.NotNil(t, spec.InitContainerSecurityContext.RunAsUser)
	assert.Equal(t, int64(1000), *spec.InitContainerSecurityContext.RunAsUser)

	require.Len(t, spec.Env, 1)
	env := spec.Env[0]
	assert.Equal(t, "OTEL_NODE_IP", env.Name)
	require.NotNil(t, env.ValueFrom)
	require.NotNil(t, env.ValueFrom.FieldRef)
	assert.Equal(t, "status.hostIP", env.ValueFrom.FieldRef.FieldPath)
	assert.Empty(t, env.Value)

	assert.Equal(t, "myjavainstrumentation:latest", spec.Java.Image)
	require.NotNil(t, spec.Java.VolumeSizeLimit)
	assert.Equal(t, "100Mi", spec.Java.VolumeSizeLimit.String())
	require.Len(t, spec.Java.Env, 2)
	require.NotNil(t, spec.Java.Env[1].ValueFrom)
	require.NotNil(t, spec.Java.Env[1].ValueFrom.FieldRef)
	assert.Equal(t, "status.hostIP", spec.Java.Env[1].ValueFrom.FieldRef.FieldPath)

	assert.Equal(t, "myapacheinstrumentation:latest", spec.ApacheHttpd.Image)
	assert.Equal(t, "/usr/local/apache2/conf", spec.ApacheHttpd.ConfigPath)
	require.NotNil(t, spec.ApacheHttpd.VolumeSizeLimit)
	assert.Equal(t, "50Mi", spec.ApacheHttpd.VolumeSizeLimit.String())

	assert.Equal(t, "mynginxinstrumentation:latest", spec.Nginx.Image)
	assert.Equal(t, "/etc/nginx/nginx.conf", spec.Nginx.ConfigFile)

	assert.Equal(t, "mygoinstrumentation:latest", spec.Go.Image)
	require.NotNil(t, spec.Go.SecurityContext)
	require.NotNil(t, spec.Go.SecurityContext.RunAsNonRoot)
	assert.True(t, *spec.Go.SecurityContext.RunAsNonRoot)
}

func TestInstrumentationYAMLV2DropsJSONTags(t *testing.T) {
	f, err := os.ReadFile("testdata/config_instrumentation.yaml")
	require.NoError(t, err)
	broken := Config{}
	require.NoError(t, yaml.Unmarshal(f, &broken))
	spec := broken.Instrumentation.Spec

	assert.Empty(t, spec.ApacheHttpd.Image, "yaml.v2 maps ApacheHttpd to apachehttpd, not apacheHttpd")
	assert.Empty(t, spec.ImagePullPolicy)
	assert.Nil(t, spec.InitContainerSecurityContext)
	assert.Empty(t, spec.Resource.Attributes, "json tag is resourceAttributes, not attributes")
	assert.False(t, spec.Resource.AddK8sUIDAttributes)
	assert.False(t, spec.Defaults.UseLabelsForResourceAttributes)
	require.NotNil(t, spec.TLS, "tls: key matches, but nested camelCase fields drop")
	assert.Empty(t, spec.TLS.SecretName)
	assert.Empty(t, spec.TLS.ConfigMapName)
	assert.Empty(t, spec.ApacheHttpd.ConfigPath)
	assert.Nil(t, spec.ApacheHttpd.VolumeSizeLimit)
	assert.Empty(t, spec.Nginx.ConfigFile)
	assert.Nil(t, spec.Go.SecurityContext)
	assert.Nil(t, spec.Java.VolumeSizeLimit)

	require.Len(t, spec.Env, 1)
	assert.Equal(t, "OTEL_NODE_IP", spec.Env[0].Name)
	assert.Nil(t, spec.Env[0].ValueFrom, "yaml.v2 drops EnvVar valueFrom")
	assert.Empty(t, spec.Env[0].Value)

	require.Len(t, spec.Java.Env, 2)
	assert.Nil(t, spec.Java.Env[1].ValueFrom)

	assert.Equal(t, "myjavainstrumentation:latest", spec.Java.Image)
	assert.Equal(t, "http://$(OTEL_NODE_IP):4317", spec.Endpoint)
	assert.Equal(t, "mynginxinstrumentation:latest", spec.Nginx.Image)
	assert.Equal(t, "mygoinstrumentation:latest", spec.Go.Image)
}

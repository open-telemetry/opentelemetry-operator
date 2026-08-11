// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func TestInstrumentationConvertTo(t *testing.T) {
	volumeSize := resource.MustParse("500Mi")

	src := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-inst",
			Namespace: "default",
		},
		Spec: v1alpha1.InstrumentationSpec{
			Exporter: v1alpha1.Exporter{
				Endpoint: "http://collector:4317",
				TLS: &v1alpha1.TLS{
					SecretName:    "tls-secret",
					ConfigMapName: "ca-configmap",
					CA:            "ca.crt",
					Cert:          "tls.crt",
					Key:           "tls.key",
				},
			},
			Propagators: []v1alpha1.Propagator{v1alpha1.TraceContext, v1alpha1.Baggage},
			Sampler: v1alpha1.Sampler{
				Type:     v1alpha1.SamplerType("parentbased_traceidratio"),
				Argument: "0.25",
			},
			Resource: v1alpha1.Resource{
				Attributes:          map[string]string{"env": "test"},
				AddK8sUIDAttributes: true,
			},
			Defaults: v1alpha1.Defaults{
				UseLabelsForResourceAttributes: true,
			},
			Env: []corev1.EnvVar{
				{Name: "COMMON_ENV", Value: "common"},
			},
			Java: v1alpha1.Java{
				Image:           "java-image:latest",
				VolumeSizeLimit: &volumeSize,
				Env: []corev1.EnvVar{
					{Name: "JAVA_ENV", Value: "java"},
				},
				Extensions: []v1alpha1.Extensions{
					{Image: "ext-image", Dir: "/ext"},
				},
			},
			ImagePullPolicy: corev1.PullAlways,
		},
		Status: v1alpha1.InstrumentationStatus{
			UpgradeBlockedVersions: map[string]string{"java": "blocked"},
		},
	}

	dst := &v1beta1.Instrumentation{}
	err := InstrumentationConvertTo(src, dst)
	require.NoError(t, err)

	lossyFields := v1alpha1Fields{
		JavaVolumeSizeLimit: &volumeSize,
	}
	annotationBytes, err := json.Marshal(lossyFields)
	require.NoError(t, err)

	enabled := true
	expected := v1beta1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-inst",
			Namespace: "default",
			Annotations: map[string]string{
				v1alpha1FieldsAnnotation: string(annotationBytes),
			},
		},
		Spec: v1beta1.InstrumentationSpec{
			EnvConfig: &v1beta1.EnvConfig{
				Exporter: v1beta1.Exporter{
					Endpoint: "http://collector:4317",
					TLS: &v1beta1.TLS{
						SecretName:    "tls-secret",
						ConfigMapName: "ca-configmap",
						CA:            "ca.crt",
						Cert:          "tls.crt",
						Key:           "tls.key",
					},
				},
				Propagators: []v1beta1.Propagator{
					v1beta1.Propagator("tracecontext"),
					v1beta1.Propagator("baggage"),
				},
				Sampler: v1beta1.Sampler{
					Type:     v1beta1.SamplerType("parentbased_traceidratio"),
					Argument: "0.25",
				},
			},
			Resource: v1beta1.Resource{
				Attributes: map[string]string{"env": "test"},
				K8sMetadata: &v1beta1.K8sMetadataConfig{
					IncludeUIDs: true,
				},
				ServiceMetadata: &v1beta1.ServiceMetadataConfig{
					Enabled: &enabled,
				},
			},
			Env: []corev1.EnvVar{
				{Name: "COMMON_ENV", Value: "common"},
			},
			Java: v1beta1.Java{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image: "java-image:latest",
					Env: []corev1.EnvVar{
						{Name: "JAVA_ENV", Value: "java"},
					},
				},
				Extensions: []v1beta1.Extensions{
					{Image: "ext-image", Dir: "/ext"},
				},
			},
			ImagePullPolicy: corev1.PullAlways,
		},
		Status: v1beta1.InstrumentationStatus{
			UpgradeBlockedVersions: map[string]string{"java": "blocked"},
		},
	}

	assert.Equal(t, expected, *dst)
}

func TestInstrumentationConvertFrom(t *testing.T) {
	enabled := true

	src := &v1beta1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-inst",
			Namespace: "default",
		},
		Spec: v1beta1.InstrumentationSpec{
			EnvConfig: &v1beta1.EnvConfig{
				Exporter: v1beta1.Exporter{
					Endpoint: "http://collector:4318",
					Protocol: "http/protobuf",
					TLS: &v1beta1.TLS{
						SecretName: "tls-secret",
					},
				},
				Propagators: []v1beta1.Propagator{v1beta1.B3},
				Sampler: v1beta1.Sampler{
					Type: v1beta1.SamplerType("always_on"),
				},
			},
			Resource: v1beta1.Resource{
				Attributes: map[string]string{"service": "api"},
				K8sMetadata: &v1beta1.K8sMetadataConfig{
					Enabled:     &enabled,
					IncludeUIDs: false,
				},
				ServiceMetadata: &v1beta1.ServiceMetadataConfig{
					Enabled: &enabled,
				},
			},
			Env: []corev1.EnvVar{
				{Name: "OTEL_ENV", Value: "prod"},
			},
			Java: v1beta1.Java{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image: "java-beta:latest",
				},
			},
		},
		Status: v1beta1.InstrumentationStatus{
			UpgradeBlockedVersions: map[string]string{"nodejs": "blocked"},
		},
	}

	dst := &v1alpha1.Instrumentation{}
	err := InstrumentationConvertFrom(dst, src)
	require.NoError(t, err)

	expected := v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-inst",
			Namespace: "default",
		},
		Spec: v1alpha1.InstrumentationSpec{
			Exporter: v1alpha1.Exporter{
				Endpoint: "http://collector:4318",
				TLS: &v1alpha1.TLS{
					SecretName: "tls-secret",
				},
			},
			Propagators: []v1alpha1.Propagator{v1alpha1.Propagator("b3")},
			Sampler: v1alpha1.Sampler{
				Type: v1alpha1.SamplerType("always_on"),
			},
			Resource: v1alpha1.Resource{
				Attributes: map[string]string{"service": "api"},
			},
			Defaults: v1alpha1.Defaults{
				UseLabelsForResourceAttributes: true,
			},
			Env: []corev1.EnvVar{
				{Name: "OTEL_ENV", Value: "prod"},
			},
			Java: v1alpha1.Java{
				Image: "java-beta:latest",
			},
		},
		Status: v1alpha1.InstrumentationStatus{
			UpgradeBlockedVersions: map[string]string{"nodejs": "blocked"},
		},
	}

	assert.Equal(t, expected, *dst)
}

func TestInstrumentationConvertFromWithAnnotation(t *testing.T) {
	volumeSize := resource.MustParse("300Mi")

	src := &v1beta1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-inst",
			Namespace: "default",
			Annotations: map[string]string{
				v1alpha1FieldsAnnotation: `{"javaVolumeSizeLimit":"300Mi","apacheHttpdAttrs":[{"name":"ApacheModuleOtelExporterEnabled","value":"ON"}]}`,
			},
		},
		Spec: v1beta1.InstrumentationSpec{
			EnvConfig: &v1beta1.EnvConfig{
				Exporter: v1beta1.Exporter{
					Endpoint: "http://collector:4317",
				},
			},
			Java: v1beta1.Java{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image: "java:latest",
					Env: []corev1.EnvVar{
						{Name: "JAVA_OPTS", Value: "-Xmx512m"},
					},
				},
			},
			ApacheHttpd: v1beta1.ApacheHttpd{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image: "apache:latest",
				},
			},
		},
	}

	dst := &v1alpha1.Instrumentation{}
	err := InstrumentationConvertFrom(dst, src)
	require.NoError(t, err)

	expected := v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-inst",
			Namespace:   "default",
			Annotations: map[string]string{},
		},
		Spec: v1alpha1.InstrumentationSpec{
			Exporter: v1alpha1.Exporter{
				Endpoint: "http://collector:4317",
			},
			Java: v1alpha1.Java{
				Image: "java:latest",
				Env: []corev1.EnvVar{
					{Name: "JAVA_OPTS", Value: "-Xmx512m"},
				},
				VolumeSizeLimit: &volumeSize,
			},
			ApacheHttpd: v1alpha1.ApacheHttpd{
				Image: "apache:latest",
				Attrs: []corev1.EnvVar{
					{Name: "ApacheModuleOtelExporterEnabled", Value: "ON"},
				},
			},
		},
	}

	assert.Equal(t, expected, *dst)
}

func TestInstrumentationConvertFromMalformedAnnotation(t *testing.T) {
	src := &v1beta1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-inst",
			Namespace: "default",
			Annotations: map[string]string{
				v1alpha1FieldsAnnotation: `{malformed json`,
			},
		},
		Spec: v1beta1.InstrumentationSpec{
			EnvConfig: &v1beta1.EnvConfig{
				Exporter: v1beta1.Exporter{
					Endpoint: "http://collector:4317",
				},
			},
		},
	}

	dst := &v1alpha1.Instrumentation{}
	err := InstrumentationConvertFrom(dst, src)
	require.NoError(t, err)

	expected := v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-inst",
			Namespace: "default",
			Annotations: map[string]string{
				v1alpha1FieldsAnnotation: `{malformed json`,
			},
		},
		Spec: v1alpha1.InstrumentationSpec{
			Exporter: v1alpha1.Exporter{
				Endpoint: "http://collector:4317",
			},
		},
	}

	assert.Equal(t, expected, *dst)
}

// TestInstrumentationRoundTrip tests full round-trip conversion with all fields populated.
func TestInstrumentationRoundTrip(t *testing.T) {
	volumeSize := resource.MustParse("500Mi")
	privileged := true
	runAsUser := int64(1000)

	original := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "roundtrip-test",
			Namespace: "test-ns",
			Labels: map[string]string{
				"app": "test",
			},
			Annotations: map[string]string{
				"existing": "annotation",
			},
		},
		Spec: v1alpha1.InstrumentationSpec{
			Exporter: v1alpha1.Exporter{
				Endpoint: "https://collector:4317",
				TLS: &v1alpha1.TLS{
					SecretName:    "tls-secret",
					ConfigMapName: "ca-configmap",
					CA:            "ca.crt",
					Cert:          "tls.crt",
					Key:           "tls.key",
				},
			},
			Propagators: []v1alpha1.Propagator{
				v1alpha1.TraceContext,
				v1alpha1.Baggage,
				v1alpha1.B3,
			},
			Sampler: v1alpha1.Sampler{
				Type:     v1alpha1.SamplerType("parentbased_traceidratio"),
				Argument: "0.5",
			},
			Resource: v1alpha1.Resource{
				Attributes: map[string]string{
					"env":     "prod",
					"service": "api",
				},
				AddK8sUIDAttributes: true,
			},
			Defaults: v1alpha1.Defaults{
				UseLabelsForResourceAttributes: true,
			},
			Env: []corev1.EnvVar{
				{Name: "COMMON_VAR", Value: "common-value"},
			},
			Java: v1alpha1.Java{
				Image:           "java-image:v1",
				VolumeSizeLimit: &volumeSize,
				Env: []corev1.EnvVar{
					{Name: "JAVA_OPTS", Value: "-Xmx512m"},
				},
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				},
				Extensions: []v1alpha1.Extensions{
					{Image: "ext-image:v1", Dir: "/extensions"},
				},
			},
			NodeJS: v1alpha1.NodeJS{
				Image:           "nodejs-image:v1",
				VolumeSizeLimit: &volumeSize,
				Env: []corev1.EnvVar{
					{Name: "NODE_OPTIONS", Value: "--max-old-space-size=4096"},
				},
			},
			Python: v1alpha1.Python{
				Image: "python-image:v1",
				Env: []corev1.EnvVar{
					{Name: "PYTHONPATH", Value: "/app"},
				},
			},
			DotNet: v1alpha1.DotNet{
				Image: "dotnet-image:v1",
				Env: []corev1.EnvVar{
					{Name: "DOTNET_ENVIRONMENT", Value: "Production"},
				},
			},
			Go: v1alpha1.Go{
				Image: "go-image:v1",
				Env: []corev1.EnvVar{
					{Name: "OTEL_GO_AUTO_TARGET_EXE", Value: "/app/main"},
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
			},
			ApacheHttpd: v1alpha1.ApacheHttpd{
				Image: "apache-image:v1",
				Env: []corev1.EnvVar{
					{Name: "APACHE_LOG_LEVEL", Value: "info"},
				},
				Attrs: []corev1.EnvVar{
					{Name: "ApacheModuleOtelExporterEnabled", Value: "ON"},
				},
				Version:    "2.4",
				ConfigPath: "/etc/apache2",
			},
			Nginx: v1alpha1.Nginx{
				Image: "nginx-image:v1",
				Env: []corev1.EnvVar{
					{Name: "NGINX_LOG_LEVEL", Value: "warn"},
				},
				Attrs: []corev1.EnvVar{
					{Name: "NginxModuleOtelExporterEnabled", Value: "ON"},
				},
				ConfigFile: "/etc/nginx/nginx.conf",
			},
			ImagePullPolicy: corev1.PullAlways,
			InitContainerSecurityContext: &corev1.SecurityContext{
				RunAsUser: &runAsUser,
			},
		},
		Status: v1alpha1.InstrumentationStatus{
			UpgradeBlockedVersions: map[string]string{
				"java": "blocked-version",
			},
		},
	}

	// Convert to v1beta1
	beta := &v1beta1.Instrumentation{}
	err := InstrumentationConvertTo(original, beta)
	require.NoError(t, err)

	// Convert back to v1alpha1
	roundTripped := &v1alpha1.Instrumentation{}
	err = InstrumentationConvertFrom(roundTripped, beta)
	require.NoError(t, err)

	// The round-trip should produce an identical object.
	// The lossy-field annotation is created during ConvertTo and
	// consumed (restored + deleted) during ConvertFrom.
	assert.Equal(t, *original, *roundTripped)
}

// TestInstrumentationConvertFromWithoutAnnotation tests conversion from native v1beta1 (no annotation).
func TestInstrumentationConvertFromWithoutAnnotation(t *testing.T) {
	src := &v1beta1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "native-beta",
			Namespace: "default",
		},
		Spec: v1beta1.InstrumentationSpec{
			EnvConfig: &v1beta1.EnvConfig{
				Exporter: v1beta1.Exporter{
					Endpoint: "http://collector:4318",
				},
			},
			Java: v1beta1.Java{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image: "java:latest",
				},
			},
			ApacheHttpd: v1beta1.ApacheHttpd{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image: "apache:latest",
				},
			},
			Nginx: v1beta1.Nginx{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image: "nginx:latest",
				},
			},
		},
	}

	dst := &v1alpha1.Instrumentation{}
	err := InstrumentationConvertFrom(dst, src)
	require.NoError(t, err)

	expected := v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "native-beta",
			Namespace: "default",
		},
		Spec: v1alpha1.InstrumentationSpec{
			Exporter: v1alpha1.Exporter{
				Endpoint: "http://collector:4318",
			},
			Java: v1alpha1.Java{
				Image: "java:latest",
			},
			ApacheHttpd: v1alpha1.ApacheHttpd{
				Image: "apache:latest",
			},
			Nginx: v1alpha1.Nginx{
				Image: "nginx:latest",
			},
		},
	}

	assert.Equal(t, expected, *dst)
}

// TestInstrumentationConvertMinimal tests round-trip with minimal data.
func TestInstrumentationConvertMinimal(t *testing.T) {
	original := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minimal",
			Namespace: "minimal-ns",
		},
	}

	// Convert to v1beta1
	beta := &v1beta1.Instrumentation{}
	err := InstrumentationConvertTo(original, beta)
	require.NoError(t, err)

	// Convert back to v1alpha1
	roundTripped := &v1alpha1.Instrumentation{}
	err = InstrumentationConvertFrom(roundTripped, beta)
	require.NoError(t, err)

	assert.Equal(t, *original, *roundTripped)
}

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

var defaultVolumeSize = resource.MustParse("200Mi")

func TestInstrumentationDefaultingWebhook(t *testing.T) {
	tests := []struct {
		name     string
		config   []config.Option
		input    Instrumentation
		expected Instrumentation
	}{
		{
			name: "default images",
			config: []config.Option{
				config.WithAutoInstrumentationJavaImage("java-img:1"),
				config.WithAutoInstrumentationNodeJSImage("nodejs-img:1"),
				config.WithAutoInstrumentationPythonImage("python-img:1"),
				config.WithAutoInstrumentationDotNetImage("dotnet-img:1"),
				config.WithAutoInstrumentationGoImage("go-img:1"),
				config.WithAutoInstrumentationNginxImage("nginx-img:1"),
				config.WithAutoInstrumentationApacheHttpdImage("apache-httpd-img:1"),
			},
			input: Instrumentation{},
			expected: Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/default-auto-instrumentation-java-image":         "java-img:1",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-nodejs-image":       "nodejs-img:1",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-python-image":       "python-img:1",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-dotnet-image":       "dotnet-img:1",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-go-image":           "go-img:1",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-nginx-image":        "nginx-img:1",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-apache-httpd-image": "apache-httpd-img:1",
					},
				},
				Spec: InstrumentationSpec{
					Java: Java{
						Image: "java-img:1",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
						},
					},
					NodeJS: NodeJS{
						Image: "nodejs-img:1",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
					Python: Python{
						Image: "python-img:1",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
						},
					},
					DotNet: DotNet{
						Image: "dotnet-img:1",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
					Go: Go{
						Image: "go-img:1",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
						},
					},
					ApacheHttpd: ApacheHttpd{
						Image:      "apache-httpd-img:1",
						Version:    "2.4",
						ConfigPath: "/usr/local/apache2/conf",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
					Nginx: Nginx{
						Image:      "nginx-img:1",
						ConfigFile: "/etc/nginx/nginx.conf",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
				},
			},
		},
		{
			name: "do not override java image",
			config: []config.Option{
				config.WithAutoInstrumentationJavaImage("java-img:1"),
			},
			input: Instrumentation{
				Spec: InstrumentationSpec{
					Java: Java{
						Image: "custom-java-img:2",
					},
				},
			},
			expected: Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/default-auto-instrumentation-java-image":         "java-img:1",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-nodejs-image":       "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-python-image":       "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-dotnet-image":       "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-go-image":           "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-nginx-image":        "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-apache-httpd-image": "",
					},
				},
				Spec: InstrumentationSpec{
					Java: Java{
						Image: "custom-java-img:2",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
						},
					},
					NodeJS: NodeJS{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
					Python: Python{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
						},
					},
					DotNet: DotNet{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
					Go: Go{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
						},
					},
					ApacheHttpd: ApacheHttpd{
						Version:    "2.4",
						ConfigPath: "/usr/local/apache2/conf",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
					Nginx: Nginx{
						ConfigFile: "/etc/nginx/nginx.conf",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
				},
			},
		},
		{
			name: "preserve java env vars",
			config: []config.Option{
				config.WithAutoInstrumentationJavaImage("java-img:1"),
			},
			input: Instrumentation{
				Spec: InstrumentationSpec{
					Java: Java{
						Env: []corev1.EnvVar{
							{
								Name:  "JAVA_TOOL_OPTIONS",
								Value: "-javaagent:/agent/opentelemetry-javaagent.jar",
							},
						},
					},
				},
			},
			expected: Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/default-auto-instrumentation-java-image":         "java-img:1",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-nodejs-image":       "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-python-image":       "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-dotnet-image":       "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-go-image":           "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-nginx-image":        "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-apache-httpd-image": "",
					},
				},
				Spec: InstrumentationSpec{
					Java: Java{
						Image: "java-img:1",
						Env: []corev1.EnvVar{
							{
								Name:  "JAVA_TOOL_OPTIONS",
								Value: "-javaagent:/agent/opentelemetry-javaagent.jar",
							},
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
						},
					},
					NodeJS: NodeJS{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
					Python: Python{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
						},
					},
					DotNet: DotNet{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
					Go: Go{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
						},
					},
					ApacheHttpd: ApacheHttpd{
						Version:    "2.4",
						ConfigPath: "/usr/local/apache2/conf",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
					Nginx: Nginx{
						ConfigFile: "/etc/nginx/nginx.conf",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
				},
			},
		},
		{
			name: "preserve resource requirements",
			config: []config.Option{
				config.WithAutoInstrumentationJavaImage("java-img:1"),
			},
			input: Instrumentation{
				Spec: InstrumentationSpec{
					Java: Java{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("32Mi"),
							},
						},
					},
				},
			},
			expected: Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/default-auto-instrumentation-java-image":         "java-img:1",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-nodejs-image":       "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-python-image":       "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-dotnet-image":       "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-go-image":           "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-nginx-image":        "",
						"instrumentation.opentelemetry.io/default-auto-instrumentation-apache-httpd-image": "",
					},
				},
				Spec: InstrumentationSpec{
					Java: Java{
						Image: "java-img:1",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("32Mi"),
							},
						},
					},
					NodeJS: NodeJS{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
					Python: Python{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
						},
					},
					DotNet: DotNet{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
					Go: Go{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("64Mi"),
							},
						},
					},
					ApacheHttpd: ApacheHttpd{
						Version:    "2.4",
						ConfigPath: "/usr/local/apache2/conf",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
					Nginx: Nginx{
						ConfigFile: "/etc/nginx/nginx.conf",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a copy of the input to avoid modifications between test cases
			input := test.input.DeepCopy()

			webhook := InstrumentationWebhook{
				cfg: config.New(test.config...),
			}

			err := webhook.Default(context.Background(), input)
			assert.NoError(t, err)

			if diff := cmp.Diff(test.expected, *input); diff != "" {
				t.Errorf("Default() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInstrumentationValidatingWebhook(t *testing.T) {
	tests := []struct {
		name     string
		err      string
		warnings admission.Warnings
		inst     Instrumentation
	}{
		{
			name: "all defaults",
			inst: Instrumentation{
				Spec: InstrumentationSpec{},
			},
			warnings: []string{"sampler type not set"},
		},
		{
			name: "sampler configuration not present",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Sampler: Sampler{},
				},
			},
			warnings: []string{"sampler type not set"},
		},
		{
			name: "argument is not a number",
			err:  "spec.sampler.argument is not a number",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Sampler: Sampler{
						Type:     ParentBasedTraceIDRatio,
						Argument: "abc",
					},
				},
			},
		},
		{
			name: "argument is a wrong number",
			err:  "spec.sampler.argument should be in rage [0..1]",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Sampler: Sampler{
						Type:     ParentBasedTraceIDRatio,
						Argument: "1.99",
					},
				},
			},
		},
		{
			name: "argument is a number",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Sampler: Sampler{
						Type:     ParentBasedTraceIDRatio,
						Argument: "0.99",
					},
				},
			},
		},
		{
			name: "argument is missing",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Sampler: Sampler{
						Type: ParentBasedTraceIDRatio,
					},
				},
			},
		},
		{
			name: "exporter: tls cert set but missing key",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Sampler: Sampler{
						Type:     ParentBasedTraceIDRatio,
						Argument: "0.99",
					},
					Exporter: Exporter{
						Endpoint: "https://collector:4317",
						TLS: &TLS{
							Cert: "cert",
						},
					},
				},
			},
			warnings: []string{"both exporter.tls.key and exporter.tls.cert mut be set"},
		},
		{
			name: "exporter: tls key set but missing cert",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Sampler: Sampler{
						Type:     ParentBasedTraceIDRatio,
						Argument: "0.99",
					},
					Exporter: Exporter{
						Endpoint: "https://collector:4317",
						TLS: &TLS{
							Key: "key",
						},
					},
				},
			},
			warnings: []string{"both exporter.tls.key and exporter.tls.cert mut be set"},
		},
		{
			name: "exporter: tls set but using http://",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Sampler: Sampler{
						Type:     ParentBasedTraceIDRatio,
						Argument: "0.99",
					},
					Exporter: Exporter{
						Endpoint: "http://collector:4317",
						TLS: &TLS{
							Key:  "key",
							Cert: "cert",
						},
					},
				},
			},
			warnings: []string{"exporter.tls is configured but exporter.endpoint is not enabling TLS with https://"},
		},
		{
			name: "exporter: exporter using http://, but the tls is nil",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Sampler: Sampler{
						Type:     ParentBasedTraceIDRatio,
						Argument: "0.99",
					},
					Exporter: Exporter{
						Endpoint: "https://collector:4317",
					},
				},
			},
			warnings: []string{"exporter is using https:// but exporter.tls is unset"},
		},
		{
			name: "exporter no warning set",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Sampler: Sampler{
						Type:     ParentBasedTraceIDRatio,
						Argument: "0.99",
					},
					Exporter: Exporter{
						Endpoint: "https://collector:4317",
						TLS: &TLS{
							Key:  "key",
							Cert: "cert",
						},
					},
				},
			},
		},
		{
			name: "sampler type is invalid",
			err:  "spec.sampler.type is not valid",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Sampler: Sampler{
						Type: "InvalidSamplerType",
					},
				},
			},
		},
		{
			name: "NodeJS with volume and volumeSizeLimit",
			err:  "spec.nodejs.volumeClaimTemplate and spec.nodejs.volumeSizeLimit cannot both be defined",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					NodeJS: NodeJS{
						VolumeClaimTemplate: corev1.PersistentVolumeClaimTemplate{
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							},
						},
						VolumeSizeLimit: &defaultVolumeSize,
					},
				},
			},
			warnings: []string{"sampler type not set"},
		},
		{
			name: "Java with volume and volumeSizeLimit",
			err:  "spec.java.volumeClaimTemplate and spec.java.volumeSizeLimit cannot both be defined",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Java: Java{
						VolumeClaimTemplate: corev1.PersistentVolumeClaimTemplate{
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							},
						},
						VolumeSizeLimit: &defaultVolumeSize,
					},
				},
			},
			warnings: []string{"sampler type not set"},
		},
		{
			name: "Python with volume and volumeSizeLimit",
			err:  "spec.python.volumeClaimTemplate and spec.python.volumeSizeLimit cannot both be defined",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Python: Python{
						VolumeClaimTemplate: corev1.PersistentVolumeClaimTemplate{
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							},
						},
						VolumeSizeLimit: &defaultVolumeSize,
					},
				},
			},
			warnings: []string{"sampler type not set"},
		},
		{
			name: "Go with volume and volumeSizeLimit",
			err:  "spec.go.volumeClaimTemplate and spec.go.volumeSizeLimit cannot both be defined",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Go: Go{
						VolumeClaimTemplate: corev1.PersistentVolumeClaimTemplate{
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							},
						},
						VolumeSizeLimit: &defaultVolumeSize,
					},
				},
			},
			warnings: []string{"sampler type not set"},
		},
		{
			name: "DotNet with volume and volumeSizeLimit",
			err:  "spec.dotnet.volumeClaimTemplate and spec.dotnet.volumeSizeLimit cannot both be defined",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					DotNet: DotNet{
						VolumeClaimTemplate: corev1.PersistentVolumeClaimTemplate{
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							},
						},
						VolumeSizeLimit: &defaultVolumeSize,
					},
				},
			},
			warnings: []string{"sampler type not set"},
		},
		{
			name: "ApacheHttpd with volume and volumeSizeLimit",
			err:  "spec.apachehttpd.volumeClaimTemplate and spec.apachehttpd.volumeSizeLimit cannot both be defined",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					ApacheHttpd: ApacheHttpd{
						VolumeClaimTemplate: corev1.PersistentVolumeClaimTemplate{
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							},
						},
						VolumeSizeLimit: &defaultVolumeSize,
					},
				},
			},
			warnings: []string{"sampler type not set"},
		},
		{
			name: "Nginx with volume and volumeSizeLimit",
			err:  "spec.nginx.volumeClaimTemplate and spec.nginx.volumeSizeLimit cannot both be defined",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Nginx: Nginx{
						VolumeClaimTemplate: corev1.PersistentVolumeClaimTemplate{
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							},
						},
						VolumeSizeLimit: &defaultVolumeSize,
					},
				},
			},
			warnings: []string{"sampler type not set"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			if test.err == "" {
				warnings, err := InstrumentationWebhook{}.ValidateCreate(ctx, &test.inst)
				assert.Equal(t, test.warnings, warnings)
				assert.Nil(t, err)
				warnings, err = InstrumentationWebhook{}.ValidateUpdate(ctx, nil, &test.inst)
				assert.Equal(t, test.warnings, warnings)
				assert.Nil(t, err)
				warnings, err = InstrumentationWebhook{}.ValidateDelete(ctx, &test.inst)
				assert.Equal(t, test.warnings, warnings)
				assert.Nil(t, err)
			} else {
				warnings, err := InstrumentationWebhook{}.ValidateCreate(ctx, &test.inst)
				assert.Equal(t, test.warnings, warnings)
				assert.Contains(t, err.Error(), test.err)
				warnings, err = InstrumentationWebhook{}.ValidateUpdate(ctx, nil, &test.inst)
				assert.Equal(t, test.warnings, warnings)
				assert.Contains(t, err.Error(), test.err)
				warnings, err = InstrumentationWebhook{}.ValidateDelete(ctx, &test.inst)
				assert.Equal(t, test.warnings, warnings)
				assert.Contains(t, err.Error(), test.err)
			}
		})
	}
}

func TestInstrumentationJaegerRemote(t *testing.T) {
	tests := []struct {
		name string
		err  string
		arg  string
	}{
		{
			name: "invalid format - missing equal sign",
			err:  "invalid argument",
			arg:  "endpoint http://jaeger-collector:14250/",
		},
		{
			name: "pollingIntervalMs is not a number",
			err:  "invalid pollingIntervalMs: abc",
			arg:  "pollingIntervalMs=abc",
		},
		{
			name: "initialSamplingRate is not a number",
			err:  "invalid initialSamplingRate",
			arg:  "initialSamplingRate=abc",
		},
		{
			name: "initialSamplingRate is negative",
			err:  "initialSamplingRate should be in rage [0..1]",
			arg:  "initialSamplingRate=-0.5",
		},
		{
			name: "initialSamplingRate is above 1",
			err:  "initialSamplingRate should be in rage [0..1]",
			arg:  "initialSamplingRate=1.99",
		},
		{
			name: "endpoint is missing",
			err:  "endpoint cannot be empty",
			arg:  "endpoint=",
		},
		{
			name: "minimal valid configuration with endpoint only",
			arg:  "endpoint=http://jaeger-collector:14250/",
		},
		{
			name: "with pollingIntervalMs only",
			arg:  "endpoint=http://jaeger-collector:14250/,pollingIntervalMs=2000",
		},
		{
			name: "with initialSamplingRate only",
			arg:  "endpoint=http://jaeger-collector:14250/,initialSamplingRate=0.5",
		},
		{
			name: "correct jaeger remote sampler configuration",
			arg:  "endpoint=http://jaeger-collector:14250/,initialSamplingRate=0.99,pollingIntervalMs=1000",
		},
	}

	samplers := []SamplerType{JaegerRemote, ParentBasedJaegerRemote}

	for _, sampler := range samplers {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				inst := Instrumentation{
					Spec: InstrumentationSpec{
						Sampler: Sampler{
							Type:     sampler,
							Argument: test.arg,
						},
					},
				}
				ctx := context.Background()
				if test.err == "" {
					warnings, err := InstrumentationWebhook{}.ValidateCreate(ctx, &inst)
					assert.Nil(t, warnings)
					assert.Nil(t, err)
					warnings, err = InstrumentationWebhook{}.ValidateUpdate(ctx, nil, &inst)
					assert.Nil(t, warnings)
					assert.Nil(t, err)
					warnings, err = InstrumentationWebhook{}.ValidateDelete(ctx, &inst)
					assert.Nil(t, warnings)
					assert.Nil(t, err)
				} else {
					warnings, err := InstrumentationWebhook{}.ValidateCreate(ctx, &inst)
					assert.Nil(t, warnings)
					assert.Contains(t, err.Error(), test.err)
					warnings, err = InstrumentationWebhook{}.ValidateUpdate(ctx, nil, &inst)
					assert.Nil(t, warnings)
					assert.Contains(t, err.Error(), test.err)
					warnings, err = InstrumentationWebhook{}.ValidateDelete(ctx, &inst)
					assert.Nil(t, warnings)
					assert.Contains(t, err.Error(), test.err)
				}
			})
		}
	}
}

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

var defaultVolumeSize = resource.MustParse("200Mi")

func TestInstrumentationDefaultingWebhook(t *testing.T) {
	type testCase struct {
		name   string
		input  *Instrumentation
		config []config.Option
		verify func(t *testing.T, inst *Instrumentation)
	}

	tests := []testCase{
		{
			name:  "default images",
			input: &Instrumentation{},
			config: []config.Option{
				config.WithAutoInstrumentationJavaImage("java-img:1"),
				config.WithAutoInstrumentationNodeJSImage("nodejs-img:1"),
				config.WithAutoInstrumentationPythonImage("python-img:1"),
				config.WithAutoInstrumentationDotNetImage("dotnet-img:1"),
				config.WithAutoInstrumentationGoImage("go-img:1"),
				config.WithAutoInstrumentationNginxImage("nginx-img:1"),
				config.WithAutoInstrumentationApacheHttpdImage("apache-httpd-img:1"),
			},
			verify: func(t *testing.T, inst *Instrumentation) {
				assert.Equal(t, "java-img:1", inst.Spec.Java.Image)
				assert.Equal(t, "nodejs-img:1", inst.Spec.NodeJS.Image)
				assert.Equal(t, "python-img:1", inst.Spec.Python.Image)
				assert.Equal(t, "dotnet-img:1", inst.Spec.DotNet.Image)
				assert.Equal(t, "go-img:1", inst.Spec.Go.Image)
				assert.Equal(t, "nginx-img:1", inst.Spec.Nginx.Image)
				assert.Equal(t, "apache-httpd-img:1", inst.Spec.ApacheHttpd.Image)

				assert.Equal(t, "java-img:1", inst.ObjectMeta.Annotations["instrumentation.opentelemetry.io/default-auto-instrumentation-java-image"])
				assert.Equal(t, "nodejs-img:1", inst.ObjectMeta.Annotations["instrumentation.opentelemetry.io/default-auto-instrumentation-nodejs-image"])
				assert.Equal(t, "python-img:1", inst.ObjectMeta.Annotations["instrumentation.opentelemetry.io/default-auto-instrumentation-python-image"])
				assert.Equal(t, "dotnet-img:1", inst.ObjectMeta.Annotations["instrumentation.opentelemetry.io/default-auto-instrumentation-dotnet-image"])
				assert.Equal(t, "go-img:1", inst.ObjectMeta.Annotations["instrumentation.opentelemetry.io/default-auto-instrumentation-go-image"])
				assert.Equal(t, "nginx-img:1", inst.ObjectMeta.Annotations["instrumentation.opentelemetry.io/default-auto-instrumentation-nginx-image"])
				assert.Equal(t, "apache-httpd-img:1", inst.ObjectMeta.Annotations["instrumentation.opentelemetry.io/default-auto-instrumentation-apache-httpd-image"])
			},
		},
		{
			name: "do not override custom image",
			input: &Instrumentation{
				Spec: InstrumentationSpec{
					Java: Java{
						Image: "custom-java-img:2",
					},
					NodeJS: NodeJS{
						Image: "custom-nodejs-img:2",
					},
					Python: Python{
						Image: "custom-python-img:2",
					},
					DotNet: DotNet{
						Image: "custom-dotnet-img:2",
					},
					Go: Go{
						Image: "custom-go-img:2",
					},
					Nginx: Nginx{
						Image: "custom-nginx-img:2",
					},
					ApacheHttpd: ApacheHttpd{
						Image: "custom-apache-httpd-img:2",
					},
				},
			},
			config: []config.Option{
				config.WithAutoInstrumentationJavaImage("java-img:1"),
				config.WithAutoInstrumentationNodeJSImage("nodejs-img:1"),
				config.WithAutoInstrumentationPythonImage("python-img:1"),
				config.WithAutoInstrumentationDotNetImage("dotnet-img:1"),
				config.WithAutoInstrumentationGoImage("go-img:1"),
				config.WithAutoInstrumentationNginxImage("nginx-img:1"),
				config.WithAutoInstrumentationApacheHttpdImage("apache-httpd-img:1"),
			},
			verify: func(t *testing.T, inst *Instrumentation) {
				assert.Equal(t, "custom-java-img:2", inst.Spec.Java.Image)
				assert.Equal(t, "custom-nodejs-img:2", inst.Spec.NodeJS.Image)
				assert.Equal(t, "custom-python-img:2", inst.Spec.Python.Image)
				assert.Equal(t, "custom-dotnet-img:2", inst.Spec.DotNet.Image)
				assert.Equal(t, "custom-go-img:2", inst.Spec.Go.Image)
				assert.Equal(t, "custom-nginx-img:2", inst.Spec.Nginx.Image)
				assert.Equal(t, "custom-apache-httpd-img:2", inst.Spec.ApacheHttpd.Image)

				assert.Equal(t, "java-img:1", inst.ObjectMeta.Annotations["instrumentation.opentelemetry.io/default-auto-instrumentation-java-image"])
				assert.Equal(t, "nodejs-img:1", inst.ObjectMeta.Annotations["instrumentation.opentelemetry.io/default-auto-instrumentation-nodejs-image"])
				assert.Equal(t, "python-img:1", inst.ObjectMeta.Annotations["instrumentation.opentelemetry.io/default-auto-instrumentation-python-image"])
				assert.Equal(t, "dotnet-img:1", inst.ObjectMeta.Annotations["instrumentation.opentelemetry.io/default-auto-instrumentation-dotnet-image"])
				assert.Equal(t, "go-img:1", inst.ObjectMeta.Annotations["instrumentation.opentelemetry.io/default-auto-instrumentation-go-image"])
				assert.Equal(t, "nginx-img:1", inst.ObjectMeta.Annotations["instrumentation.opentelemetry.io/default-auto-instrumentation-nginx-image"])
				assert.Equal(t, "apache-httpd-img:1", inst.ObjectMeta.Annotations["instrumentation.opentelemetry.io/default-auto-instrumentation-apache-httpd-image"])
			},
		},
		{
			name:   "default resource and config settings",
			input:  &Instrumentation{},
			config: []config.Option{},
			verify: func(t *testing.T, inst *Instrumentation) {
				assert.Equal(t, resource.MustParse("500m"), inst.Spec.Java.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("64Mi"), inst.Spec.Java.Resources.Limits[corev1.ResourceMemory])
				assert.Equal(t, resource.MustParse("50m"), inst.Spec.Java.Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("64Mi"), inst.Spec.Java.Resources.Requests[corev1.ResourceMemory])

				assert.Equal(t, resource.MustParse("500m"), inst.Spec.NodeJS.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("128Mi"), inst.Spec.NodeJS.Resources.Limits[corev1.ResourceMemory])
				assert.Equal(t, resource.MustParse("50m"), inst.Spec.NodeJS.Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("128Mi"), inst.Spec.NodeJS.Resources.Requests[corev1.ResourceMemory])

				assert.Equal(t, resource.MustParse("500m"), inst.Spec.Python.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("64Mi"), inst.Spec.Python.Resources.Limits[corev1.ResourceMemory])
				assert.Equal(t, resource.MustParse("50m"), inst.Spec.Python.Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("64Mi"), inst.Spec.Python.Resources.Requests[corev1.ResourceMemory])

				assert.Equal(t, resource.MustParse("500m"), inst.Spec.DotNet.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("128Mi"), inst.Spec.DotNet.Resources.Limits[corev1.ResourceMemory])
				assert.Equal(t, resource.MustParse("50m"), inst.Spec.DotNet.Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("128Mi"), inst.Spec.DotNet.Resources.Requests[corev1.ResourceMemory])

				assert.Equal(t, resource.MustParse("500m"), inst.Spec.Go.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("64Mi"), inst.Spec.Go.Resources.Limits[corev1.ResourceMemory])
				assert.Equal(t, resource.MustParse("50m"), inst.Spec.Go.Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("64Mi"), inst.Spec.Go.Resources.Requests[corev1.ResourceMemory])

				assert.Equal(t, resource.MustParse("500m"), inst.Spec.Nginx.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("128Mi"), inst.Spec.Nginx.Resources.Limits[corev1.ResourceMemory])
				assert.Equal(t, resource.MustParse("1m"), inst.Spec.Nginx.Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("128Mi"), inst.Spec.Nginx.Resources.Requests[corev1.ResourceMemory])

				assert.Equal(t, resource.MustParse("500m"), inst.Spec.ApacheHttpd.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("128Mi"), inst.Spec.ApacheHttpd.Resources.Limits[corev1.ResourceMemory])
				assert.Equal(t, resource.MustParse("1m"), inst.Spec.ApacheHttpd.Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("128Mi"), inst.Spec.ApacheHttpd.Resources.Requests[corev1.ResourceMemory])

				assert.Equal(t, "/etc/nginx/nginx.conf", inst.Spec.Nginx.ConfigFile)
				assert.Equal(t, "/usr/local/apache2/conf", inst.Spec.ApacheHttpd.ConfigPath)
				assert.Equal(t, "2.4", inst.Spec.ApacheHttpd.Version)
			},
		},
		{
			name: "preserve custom requirements and config settings",
			input: &Instrumentation{
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
					NodeJS: NodeJS{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("300m"),
								corev1.ResourceMemory: resource.MustParse("200Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("200m"),
								corev1.ResourceMemory: resource.MustParse("100Mi"),
							},
						},
					},
					Python: Python{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("400m"),
								corev1.ResourceMemory: resource.MustParse("256Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("150m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
					DotNet: DotNet{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("600m"),
								corev1.ResourceMemory: resource.MustParse("400Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("250m"),
								corev1.ResourceMemory: resource.MustParse("200Mi"),
							},
						},
					},
					Go: Go{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("700m"),
								corev1.ResourceMemory: resource.MustParse("150Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("350m"),
								corev1.ResourceMemory: resource.MustParse("75Mi"),
							},
						},
					},
					Nginx: Nginx{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("800m"),
								corev1.ResourceMemory: resource.MustParse("256Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("400m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
						ConfigFile: "/custom/path/nginx.conf",
					},
					ApacheHttpd: ApacheHttpd{
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("900m"),
								corev1.ResourceMemory: resource.MustParse("300Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("450m"),
								corev1.ResourceMemory: resource.MustParse("150Mi"),
							},
						},
						ConfigPath: "/custom/apache/conf",
						Version:    "2.5",
					},
				},
			},
			config: []config.Option{},
			verify: func(t *testing.T, inst *Instrumentation) {
				assert.Equal(t, resource.MustParse("500m"), inst.Spec.Java.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("128Mi"), inst.Spec.Java.Resources.Limits[corev1.ResourceMemory])
				assert.Equal(t, resource.MustParse("100m"), inst.Spec.Java.Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("32Mi"), inst.Spec.Java.Resources.Requests[corev1.ResourceMemory])

				assert.Equal(t, resource.MustParse("300m"), inst.Spec.NodeJS.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("200Mi"), inst.Spec.NodeJS.Resources.Limits[corev1.ResourceMemory])
				assert.Equal(t, resource.MustParse("200m"), inst.Spec.NodeJS.Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("100Mi"), inst.Spec.NodeJS.Resources.Requests[corev1.ResourceMemory])

				assert.Equal(t, resource.MustParse("400m"), inst.Spec.Python.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("256Mi"), inst.Spec.Python.Resources.Limits[corev1.ResourceMemory])
				assert.Equal(t, resource.MustParse("150m"), inst.Spec.Python.Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("128Mi"), inst.Spec.Python.Resources.Requests[corev1.ResourceMemory])

				assert.Equal(t, resource.MustParse("600m"), inst.Spec.DotNet.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("400Mi"), inst.Spec.DotNet.Resources.Limits[corev1.ResourceMemory])
				assert.Equal(t, resource.MustParse("250m"), inst.Spec.DotNet.Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("200Mi"), inst.Spec.DotNet.Resources.Requests[corev1.ResourceMemory])

				assert.Equal(t, resource.MustParse("700m"), inst.Spec.Go.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("150Mi"), inst.Spec.Go.Resources.Limits[corev1.ResourceMemory])
				assert.Equal(t, resource.MustParse("350m"), inst.Spec.Go.Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("75Mi"), inst.Spec.Go.Resources.Requests[corev1.ResourceMemory])

				assert.Equal(t, resource.MustParse("800m"), inst.Spec.Nginx.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("256Mi"), inst.Spec.Nginx.Resources.Limits[corev1.ResourceMemory])
				assert.Equal(t, resource.MustParse("400m"), inst.Spec.Nginx.Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("128Mi"), inst.Spec.Nginx.Resources.Requests[corev1.ResourceMemory])
				assert.Equal(t, "/custom/path/nginx.conf", inst.Spec.Nginx.ConfigFile)

				assert.Equal(t, resource.MustParse("900m"), inst.Spec.ApacheHttpd.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("300Mi"), inst.Spec.ApacheHttpd.Resources.Limits[corev1.ResourceMemory])
				assert.Equal(t, resource.MustParse("450m"), inst.Spec.ApacheHttpd.Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("150Mi"), inst.Spec.ApacheHttpd.Resources.Requests[corev1.ResourceMemory])
				assert.Equal(t, "/custom/apache/conf", inst.Spec.ApacheHttpd.ConfigPath)
				assert.Equal(t, "2.5", inst.Spec.ApacheHttpd.Version)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			webhook := InstrumentationWebhook{
				cfg: config.New(test.config...),
			}

			err := webhook.Default(context.Background(), test.input)
			assert.NoError(t, err)

			test.verify(t, test.input)
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

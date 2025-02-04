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
	inst := &Instrumentation{}
	err := InstrumentationWebhook{
		cfg: config.New(
			config.WithAutoInstrumentationJavaImage("java-img:1"),
			config.WithAutoInstrumentationNodeJSImage("nodejs-img:1"),
			config.WithAutoInstrumentationPythonImage("python-img:1"),
			config.WithAutoInstrumentationDotNetImage("dotnet-img:1"),
			config.WithAutoInstrumentationApacheHttpdImage("apache-httpd-img:1"),
			config.WithAutoInstrumentationNginxImage("nginx-img:1"),
		),
	}.Default(context.Background(), inst)
	assert.NoError(t, err)
	assert.Equal(t, "java-img:1", inst.Spec.Java.Image)
	assert.Equal(t, "nodejs-img:1", inst.Spec.NodeJS.Image)
	assert.Equal(t, "python-img:1", inst.Spec.Python.Image)
	assert.Equal(t, "dotnet-img:1", inst.Spec.DotNet.Image)
	assert.Equal(t, "apache-httpd-img:1", inst.Spec.ApacheHttpd.Image)
	assert.Equal(t, "nginx-img:1", inst.Spec.Nginx.Image)
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
			name: "with volume and volumeSizeLimit",
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
			} else {
				warnings, err := InstrumentationWebhook{}.ValidateCreate(ctx, &test.inst)
				assert.Equal(t, test.warnings, warnings)
				assert.Contains(t, err.Error(), test.err)
				warnings, err = InstrumentationWebhook{}.ValidateUpdate(ctx, nil, &test.inst)
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
			name: "pollingIntervalMs is not a number",
			err:  "invalid pollingIntervalMs: abc",
			arg:  "pollingIntervalMs=abc",
		},
		{
			name: "initialSamplingRate is out of range",
			err:  "initialSamplingRate should be in rage [0..1]",
			arg:  "initialSamplingRate=1.99",
		},
		{
			name: "endpoint is missing",
			err:  "endpoint cannot be empty",
			arg:  "endpoint=",
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
				} else {
					warnings, err := InstrumentationWebhook{}.ValidateCreate(ctx, &inst)
					assert.Nil(t, warnings)
					assert.Contains(t, err.Error(), test.err)
					warnings, err = InstrumentationWebhook{}.ValidateUpdate(ctx, nil, &inst)
					assert.Nil(t, warnings)
					assert.Contains(t, err.Error(), test.err)
				}
			})
		}
	}
}

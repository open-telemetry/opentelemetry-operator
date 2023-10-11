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

package instrumentation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestInstrumentationDefaultingWebhook(t *testing.T) {
	inst := &v1alpha1.Instrumentation{}
	err := Webhook{
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
		inst     v1alpha1.Instrumentation
	}{
		{
			name: "all defaults",
			inst: v1alpha1.Instrumentation{
				Spec: v1alpha1.InstrumentationSpec{},
			},
			warnings: []string{"sampler type not set"},
		},
		{
			name: "sampler configuration not present",
			inst: v1alpha1.Instrumentation{
				Spec: v1alpha1.InstrumentationSpec{
					Sampler: v1alpha1.Sampler{},
				},
			},
			warnings: []string{"sampler type not set"},
		},
		{
			name: "argument is not a number",
			err:  "spec.sampler.argument is not a number",
			inst: v1alpha1.Instrumentation{
				Spec: v1alpha1.InstrumentationSpec{
					Sampler: v1alpha1.Sampler{
						Type:     v1alpha1.ParentBasedTraceIDRatio,
						Argument: "abc",
					},
				},
			},
		},
		{
			name: "argument is a wrong number",
			err:  "spec.sampler.argument should be in rage [0..1]",
			inst: v1alpha1.Instrumentation{
				Spec: v1alpha1.InstrumentationSpec{
					Sampler: v1alpha1.Sampler{
						Type:     v1alpha1.ParentBasedTraceIDRatio,
						Argument: "1.99",
					},
				},
			},
		},
		{
			name: "argument is a number",
			inst: v1alpha1.Instrumentation{
				Spec: v1alpha1.InstrumentationSpec{
					Sampler: v1alpha1.Sampler{
						Type:     v1alpha1.ParentBasedTraceIDRatio,
						Argument: "0.99",
					},
				},
			},
		},
		{
			name: "argument is missing",
			inst: v1alpha1.Instrumentation{
				Spec: v1alpha1.InstrumentationSpec{
					Sampler: v1alpha1.Sampler{
						Type: v1alpha1.ParentBasedTraceIDRatio,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			if test.err == "" {
				warnings, err := Webhook{}.ValidateCreate(ctx, &test.inst)
				assert.Equal(t, test.warnings, warnings)
				assert.Nil(t, err)
				warnings, err = Webhook{}.ValidateUpdate(ctx, nil, &test.inst)
				assert.Equal(t, test.warnings, warnings)
				assert.Nil(t, err)
			} else {
				warnings, err := Webhook{}.ValidateCreate(ctx, &test.inst)
				assert.Equal(t, test.warnings, warnings)
				assert.Contains(t, err.Error(), test.err)
				warnings, err = Webhook{}.ValidateUpdate(ctx, nil, &test.inst)
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

	samplers := []v1alpha1.SamplerType{v1alpha1.JaegerRemote, v1alpha1.ParentBasedJaegerRemote}

	for _, sampler := range samplers {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				inst := v1alpha1.Instrumentation{
					Spec: v1alpha1.InstrumentationSpec{
						Sampler: v1alpha1.Sampler{
							Type:     sampler,
							Argument: test.arg,
						},
					},
				}
				ctx := context.Background()
				if test.err == "" {
					warnings, err := Webhook{}.ValidateCreate(ctx, &inst)
					assert.Nil(t, warnings)
					assert.Nil(t, err)
					warnings, err = Webhook{}.ValidateUpdate(ctx, nil, &inst)
					assert.Nil(t, warnings)
					assert.Nil(t, err)
				} else {
					warnings, err := Webhook{}.ValidateCreate(ctx, &inst)
					assert.Nil(t, warnings)
					assert.Contains(t, err.Error(), test.err)
					warnings, err = Webhook{}.ValidateUpdate(ctx, nil, &inst)
					assert.Nil(t, warnings)
					assert.Contains(t, err.Error(), test.err)
				}
			})
		}
	}
}

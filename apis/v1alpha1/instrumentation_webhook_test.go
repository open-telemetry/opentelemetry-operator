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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInstrumentationDefaultingWebhook(t *testing.T) {
	inst := &Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				AnnotationDefaultAutoInstrumentationJava:        "java-img:1",
				AnnotationDefaultAutoInstrumentationNodeJS:      "nodejs-img:1",
				AnnotationDefaultAutoInstrumentationPython:      "python-img:1",
				AnnotationDefaultAutoInstrumentationDotNet:      "dotnet-img:1",
				AnnotationDefaultAutoInstrumentationApacheHttpd: "apache-httpd-img:1",
			},
		},
	}
	inst.Default()
	assert.Equal(t, "java-img:1", inst.Spec.Java.Image)
	assert.Equal(t, "nodejs-img:1", inst.Spec.NodeJS.Image)
	assert.Equal(t, "python-img:1", inst.Spec.Python.Image)
	assert.Equal(t, "dotnet-img:1", inst.Spec.DotNet.Image)
	assert.Equal(t, "apache-httpd-img:1", inst.Spec.ApacheHttpd.Image)
}

func TestInstrumentationValidatingWebhook(t *testing.T) {
	tests := []struct {
		name string
		err  string
		inst Instrumentation
	}{
		{
			name: "all defaults",
			inst: Instrumentation{
				Spec: InstrumentationSpec{},
			},
		},
		{
			name: "sampler configuration not present",
			inst: Instrumentation{
				Spec: InstrumentationSpec{
					Sampler: Sampler{},
				},
			},
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.err == "" {
				warnings, err := test.inst.ValidateCreate()
				assert.Nil(t, warnings)
				assert.Nil(t, err)
				warnings, err = test.inst.ValidateUpdate(nil)
				assert.Nil(t, warnings)
				assert.Nil(t, err)
			} else {
				warnings, err := test.inst.ValidateCreate()
				assert.Nil(t, warnings)
				assert.Contains(t, err.Error(), test.err)
				warnings, err = test.inst.ValidateUpdate(nil)
				assert.Nil(t, warnings)
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
				if test.err == "" {
					warnings, err := inst.ValidateCreate()
					assert.Nil(t, warnings)
					assert.Nil(t, err)
					warnings, err = inst.ValidateUpdate(nil)
					assert.Nil(t, warnings)
					assert.Nil(t, err)
				} else {
					warnings, err := inst.ValidateCreate()
					assert.Nil(t, warnings)
					assert.Contains(t, err.Error(), test.err)
					warnings, err = inst.ValidateUpdate(nil)
					assert.Nil(t, warnings)
					assert.Contains(t, err.Error(), test.err)
				}
			})
		}
	}
}

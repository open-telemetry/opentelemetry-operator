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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInstrumentationDefaultingWebhook(t *testing.T) {
	inst := &Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				AnnotationDefaultAutoInstrumentationJava:   "java-img:1",
				AnnotationDefaultAutoInstrumentationNodeJS: "nodejs-img:1",
				AnnotationDefaultAutoInstrumentationPython: "python-img:1",
				AnnotationDefaultAutoInstrumentationDotNet: "dotnet-img:1",
			},
		},
	}
	inst.Default()
	assert.Equal(t, "java-img:1", inst.Spec.Java.Image)
	assert.Equal(t, "nodejs-img:1", inst.Spec.NodeJS.Image)
	assert.Equal(t, "python-img:1", inst.Spec.Python.Image)
	assert.Equal(t, "dotnet-img:1", inst.Spec.DotNet.Image)
	assert.Equal(t, true, inst.isEnvVarSet(envOtelDotnetAutoTracesEnabledInstrumentations))
}

func TestInstrumentationDefaultingWebhookOtelDotNetTracesEnabledInstruEnvSet(t *testing.T) {
	inst := &Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				AnnotationDefaultAutoInstrumentationJava:   "java-img:1",
				AnnotationDefaultAutoInstrumentationNodeJS: "nodejs-img:1",
				AnnotationDefaultAutoInstrumentationPython: "python-img:1",
				AnnotationDefaultAutoInstrumentationDotNet: "dotnet-img:1",
			},
		},
		Spec: InstrumentationSpec{
			DotNet: DotNet{
				Env: []v1.EnvVar{
					{
						Name:  envOtelDotnetAutoTracesEnabledInstrumentations,
						Value: "AspNet,HttpClient",
					},
				},
			},
		},
	}
	inst.Default()
	for _, env := range inst.Spec.DotNet.Env {
		if env.Name == envOtelDotnetAutoTracesEnabledInstrumentations {
			assert.Equal(t, "AspNet,HttpClient", env.Value)
			break
		}
	}
}

func TestInstrumentationValidatingWebhook(t *testing.T) {
	tests := []struct {
		name string
		err  string
		inst Instrumentation
	}{
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
				assert.Nil(t, test.inst.ValidateCreate())
				assert.Nil(t, test.inst.ValidateUpdate(nil))
			} else {
				err := test.inst.ValidateCreate()
				assert.Contains(t, err.Error(), test.err)
				err = test.inst.ValidateUpdate(nil)
				assert.Contains(t, err.Error(), test.err)
			}
		})
	}
}

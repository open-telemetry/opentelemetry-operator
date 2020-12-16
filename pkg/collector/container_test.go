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

package collector_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

var logger = logf.Log.WithName("unit-tests")

func TestContainerNewDefault(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{}
	cfg := config.New(config.WithCollectorImage("default-image"))

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Equal(t, "default-image", c.Image)
}

func TestContainerWithImageOverridden(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Image: "overridden-image",
		},
	}
	cfg := config.New(config.WithCollectorImage("default-image"))

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Equal(t, "overridden-image", c.Image)
}

func TestContainerConfigFlagIsIgnored(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Args: map[string]string{
				"key":    "value",
				"config": "/some-custom-file.yaml",
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Len(t, c.Args, 2)
	assert.Contains(t, c.Args, "--key=value")
	assert.NotContains(t, c.Args, "--config=/some-custom-file.yaml")
}

func TestContainerCustomVolumes(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			VolumeMounts: []corev1.VolumeMount{{
				Name: "custom-volume-mount",
			}},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Len(t, c.VolumeMounts, 2)
	assert.Equal(t, "custom-volume-mount", c.VolumeMounts[1].Name)
}

func TestContainerEnvVarsOverridden(t *testing.T) {
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Env: []corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
			},
		},
	}

	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Len(t, c.Env, 1)
	assert.Equal(t, "foo", c.Env[0].Name)
	assert.Equal(t, "bar", c.Env[0].Value)
}

func TestContainerEmptyEnvVarsByDefault(t *testing.T) {
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{},
	}

	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Empty(t, c.Env)
}

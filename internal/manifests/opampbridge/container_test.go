// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opampbridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var logger = logf.Log.WithName("unit-tests")

func TestContainerNewDefault(t *testing.T) {
	// prepare
	opampBridge := v1alpha1.OpAMPBridge{}
	cfg := config.Config{
		OperatorOpAMPBridgeImage: "default-image",
	}

	// test
	c := Container(cfg, logger, opampBridge)

	// verify
	assert.Equal(t, "default-image", c.Image)
}

func TestContainerWithImageOverridden(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpAMPBridge{
		Spec: v1alpha1.OpAMPBridgeSpec{
			Image: "overridden-image",
		},
	}

	cfg := config.Config{
		OperatorOpAMPBridgeImage: "default-image",
	}

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Equal(t, "overridden-image", c.Image)
}

func TestContainerVolumes(t *testing.T) {
	// prepare
	opampBridge := v1alpha1.OpAMPBridge{
		Spec: v1alpha1.OpAMPBridgeSpec{
			Image: "default-image",
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, opampBridge)

	// verify
	assert.Len(t, c.VolumeMounts, 1)
	assert.Equal(t, naming.OpAMPBridgeConfigMapVolume(), c.VolumeMounts[0].Name)
}

// Regression test: when Spec.Env has spare backing-array capacity,
// the container's Env must not share the underlying array with the spec.
func TestContainerEnvAliasing(t *testing.T) {
	env := make([]corev1.EnvVar, 0, 10)
	env = append(env, corev1.EnvVar{Name: "USER_VAR", Value: "val"})

	opampBridge := v1alpha1.OpAMPBridge{
		Spec: v1alpha1.OpAMPBridgeSpec{
			Env: env,
		},
	}
	cfg := config.New()

	c := Container(cfg, logger, opampBridge)

	// Mutate the original spec — container must not be affected.
	opampBridge.Spec.Env = append(opampBridge.Spec.Env,
		corev1.EnvVar{Name: "intruder", Value: "bad"})

	for _, e := range c.Env {
		assert.NotEqual(t, "intruder", e.Name,
			"container Env shares backing array with spec")
	}
}

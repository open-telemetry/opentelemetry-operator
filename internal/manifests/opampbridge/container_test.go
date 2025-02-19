// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opampbridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var logger = logf.Log.WithName("unit-tests")

func TestContainerNewDefault(t *testing.T) {
	// prepare
	opampBridge := v1alpha1.OpAMPBridge{}
	cfg := config.New(config.WithOperatorOpAMPBridgeImage("default-image"))

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

	cfg := config.New(config.WithOperatorOpAMPBridgeImage("default-image"))

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

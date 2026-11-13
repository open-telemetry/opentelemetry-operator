// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opampbridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func TestVolumeNewDefault(t *testing.T) {
	// prepare
	opampBridge := v1alpha1.OpAMPBridge{}
	cfg := config.New()

	// test
	volumes := Volumes(cfg, opampBridge)

	// verify
	assert.Len(t, volumes, 1)

	// check if the number of elements in the volume source items list is 1
	assert.Len(t, volumes[0].ConfigMap.Items, 1)

	// check that it's the opamp-bridge-internal volume, with the config map
	assert.Equal(t, naming.OpAMPBridgeConfigMapVolume(), volumes[0].Name)
}

func TestUserDefinedVolume(t *testing.T) {
	opampBridge := v1alpha1.OpAMPBridge{
		Spec: v1alpha1.OpAMPBridgeSpec{
			Volumes: []corev1.Volume{
				{
					Name: "custom-volume",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
	cfg := config.New()

	volumes := Volumes(cfg, opampBridge)

	assert.Len(t, volumes, 2)
	assert.Contains(t, volumes, opampBridge.Spec.Volumes[0])
}

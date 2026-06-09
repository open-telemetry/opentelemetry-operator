// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clusterobservability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/clusterobservability/config"
)

func TestGetCollectorImage(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "rewrites tagged release image to contrib",
			in:   "ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector:0.151.0",
			want: "ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:0.151.0",
		},
		{
			name: "dev-build placeholder maps to contrib:latest",
			in:   "ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector:0.0.0",
			want: "ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:latest",
		},
		{
			name: "already-contrib image is unchanged",
			in:   "ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:0.151.0",
			want: "ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:0.151.0",
		},
		{
			name: "custom registry is rewritten",
			in:   "registry.example.com/otel/opentelemetry-collector:1.2.3",
			want: "registry.example.com/otel/opentelemetry-collector-contrib:1.2.3",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, getCollectorImage(tc.in))
		})
	}
}

func TestAgentVolumesAndMounts_DistroConditional(t *testing.T) {
	const kubeletCAName = "kubelet-serving-ca"

	t.Run("non-openshift distros omit kubelet-serving-ca", func(t *testing.T) {
		volumes := agentVolumes(config.DistroProvider(""))
		mounts := agentVolumeMounts(config.DistroProvider(""))
		assert.NotContains(t, volumeNames(volumes), kubeletCAName)
		assert.NotContains(t, mountNames(mounts), kubeletCAName)
	})

	t.Run("openshift mounts kubelet-serving-ca as HostPathFile", func(t *testing.T) {
		volumes := agentVolumes(config.OpenShift)
		mounts := agentVolumeMounts(config.OpenShift)
		assert.Contains(t, volumeNames(volumes), kubeletCAName)
		assert.Contains(t, mountNames(mounts), kubeletCAName)

		v := findVolume(t, volumes, kubeletCAName)
		require.NotNil(t, v.HostPath, "%s must be a HostPath volume", kubeletCAName)
		assert.Equal(t, "/etc/kubernetes/kubelet-ca.crt", v.HostPath.Path)
		require.NotNil(t, v.HostPath.Type, "HostPath type must be set so the mount fails fast if missing")
		assert.Equal(t, corev1.HostPathFile, *v.HostPath.Type)

		m := findMount(t, mounts, kubeletCAName)
		assert.True(t, m.ReadOnly)
	})

	t.Run("base host volumes are present in every distro", func(t *testing.T) {
		baseHostMounts := []string{"host-dev", "host-etc", "host-proc", "host-sys", "var-log-pods"}
		for _, distro := range []config.DistroProvider{"", config.OpenShift} {
			names := volumeNames(agentVolumes(distro))
			for _, want := range baseHostMounts {
				assert.Contains(t, names, want, "distro=%q missing %s", distro, want)
			}
			assert.Equal(t, len(agentVolumes(distro)), len(agentVolumeMounts(distro)),
				"volumes and mounts must be paired 1:1")
		}
	})
}

func volumeNames(vs []corev1.Volume) []string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.Name)
	}
	return out
}

func mountNames(ms []corev1.VolumeMount) []string {
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.Name)
	}
	return out
}

func findVolume(t *testing.T, vs []corev1.Volume, name string) corev1.Volume {
	t.Helper()
	for _, v := range vs {
		if v.Name == name {
			return v
		}
	}
	t.Fatalf("volume %q not found", name)
	return corev1.Volume{}
}

func findMount(t *testing.T, ms []corev1.VolumeMount, name string) corev1.VolumeMount {
	t.Helper()
	for _, m := range ms {
		if m.Name == name {
			return m
		}
	}
	t.Fatalf("mount %q not found", name)
	return corev1.VolumeMount{}
}

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clusterobservability

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	operatorcfg "github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/clusterobservability/config"
)

// The ClusterObservability collectors default to the k8s distribution at the same
// version as the operator-wide collector image (sourced from versions.txt).
func TestClusterObservabilityCollectorImage_Default(t *testing.T) {
	cfg := operatorcfg.New()
	want := strings.Replace(cfg.CollectorImage, "/opentelemetry-collector:", "/opentelemetry-collector-k8s:", 1)
	assert.Equal(t, want, cfg.ClusterObservabilityCollectorImage)
	assert.Contains(t, cfg.ClusterObservabilityCollectorImage, "opentelemetry-collector-k8s:")
}

func TestBuildCollectors_UseClusterObservabilityCollectorImage(t *testing.T) {
	const img = "example.com/otel/opentelemetry-collector-k8s:9.9.9"
	params := manifests.Params{
		ClusterObservability: v1alpha1.ClusterObservability{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster-obs", Namespace: "observability"},
		},
		Config: operatorcfg.Config{ClusterObservabilityCollectorImage: img},
	}

	agent, err := buildAgentCollector(params, config.DistroProvider(""))
	require.NoError(t, err)
	assert.Equal(t, img, agent.Spec.Image)

	cluster, err := buildClusterCollector(params)
	require.NoError(t, err)
	assert.Equal(t, img, cluster.Spec.Image)
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

	t.Run("host root is mounted at /hostfs for hostmetrics", func(t *testing.T) {
		for _, distro := range []config.DistroProvider{"", config.OpenShift} {
			volumes := agentVolumes(distro)
			mounts := agentVolumeMounts(distro)

			v := findVolume(t, volumes, "hostfs")
			require.NotNil(t, v.HostPath, "hostfs must be a HostPath volume")
			assert.Equal(t, "/", v.HostPath.Path, "distro=%q hostfs must mount host root", distro)

			m := findMount(t, mounts, "hostfs")
			assert.Equal(t, "/hostfs", m.MountPath)
			assert.True(t, m.ReadOnly)
			require.NotNil(t, m.MountPropagation)
			assert.Equal(t, corev1.MountPropagationHostToContainer, *m.MountPropagation)

			// The granular per-directory host mounts (and the os-release special
			// case) are replaced by the single host-root mount.
			for _, gone := range []string{"host-dev", "host-etc", "host-proc", "host-sys", "host-usr-lib-osrelease"} {
				assert.NotContains(t, volumeNames(volumes), gone, "distro=%q must not mount %s", distro, gone)
			}
		}
	})

	t.Run("base volumes are present and paired in every distro", func(t *testing.T) {
		baseHostMounts := []string{"hostfs", "var-log-pods", "var-lib-docker-containers"}
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

func TestAgentSecurityContext(t *testing.T) {
	t.Run("openshift runs the agent as root so it can read pod logs", func(t *testing.T) {
		sc := agentSecurityContext(config.OpenShift)
		require.NotNil(t, sc.RunAsUser)
		assert.Equal(t, int64(0), *sc.RunAsUser)
		require.NotNil(t, sc.RunAsNonRoot)
		assert.False(t, *sc.RunAsNonRoot)
		// No extra privileges beyond running as root: caps dropped, no escalation.
		assert.ElementsMatch(t, []corev1.Capability{"ALL"}, sc.Capabilities.Drop)
		require.NotNil(t, sc.AllowPrivilegeEscalation)
		assert.False(t, *sc.AllowPrivilegeEscalation)

		psc := agentPodSecurityContext(config.OpenShift)
		require.NotNil(t, psc.RunAsUser)
		assert.Equal(t, int64(0), *psc.RunAsUser)
	})

	t.Run("other distros keep the agent unprivileged", func(t *testing.T) {
		sc := agentSecurityContext(config.DistroProvider(""))
		require.NotNil(t, sc.RunAsNonRoot)
		assert.True(t, *sc.RunAsNonRoot)
		assert.Nil(t, sc.RunAsUser)

		psc := agentPodSecurityContext(config.DistroProvider(""))
		require.NotNil(t, psc.RunAsNonRoot)
		assert.True(t, *psc.RunAsNonRoot)
		assert.Nil(t, psc.RunAsUser)
	})
}

func TestBuildInstrumentations_Endpoint(t *testing.T) {
	params := manifests.Params{
		ClusterObservability: v1alpha1.ClusterObservability{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster-obs", Namespace: "observability"},
		},
		Config: operatorcfg.Config{
			EnableJavaAutoInstrumentation:   true,
			EnableNodeJSAutoInstrumentation: true,
			EnablePythonAutoInstrumentation: true,
			EnableDotNetAutoInstrumentation: true,
			EnableGoAutoInstrumentation:     true,
			AutoInstrumentationJavaImage:    "java:latest",
			AutoInstrumentationNodeJSImage:  "nodejs:latest",
			AutoInstrumentationPythonImage:  "python:latest",
			AutoInstrumentationDotNetImage:  "dotnet:latest",
			AutoInstrumentationGoImage:      "go:latest",
		},
	}

	objs, err := buildInstrumentations(params)
	require.NoError(t, err)
	require.Len(t, objs, 1)

	inst, ok := objs[0].(*v1alpha1.Instrumentation)
	require.True(t, ok)

	assert.Equal(t, "http://$(OTEL_NODE_IP):4318", inst.Spec.Endpoint)
	assert.Empty(t, inst.Spec.Java.Env)
	assert.Empty(t, inst.Spec.NodeJS.Env)
	assert.Empty(t, inst.Spec.Python.Env)
	assert.Empty(t, inst.Spec.DotNet.Env)
	assert.Empty(t, inst.Spec.Go.Env)
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

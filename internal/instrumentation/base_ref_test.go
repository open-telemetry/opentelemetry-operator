// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

func TestOverlayInstrumentation(t *testing.T) {
	parent := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{Name: "common", Namespace: "observability"},
		Spec: v1alpha1.InstrumentationSpec{
			Exporter: v1alpha1.Exporter{Endpoint: "http://common:4317"},
			Sampler:  v1alpha1.Sampler{Type: v1alpha1.TraceIDRatio, Argument: "0.1"},
			Resource: v1alpha1.Resource{Attributes: map[string]string{"cluster": "shared", "team": "common"}},
			Env: []corev1.EnvVar{
				{Name: "TEAM_MODE", Value: "common"},
				{Name: constants.EnvOTELExporterOTLPEndpoint, Value: "http://common-env:4317"},
				{Name: constants.EnvOTELResourceAttrs, Value: "source=base,team=common"},
			},
			Java: v1alpha1.Java{
				Image:      "java-agent:base",
				Extensions: []v1alpha1.Extensions{{Image: "plugin:base", Dir: "/plugins"}},
				Env: []corev1.EnvVar{
					{Name: "TEAM_MODE", Value: "base-language"},
					{Name: constants.EnvOTELExporterOTLPEndpoint, Value: "http://base-language:4317"},
					{Name: constants.EnvOTELTracesSampler, Value: "always_off"},
					{Name: constants.EnvOTELPropagators, Value: "b3"},
				},
				Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("256Mi"),
				}},
			},
		},
	}
	child := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "payments", Labels: map[string]string{"owner": "payments"}},
		Spec: v1alpha1.InstrumentationSpec{
			BaseRef:     &v1alpha1.InstrumentationReference{Name: "common", Namespace: "observability"},
			Exporter:    v1alpha1.Exporter{Endpoint: "http://payments:4317"},
			Sampler:     v1alpha1.Sampler{Type: v1alpha1.ParentBasedAlwaysOn},
			Propagators: []v1alpha1.Propagator{v1alpha1.TraceContext},
			Resource:    v1alpha1.Resource{Attributes: map[string]string{"team": "payments"}},
			Env:         []corev1.EnvVar{{Name: "TEAM_MODE", Value: "payments"}},
			Java: v1alpha1.Java{
				Env: []corev1.EnvVar{{Name: "CHILD_LANGUAGE", Value: "yes"}},
				Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("750m"),
				}},
			},
		},
	}
	beforeParent := parent.DeepCopy()
	beforeChild := child.DeepCopy()

	result, err := overlayInstrumentation(parent, child)
	require.NoError(t, err)
	assert.Equal(t, child.ObjectMeta, result.ObjectMeta)
	assert.Nil(t, result.Spec.BaseRef)
	assert.Equal(t, "java-agent:base", result.Spec.Java.Image)
	assert.Equal(t, parent.Spec.Java.Extensions, result.Spec.Java.Extensions)
	assert.Equal(t, "http://payments:4317", result.Spec.Endpoint)
	assert.Equal(t, v1alpha1.ParentBasedAlwaysOn, result.Spec.Type)
	assert.Empty(t, result.Spec.Argument)
	assert.Equal(t, []v1alpha1.Propagator{v1alpha1.TraceContext}, result.Spec.Propagators)
	assert.Equal(t, map[string]string{"cluster": "shared", "team": "payments"}, result.Spec.Resource.Attributes)
	assert.Equal(t, []corev1.EnvVar{{Name: "CHILD_LANGUAGE", Value: "yes"}}, result.Spec.Java.Env)
	assert.Equal(t, []corev1.EnvVar{
		{Name: "TEAM_MODE", Value: "payments"},
		{Name: constants.EnvOTELResourceAttrs, Value: "source=base"},
	}, result.Spec.Env)
	assert.Equal(t, resource.MustParse("750m"), result.Spec.Java.Resources.Limits[corev1.ResourceCPU])
	assert.Equal(t, resource.MustParse("256Mi"), result.Spec.Java.Resources.Limits[corev1.ResourceMemory])
	assert.Equal(t, beforeParent, parent)
	assert.Equal(t, beforeChild, child)
}

func TestOverlayInstrumentationLanguageEnvAndLists(t *testing.T) {
	parent := &v1alpha1.Instrumentation{Spec: v1alpha1.InstrumentationSpec{
		Env: []corev1.EnvVar{{Name: "MODE", Value: "base-common"}},
		Java: v1alpha1.Java{
			Extensions: []v1alpha1.Extensions{{Image: "base-plugin", Dir: "/plugins"}},
			Env:        []corev1.EnvVar{{Name: "MODE", Value: "base-language"}},
		},
	}}
	child := &v1alpha1.Instrumentation{Spec: v1alpha1.InstrumentationSpec{
		Env: []corev1.EnvVar{{Name: "MODE", Value: "child-common"}},
		Java: v1alpha1.Java{
			Extensions: []v1alpha1.Extensions{{Image: "child-plugin", Dir: "/plugins"}},
		},
	}}

	result, err := overlayInstrumentation(parent, child)
	require.NoError(t, err)
	assert.Empty(t, result.Spec.Java.Env)
	assert.Equal(t, "child-common", result.Spec.Env[0].Value)
	assert.Equal(t, child.Spec.Java.Extensions, result.Spec.Java.Extensions)
}

func TestOverlayInstrumentationLanguageFields(t *testing.T) {
	volumeLimit := resource.MustParse("100Mi")
	childVolumeLimit := resource.MustParse("150Mi")
	runAsUser := int64(1000)
	privileged := true
	tests := []struct {
		name   string
		parent v1alpha1.InstrumentationSpec
		child  v1alpha1.InstrumentationSpec
		check  func(t *testing.T, spec v1alpha1.InstrumentationSpec)
	}{
		{
			name: "NodeJS",
			parent: v1alpha1.InstrumentationSpec{NodeJS: v1alpha1.NodeJS{
				Image: "node-base", VolumeSizeLimit: &volumeLimit,
				Env: []corev1.EnvVar{{Name: "MODE", Value: "base"}},
				Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("256Mi"),
				}},
			}},
			child: v1alpha1.InstrumentationSpec{NodeJS: v1alpha1.NodeJS{
				Image: "node-child", VolumeClaimTemplate: corev1.PersistentVolumeClaimTemplate{ObjectMeta: metav1.ObjectMeta{Name: "node-claim"}},
				Env: []corev1.EnvVar{{Name: "MODE", Value: "child"}},
				Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("250m"),
				}},
			}},
			check: func(t *testing.T, spec v1alpha1.InstrumentationSpec) {
				assert.Equal(t, "node-child", spec.NodeJS.Image)
				assert.Nil(t, spec.NodeJS.VolumeSizeLimit)
				assert.Equal(t, "node-claim", spec.NodeJS.VolumeClaimTemplate.Name)
				assert.Equal(t, []corev1.EnvVar{{Name: "MODE", Value: "child"}}, spec.NodeJS.Env)
				assert.Equal(t, resource.MustParse("250m"), spec.NodeJS.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("256Mi"), spec.NodeJS.Resources.Limits[corev1.ResourceMemory])
			},
		},
		{
			name: "Python",
			parent: v1alpha1.InstrumentationSpec{Python: v1alpha1.Python{
				Image: "python-base", VolumeClaimTemplate: corev1.PersistentVolumeClaimTemplate{ObjectMeta: metav1.ObjectMeta{Name: "python-claim"}},
				Env: []corev1.EnvVar{{Name: "BASE", Value: "yes"}},
			}},
			child: v1alpha1.InstrumentationSpec{Python: v1alpha1.Python{
				VolumeSizeLimit: &childVolumeLimit,
				Env:             []corev1.EnvVar{{Name: "CHILD", Value: "yes"}},
			}},
			check: func(t *testing.T, spec v1alpha1.InstrumentationSpec) {
				assert.Equal(t, "python-base", spec.Python.Image)
				assert.Equal(t, &childVolumeLimit, spec.Python.VolumeSizeLimit)
				assert.Empty(t, spec.Python.VolumeClaimTemplate.Name)
				assert.Equal(t, []corev1.EnvVar{{Name: "CHILD", Value: "yes"}, {Name: "BASE", Value: "yes"}}, spec.Python.Env)
			},
		},
		{
			name: "DotNet",
			parent: v1alpha1.InstrumentationSpec{DotNet: v1alpha1.DotNet{
				Image: "dotnet-base", VolumeSizeLimit: &volumeLimit,
				Env: []corev1.EnvVar{{Name: "MODE", Value: "base"}},
			}},
			child: v1alpha1.InstrumentationSpec{DotNet: v1alpha1.DotNet{
				Image: "dotnet-child", Env: []corev1.EnvVar{{Name: "MODE", Value: "child"}},
			}},
			check: func(t *testing.T, spec v1alpha1.InstrumentationSpec) {
				assert.Equal(t, "dotnet-child", spec.DotNet.Image)
				assert.Equal(t, &volumeLimit, spec.DotNet.VolumeSizeLimit)
				assert.Equal(t, []corev1.EnvVar{{Name: "MODE", Value: "child"}}, spec.DotNet.Env)
			},
		},
		{
			name: "Go",
			parent: v1alpha1.InstrumentationSpec{Go: v1alpha1.Go{
				Image: "go-base", Env: []corev1.EnvVar{{Name: "MODE", Value: "base"}},
				SecurityContext: &corev1.SecurityContext{Privileged: &privileged},
			}},
			child: v1alpha1.InstrumentationSpec{Go: v1alpha1.Go{
				SecurityContext: &corev1.SecurityContext{RunAsUser: &runAsUser},
				Env:             []corev1.EnvVar{{Name: "MODE", Value: "child"}},
			}},
			check: func(t *testing.T, spec v1alpha1.InstrumentationSpec) {
				assert.Equal(t, "go-base", spec.Go.Image)
				assert.Equal(t, &runAsUser, spec.Go.SecurityContext.RunAsUser)
				assert.Nil(t, spec.Go.SecurityContext.Privileged)
				assert.Equal(t, []corev1.EnvVar{{Name: "MODE", Value: "child"}}, spec.Go.Env)
			},
		},
		{
			name: "ApacheHttpd",
			parent: v1alpha1.InstrumentationSpec{ApacheHttpd: v1alpha1.ApacheHttpd{
				Image: "apache-base", Version: "2.4", ConfigPath: "/base/conf",
				Attrs: []corev1.EnvVar{{Name: "MODE", Value: "base"}},
			}},
			child: v1alpha1.InstrumentationSpec{ApacheHttpd: v1alpha1.ApacheHttpd{
				Version: "2.2", ConfigPath: "/child/conf",
				Attrs: []corev1.EnvVar{{Name: "MODE", Value: "child"}},
			}},
			check: func(t *testing.T, spec v1alpha1.InstrumentationSpec) {
				assert.Equal(t, "apache-base", spec.ApacheHttpd.Image)
				assert.Equal(t, "2.2", spec.ApacheHttpd.Version)
				assert.Equal(t, "/child/conf", spec.ApacheHttpd.ConfigPath)
				assert.Equal(t, []corev1.EnvVar{{Name: "MODE", Value: "child"}}, spec.ApacheHttpd.Attrs)
			},
		},
		{
			name: "Nginx",
			parent: v1alpha1.InstrumentationSpec{Nginx: v1alpha1.Nginx{
				Image: "nginx-base", ConfigFile: "/base/nginx.conf",
				Attrs: []corev1.EnvVar{{Name: "MODE", Value: "base"}},
			}},
			child: v1alpha1.InstrumentationSpec{Nginx: v1alpha1.Nginx{
				ConfigFile: "/child/nginx.conf",
				Attrs:      []corev1.EnvVar{{Name: "MODE", Value: "child"}},
			}},
			check: func(t *testing.T, spec v1alpha1.InstrumentationSpec) {
				assert.Equal(t, "nginx-base", spec.Nginx.Image)
				assert.Equal(t, "/child/nginx.conf", spec.Nginx.ConfigFile)
				assert.Equal(t, []corev1.EnvVar{{Name: "MODE", Value: "child"}}, spec.Nginx.Attrs)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := overlayInstrumentation(&v1alpha1.Instrumentation{Spec: test.parent}, &v1alpha1.Instrumentation{Spec: test.child})
			require.NoError(t, err)
			test.check(t, result.Spec)
		})
	}
}

func TestOverlayInstrumentationSuppressesBaseExporterEnv(t *testing.T) {
	parent := &v1alpha1.Instrumentation{}
	baseEnvNames := []string{
		constants.EnvOTELExporterOTLPEndpoint,
		envOTELExporterOTLPTracesEndpoint,
		envOTELExporterOTLPMetricsEndpoint,
		envOTELExporterOTLPLogsEndpoint,
		constants.EnvOTELExporterCertificate,
		envOTELExporterOTLPTracesCertificate,
		envOTELExporterOTLPMetricsCertificate,
		envOTELExporterOTLPLogsCertificate,
		constants.EnvOTELExporterClientCertificate,
		envOTELExporterOTLPTracesClientCertificate,
		envOTELExporterOTLPMetricsClientCertificate,
		envOTELExporterOTLPLogsClientCertificate,
		constants.EnvOTELExporterClientKey,
		envOTELExporterOTLPTracesClientKey,
		envOTELExporterOTLPMetricsClientKey,
		envOTELExporterOTLPLogsClientKey,
	}
	for index, name := range baseEnvNames {
		env := corev1.EnvVar{Name: name, Value: "base"}
		if index%2 == 0 {
			parent.Spec.Env = append(parent.Spec.Env, env)
		} else {
			parent.Spec.Java.Env = append(parent.Spec.Java.Env, env)
		}
	}
	child := &v1alpha1.Instrumentation{Spec: v1alpha1.InstrumentationSpec{
		Exporter: v1alpha1.Exporter{Endpoint: "http://child:4317"},
	}}

	result, err := overlayInstrumentation(parent, child)
	require.NoError(t, err)
	assert.Empty(t, result.Spec.Env)
	assert.Empty(t, result.Spec.Java.Env)
}

func TestOverlayInstrumentationResourceEnvValueFrom(t *testing.T) {
	parent := &v1alpha1.Instrumentation{Spec: v1alpha1.InstrumentationSpec{
		Env: []corev1.EnvVar{{Name: constants.EnvOTELResourceAttrs, ValueFrom: &corev1.EnvVarSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "attrs"}, Key: "otel"},
		}}},
	}}
	child := &v1alpha1.Instrumentation{Spec: v1alpha1.InstrumentationSpec{
		Resource: v1alpha1.Resource{Attributes: map[string]string{"team": "payments"}},
	}}

	_, err := overlayInstrumentation(parent, child)
	require.ErrorContains(t, err, "cannot merge OTEL_RESOURCE_ATTRIBUTES from valueFrom")
}

func TestOverlayInstrumentationPartialExporterAndSampler(t *testing.T) {
	parent := &v1alpha1.Instrumentation{Spec: v1alpha1.InstrumentationSpec{
		Exporter: v1alpha1.Exporter{Endpoint: "https://common:4317", TLS: &v1alpha1.TLS{SecretName: "base-tls"}},
		Sampler:  v1alpha1.Sampler{Type: v1alpha1.TraceIDRatio, Argument: "0.1"},
		Env: []corev1.EnvVar{
			{Name: constants.EnvOTELExporterOTLPEndpoint, Value: "http://common-env:4317"},
			{Name: constants.EnvOTELExporterCertificate, Value: "/base/ca.crt"},
			{Name: envOTELExporterOTLPTracesCertificate, Value: "/base/traces-ca.crt"},
		},
	}}
	tests := []struct {
		name             string
		exporter         v1alpha1.Exporter
		sampler          v1alpha1.Sampler
		wantEndpoint     string
		wantTLS          *v1alpha1.TLS
		wantSamplerType  v1alpha1.SamplerType
		wantSamplerArg   string
		wantParentEnvLen int
	}{
		{
			name:             "TLS only inherits endpoint",
			exporter:         v1alpha1.Exporter{TLS: &v1alpha1.TLS{SecretName: "child-tls"}},
			wantEndpoint:     "https://common:4317",
			wantTLS:          &v1alpha1.TLS{SecretName: "child-tls"},
			wantSamplerType:  v1alpha1.TraceIDRatio,
			wantSamplerArg:   "0.1",
			wantParentEnvLen: 0,
		},
		{
			name:             "endpoint only clears parent TLS",
			exporter:         v1alpha1.Exporter{Endpoint: "http://child:4317"},
			sampler:          v1alpha1.Sampler{Type: v1alpha1.ParentBasedAlwaysOn},
			wantEndpoint:     "http://child:4317",
			wantSamplerType:  v1alpha1.ParentBasedAlwaysOn,
			wantParentEnvLen: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			child := &v1alpha1.Instrumentation{Spec: v1alpha1.InstrumentationSpec{Exporter: test.exporter, Sampler: test.sampler}}
			result, err := overlayInstrumentation(parent, child)
			require.NoError(t, err)
			assert.Equal(t, test.wantEndpoint, result.Spec.Endpoint)
			assert.Equal(t, test.wantTLS, result.Spec.TLS)
			assert.Equal(t, test.wantSamplerType, result.Spec.Type)
			assert.Equal(t, test.wantSamplerArg, result.Spec.Argument)
			assert.Len(t, result.Spec.Env, test.wantParentEnvLen)
		})
	}
}

func TestOverlayInstrumentationRejectsTLSWithoutBaseHTTPSEndpoint(t *testing.T) {
	child := &v1alpha1.Instrumentation{Spec: v1alpha1.InstrumentationSpec{
		Exporter: v1alpha1.Exporter{TLS: &v1alpha1.TLS{SecretName: "child-tls"}},
	}}
	for _, test := range []struct {
		name   string
		parent v1alpha1.InstrumentationSpec
	}{
		{
			name:   "HTTP endpoint",
			parent: v1alpha1.InstrumentationSpec{Exporter: v1alpha1.Exporter{Endpoint: "http://common:4317"}},
		},
		{
			name:   "endpoint only in env",
			parent: v1alpha1.InstrumentationSpec{Env: []corev1.EnvVar{{Name: constants.EnvOTELExporterOTLPEndpoint, Value: "https://common:4317"}}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := overlayInstrumentation(&v1alpha1.Instrumentation{Spec: test.parent}, child)
			require.ErrorContains(t, err, "base spec.exporter.endpoint must use https://")
		})
	}
}

func TestOverlayInstrumentationRejectsSamplerArgumentWithoutType(t *testing.T) {
	parent := &v1alpha1.Instrumentation{Spec: v1alpha1.InstrumentationSpec{
		Sampler: v1alpha1.Sampler{Type: v1alpha1.TraceIDRatio, Argument: "0.1"},
	}}
	child := &v1alpha1.Instrumentation{Spec: v1alpha1.InstrumentationSpec{
		Sampler: v1alpha1.Sampler{Argument: "0.5"},
	}}

	_, err := overlayInstrumentation(parent, child)
	require.ErrorContains(t, err, "spec.sampler.type is required when overriding spec.sampler.argument")
}

func TestOverlayInstrumentationEmptyParentResourceEnv(t *testing.T) {
	parent := &v1alpha1.Instrumentation{Spec: v1alpha1.InstrumentationSpec{
		Env: []corev1.EnvVar{{Name: constants.EnvOTELResourceAttrs}},
	}}
	child := &v1alpha1.Instrumentation{Spec: v1alpha1.InstrumentationSpec{
		Resource: v1alpha1.Resource{Attributes: map[string]string{"team": "payments"}},
	}}

	result, err := overlayInstrumentation(parent, child)
	require.NoError(t, err)
	assert.Empty(t, result.Spec.Env)
}

func TestResolveInstrumentationReference(t *testing.T) {
	parent := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{Name: "common", Namespace: "observability"},
		Spec:       v1alpha1.InstrumentationSpec{Java: v1alpha1.Java{Image: "java-agent:base"}},
	}
	localParent := parent.DeepCopy()
	localParent.Namespace = "payments"
	child := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "payments"},
		Spec: v1alpha1.InstrumentationSpec{
			BaseRef: &v1alpha1.InstrumentationReference{Name: "common"},
		},
	}
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(parent, localParent, child).Build()
	pm := &instPodMutator{Client: client}

	result, err := pm.resolveInstrumentation(t.Context(), child)
	require.NoError(t, err)
	assert.Equal(t, "payments", result.Namespace)
	assert.Equal(t, "java-agent:base", result.Spec.Java.Image)

	crossNamespaceChild := child.DeepCopy()
	crossNamespaceChild.Spec.BaseRef.Namespace = "observability"
	result, err = pm.resolveInstrumentation(t.Context(), crossNamespaceChild)
	require.NoError(t, err)
	assert.Equal(t, "java-agent:base", result.Spec.Java.Image)

	missingChild := child.DeepCopy()
	missingChild.Spec.BaseRef.Name = "missing"
	_, err = pm.resolveInstrumentation(t.Context(), missingChild)
	require.ErrorContains(t, err, "failed to get base instrumentation payments/missing")

	selfChild := child.DeepCopy()
	selfChild.Spec.BaseRef.Name = "service"
	_, err = pm.resolveInstrumentation(t.Context(), selfChild)
	require.ErrorContains(t, err, "cannot reference itself")

	chainChild := child.DeepCopy()
	chainChild.Spec.BaseRef.Name = "chain"
	chainParent := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{Name: "chain", Namespace: "payments"},
		Spec: v1alpha1.InstrumentationSpec{
			BaseRef: &v1alpha1.InstrumentationReference{Name: "common"},
		},
	}
	require.NoError(t, client.Create(t.Context(), chainParent))
	_, err = pm.resolveInstrumentation(t.Context(), chainChild)
	require.ErrorContains(t, err, "must not set spec.baseRef")
}

func TestMutatePodWithBaseInstrumentation(t *testing.T) {
	parent := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{Name: "common", Namespace: "observability"},
		Spec: v1alpha1.InstrumentationSpec{
			Java: v1alpha1.Java{Image: "java-agent:base", Env: []corev1.EnvVar{{Name: "TEAM", Value: "base"}}},
			Env: []corev1.EnvVar{
				{Name: constants.EnvOTELExporterOTLPEndpoint, Value: "http://base:4317"},
				{Name: envOTELExporterOTLPTracesEndpoint, Value: "http://base-traces:4318"},
			},
		},
	}
	child := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "payments"},
		Spec: v1alpha1.InstrumentationSpec{
			BaseRef:  &v1alpha1.InstrumentationReference{Name: "common", Namespace: "observability"},
			Exporter: v1alpha1.Exporter{Endpoint: "http://payments:4317"},
			Env:      []corev1.EnvVar{{Name: "TEAM", Value: "payments"}},
		},
	}
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(parent, child).Build()
	mutator := NewMutator(logr.Discard(), client, events.NewFakeRecorder(10), config.New())
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "payments"}}
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "payments", Name: "app", Annotations: map[string]string{
			annotationInjectJava: "service",
		}},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Env: []corev1.EnvVar{{Name: "POD", Value: "own"}}}}},
	}
	withOwnEnv := pod.DeepCopy()
	withOwnEnv.Spec.Containers[0].Env = append(withOwnEnv.Spec.Containers[0].Env,
		corev1.EnvVar{Name: "TEAM", Value: "pod"},
		corev1.EnvVar{Name: constants.EnvOTELExporterOTLPEndpoint, Value: "http://pod:4317"},
		corev1.EnvVar{Name: envOTELExporterOTLPTracesEndpoint, Value: "http://pod-traces:4318"},
		corev1.EnvVar{Name: envOTELExporterOTLPTracesCertificate, Value: "/pod/ca.crt"},
	)

	result, err := mutator.Mutate(t.Context(), ns, pod)
	require.NoError(t, err)
	require.NotEmpty(t, result.Spec.InitContainers)
	assert.Equal(t, "java-agent:base", result.Spec.InitContainers[0].Image)
	assert.Equal(t, "payments", result.Spec.Containers[0].Env[getIndexOfEnv(result.Spec.Containers[0].Env, "TEAM")].Value)
	assert.Equal(t, "http://payments:4317", result.Spec.Containers[0].Env[getIndexOfEnv(result.Spec.Containers[0].Env, constants.EnvOTELExporterOTLPEndpoint)].Value)
	assert.Equal(t, -1, getIndexOfEnv(result.Spec.Containers[0].Env, envOTELExporterOTLPTracesEndpoint))
	assert.Equal(t, "own", result.Spec.Containers[0].Env[getIndexOfEnv(result.Spec.Containers[0].Env, "POD")].Value)

	result, err = mutator.Mutate(t.Context(), ns, *withOwnEnv)
	require.NoError(t, err)
	assert.Equal(t, "pod", result.Spec.Containers[0].Env[getIndexOfEnv(result.Spec.Containers[0].Env, "TEAM")].Value)
	assert.Equal(t, "http://pod:4317", result.Spec.Containers[0].Env[getIndexOfEnv(result.Spec.Containers[0].Env, constants.EnvOTELExporterOTLPEndpoint)].Value)
	assert.Equal(t, "http://pod-traces:4318", result.Spec.Containers[0].Env[getIndexOfEnv(result.Spec.Containers[0].Env, envOTELExporterOTLPTracesEndpoint)].Value)
	assert.Equal(t, "/pod/ca.crt", result.Spec.Containers[0].Env[getIndexOfEnv(result.Spec.Containers[0].Env, envOTELExporterOTLPTracesCertificate)].Value)
}

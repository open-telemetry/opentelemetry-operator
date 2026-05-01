// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/yaml"
)

var testLogger = logr.Discard()

func getFakeK8sClient(t *testing.T, objs ...client.Object) client.Client {
	return getFakeK8sClientWithInterceptor(t, interceptor.Funcs{}, objs...)
}

func getFakeK8sClientWithInterceptor(t *testing.T, funcs interceptor.Funcs, objs ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	require.NoError(t, v1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	builder := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(funcs)
	for _, obj := range objs {
		builder = builder.WithObjects(obj)
	}
	return builder.Build()
}

func newTestClient(k8s client.Client) *Client {
	return NewClient("test-bridge", testLogger, k8s, nil, nil)
}

func mustMarshalStandaloneConfig(t *testing.T, name, namespace string, config map[string]string) []byte {
	t.Helper()
	b, err := yaml.Marshal(standaloneConfig{
		Version:   standaloneConfigVersion,
		Name:      name,
		Namespace: namespace,
		Config:    config,
	})
	require.NoError(t, err)
	return b
}

// managedConfigMap returns a ConfigMap with the managed-by label and a workload annotation.
func managedConfigMap(name, namespace, workloadAnnotation string) *v1.ConfigMap {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				managedByLabel: managedByValue,
			},
		},
		Data: map[string]string{
			"collector.yaml": "receivers:\n  otlp:\n",
		},
	}
	if workloadAnnotation != "" {
		cm.Annotations = map[string]string{
			rolloutAnnotationKey: workloadAnnotation,
		}
	}
	return cm
}

func testDeployment(name string) *appsv1.Deployment {
	replicas := int32(2)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
				Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "collector", Image: "otel/collector:latest"}}},
			},
		},
	}
}

func testDaemonSet(name string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
				Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "collector", Image: "otel/collector:latest"}}},
			},
		},
	}
}

// --- Apply tests ---

func TestApply_CreatesConfigMap(t *testing.T) {
	k8s := getFakeK8sClient(t)
	c := newTestClient(k8s)

	newData := map[string]string{"collector.yaml": "receivers:\n  otlp:\n"}
	err := c.Apply("my-config", "default", &protobufs.AgentConfigFile{
		Body:        mustMarshalStandaloneConfig(t, "my-config", "default", newData),
		ContentType: "yaml",
	})
	require.NoError(t, err)

	cm := &v1.ConfigMap{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "my-config", Namespace: "default"}, cm))
	assert.Equal(t, newData, cm.Data)
	assert.Equal(t, managedByValue, cm.Labels[managedByLabel])
}

func TestApply_UpdatesExistingConfigMap(t *testing.T) {
	cm := managedConfigMap("my-config", "default", "")
	k8s := getFakeK8sClient(t, cm)
	c := newTestClient(k8s)

	newData := map[string]string{"collector.yaml": "new config"}
	err := c.Apply("my-config", "default", &protobufs.AgentConfigFile{
		Body:        mustMarshalStandaloneConfig(t, "my-config", "default", newData),
		ContentType: "yaml",
	})
	require.NoError(t, err)

	updated := &v1.ConfigMap{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "my-config", Namespace: "default"}, updated))
	assert.Equal(t, newData, updated.Data)
}

func TestApply_RejectsUnmanagedConfigMap(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "missing managed-by label",
		},
		{
			name:   "wrong managed-by label value",
			labels: map[string]string{managedByLabel: "someone-else"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalData := map[string]string{"collector.yaml": "original config"}
			cm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-config",
					Namespace: "default",
					Labels:    tt.labels,
				},
				Data: originalData,
			}
			k8s := getFakeK8sClient(t, cm)
			c := newTestClient(k8s)

			err := c.Apply("my-config", "default", &protobufs.AgentConfigFile{
				Body:        mustMarshalStandaloneConfig(t, "my-config", "default", map[string]string{"collector.yaml": "new config"}),
				ContentType: "yaml",
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "cannot modify unmanaged ConfigMap default/my-config")

			unchanged := &v1.ConfigMap{}
			require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "my-config", Namespace: "default"}, unchanged))
			assert.Equal(t, originalData, unchanged.Data)
		})
	}
}

func TestApply_ReplacesAllKeys(t *testing.T) {
	cm := managedConfigMap("my-config", "default", "")
	cm.Data = map[string]string{"old.yaml": "old", "extra.yaml": "extra"}
	k8s := getFakeK8sClient(t, cm)
	c := newTestClient(k8s)

	newData := map[string]string{"collector.yaml": "new"}
	err := c.Apply("my-config", "default", &protobufs.AgentConfigFile{
		Body:        mustMarshalStandaloneConfig(t, "my-config", "default", newData),
		ContentType: "yaml",
	})
	require.NoError(t, err)

	updated := &v1.ConfigMap{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "my-config", Namespace: "default"}, updated))
	assert.Equal(t, newData, updated.Data)
	assert.NotContains(t, updated.Data, "old.yaml")
	assert.NotContains(t, updated.Data, "extra.yaml")
}

func TestApply_DoesNotTriggerRolloutOnCreate(t *testing.T) {
	// The standalone wire schema does not carry rollout targets. Newly-created
	// ConfigMaps can only trigger rollouts after a local annotation is present.
	deploy := testDeployment("my-collector")
	k8s := getFakeK8sClient(t, deploy)
	c := newTestClient(k8s)

	body := mustMarshalStandaloneConfig(t, "my-config", "default", map[string]string{"collector.yaml": "receivers:\n  otlp:\n"})
	err := c.Apply("my-config", "default", &protobufs.AgentConfigFile{Body: body, ContentType: "yaml"})
	require.NoError(t, err)

	updated := &appsv1.Deployment{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "my-collector", Namespace: "default"}, updated))
	assert.NotContains(t, updated.Spec.Template.Annotations, restartAnnotation)
}

func TestApply_TriggersDeploymentRollout(t *testing.T) {
	cm := managedConfigMap("my-config", "default", "Deployment/my-collector")
	deploy := testDeployment("my-collector")
	k8s := getFakeK8sClient(t, cm, deploy)
	c := newTestClient(k8s)

	err := c.Apply("my-config", "default", &protobufs.AgentConfigFile{
		Body:        mustMarshalStandaloneConfig(t, "my-config", "default", map[string]string{"x": "y"}),
		ContentType: "yaml",
	})
	require.NoError(t, err)

	updated := &appsv1.Deployment{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "my-collector", Namespace: "default"}, updated))
	assert.Contains(t, updated.Spec.Template.Annotations, restartAnnotation)
}

func TestApply_TriggersDaemonSetRollout(t *testing.T) {
	cm := managedConfigMap("my-config", "default", "DaemonSet/my-agent")
	ds := testDaemonSet("my-agent")
	k8s := getFakeK8sClient(t, cm, ds)
	c := newTestClient(k8s)

	err := c.Apply("my-config", "default", &protobufs.AgentConfigFile{
		Body:        mustMarshalStandaloneConfig(t, "my-config", "default", map[string]string{"x": "y"}),
		ContentType: "yaml",
	})
	require.NoError(t, err)

	updated := &appsv1.DaemonSet{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "my-agent", Namespace: "default"}, updated))
	assert.Contains(t, updated.Spec.Template.Annotations, restartAnnotation)
}

func TestApply_TriggersMultipleWorkloadRollouts(t *testing.T) {
	cm := managedConfigMap("my-config", "default", "Deployment/col-a,DaemonSet/agent-b")
	deploy := testDeployment("col-a")
	ds := testDaemonSet("agent-b")
	k8s := getFakeK8sClient(t, cm, deploy, ds)
	c := newTestClient(k8s)

	err := c.Apply("my-config", "default", &protobufs.AgentConfigFile{
		Body:        mustMarshalStandaloneConfig(t, "my-config", "default", map[string]string{"x": "y"}),
		ContentType: "yaml",
	})
	require.NoError(t, err)

	updatedDeploy := &appsv1.Deployment{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "col-a", Namespace: "default"}, updatedDeploy))
	assert.Contains(t, updatedDeploy.Spec.Template.Annotations, restartAnnotation)

	updatedDS := &appsv1.DaemonSet{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "agent-b", Namespace: "default"}, updatedDS))
	assert.Contains(t, updatedDS.Spec.Template.Annotations, restartAnnotation)
}

func TestApply_NoRolloutWhenNoAnnotation(t *testing.T) {
	cm := managedConfigMap("my-config", "default", "")
	deploy := testDeployment("my-collector")
	k8s := getFakeK8sClient(t, cm, deploy)
	c := newTestClient(k8s)

	err := c.Apply("my-config", "default", &protobufs.AgentConfigFile{
		Body:        mustMarshalStandaloneConfig(t, "my-config", "default", map[string]string{"x": "y"}),
		ContentType: "yaml",
	})
	require.NoError(t, err)

	// Deployment should NOT have been restarted.
	updated := &appsv1.Deployment{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "my-collector", Namespace: "default"}, updated))
	assert.NotContains(t, updated.Spec.Template.Annotations, restartAnnotation)
}

func TestApply_EmptyBody(t *testing.T) {
	k8s := getFakeK8sClient(t)
	c := newTestClient(k8s)

	err := c.Apply("my-config", "default", &protobufs.AgentConfigFile{Body: []byte{}})
	assert.Error(t, err)
}

func TestApply_InvalidStandaloneConfig(t *testing.T) {
	tests := []struct {
		name          string
		targetName    string
		targetNS      string
		config        standaloneConfig
		errorContains string
	}{
		{
			name:          "bad version",
			targetName:    "my-config",
			targetNS:      "default",
			config:        standaloneConfig{Version: "v1", Name: "my-config", Namespace: "default", Config: map[string]string{"x": "y"}},
			errorContains: "unsupported standalone config version",
		},
		{
			name:          "missing name",
			targetName:    "my-config",
			targetNS:      "default",
			config:        standaloneConfig{Version: standaloneConfigVersion, Namespace: "default", Config: map[string]string{"x": "y"}},
			errorContains: "name is required",
		},
		{
			name:          "missing namespace",
			targetName:    "my-config",
			targetNS:      "default",
			config:        standaloneConfig{Version: standaloneConfigVersion, Name: "my-config", Config: map[string]string{"x": "y"}},
			errorContains: "namespace is required",
		},
		{
			name:          "name mismatch",
			targetName:    "my-config",
			targetNS:      "default",
			config:        standaloneConfig{Version: standaloneConfigVersion, Name: "other", Namespace: "default", Config: map[string]string{"x": "y"}},
			errorContains: "does not match target name",
		},
		{
			name:          "namespace mismatch",
			targetName:    "my-config",
			targetNS:      "default",
			config:        standaloneConfig{Version: standaloneConfigVersion, Name: "my-config", Namespace: "other", Config: map[string]string{"x": "y"}},
			errorContains: "does not match target namespace",
		},
		{
			name:          "empty config",
			targetName:    "my-config",
			targetNS:      "default",
			config:        standaloneConfig{Version: standaloneConfigVersion, Name: "my-config", Namespace: "default"},
			errorContains: "config data is required",
		},
		{
			name:          "empty key",
			targetName:    "my-config",
			targetNS:      "default",
			config:        standaloneConfig{Version: standaloneConfigVersion, Name: "my-config", Namespace: "default", Config: map[string]string{"": "y"}},
			errorContains: "empty data key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8s := getFakeK8sClient(t)
			c := newTestClient(k8s)
			body, err := yaml.Marshal(tt.config)
			require.NoError(t, err)

			err = c.Apply(tt.targetName, tt.targetNS, &protobufs.AgentConfigFile{
				Body:        body,
				ContentType: "yaml",
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
		})
	}
}

func TestApply_MissingWorkloadIsSkipped(t *testing.T) {
	// Annotation references a Deployment that doesn't exist — should not error.
	configName := "test-config"
	ns := "test-ns"
	cm := managedConfigMap(configName, ns, "Deployment/missing")
	k8s := getFakeK8sClient(t, cm)
	c := newTestClient(k8s)

	err := c.Apply(configName, ns, &protobufs.AgentConfigFile{
		Body:        mustMarshalStandaloneConfig(t, configName, ns, map[string]string{"x": "y"}),
		ContentType: "yaml",
	})
	require.NoError(t, err, "missing workload should not cause Apply to fail")

	updated := &v1.ConfigMap{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: configName, Namespace: ns}, updated))
	assert.Equal(t, map[string]string{"x": "y"}, updated.Data)
}

func TestApply_MalformedWorkloadReferenceIsSkipped(t *testing.T) {
	configName := "my-config"
	ns := "default"
	cm := managedConfigMap(configName, ns, "malformed,Deployment/my-collector")
	deploy := testDeployment("my-collector")
	k8s := getFakeK8sClient(t, cm, deploy)
	c := newTestClient(k8s)

	err := c.Apply(configName, ns, &protobufs.AgentConfigFile{
		Body:        mustMarshalStandaloneConfig(t, configName, ns, map[string]string{"x": "y"}),
		ContentType: "yaml",
	})
	require.NoError(t, err, "malformed rollout refs should not cause Apply to fail")

	updatedCM := &v1.ConfigMap{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: configName, Namespace: ns}, updatedCM))
	assert.Equal(t, map[string]string{"x": "y"}, updatedCM.Data)

	updatedDeployment := &appsv1.Deployment{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "my-collector", Namespace: ns}, updatedDeployment))
	assert.Contains(t, updatedDeployment.Spec.Template.Annotations, restartAnnotation)
}

func TestApply_UnsupportedWorkloadKindIsSkipped(t *testing.T) {
	cm := managedConfigMap("my-config", "default", "StatefulSet/my-collector,Deployment/my-collector")
	deploy := testDeployment("my-collector")
	k8s := getFakeK8sClient(t, cm, deploy)
	c := newTestClient(k8s)

	err := c.Apply("my-config", "default", &protobufs.AgentConfigFile{
		Body:        mustMarshalStandaloneConfig(t, "my-config", "default", map[string]string{"x": "y"}),
		ContentType: "yaml",
	})
	require.NoError(t, err, "unsupported rollout kinds should not cause Apply to fail")

	updatedCM := &v1.ConfigMap{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "my-config", Namespace: "default"}, updatedCM))
	assert.Equal(t, map[string]string{"x": "y"}, updatedCM.Data)

	updatedDeployment := &appsv1.Deployment{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "my-collector", Namespace: "default"}, updatedDeployment))
	assert.Contains(t, updatedDeployment.Spec.Template.Annotations, restartAnnotation)
}

func TestApply_RolloutUpdateFailureIsBestEffort(t *testing.T) {
	cm := managedConfigMap("my-config", "default", "Deployment/my-collector")
	deploy := testDeployment("my-collector")
	k8s := getFakeK8sClientWithInterceptor(t, interceptor.Funcs{
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			if _, ok := obj.(*appsv1.Deployment); ok {
				return errors.New("simulated deployment update failure")
			}
			return c.Update(ctx, obj, opts...)
		},
	}, cm, deploy)
	c := newTestClient(k8s)

	err := c.Apply("my-config", "default", &protobufs.AgentConfigFile{
		Body:        mustMarshalStandaloneConfig(t, "my-config", "default", map[string]string{"x": "y"}),
		ContentType: "yaml",
	})
	require.NoError(t, err, "rollout update failures should not cause Apply to fail")

	updatedCM := &v1.ConfigMap{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "my-config", Namespace: "default"}, updatedCM))
	assert.Equal(t, map[string]string{"x": "y"}, updatedCM.Data)
}

// --- Delete tests ---

func TestDelete_RemovesConfigMap(t *testing.T) {
	cm := managedConfigMap("my-config", "default", "")
	k8s := getFakeK8sClient(t, cm)
	c := newTestClient(k8s)

	require.NoError(t, c.Delete("my-config", "default"))

	err := k8s.Get(context.Background(), client.ObjectKey{Name: "my-config", Namespace: "default"}, &v1.ConfigMap{})
	assert.Error(t, err, "ConfigMap should be deleted")
}

func TestDelete_RejectsUnmanagedConfigMap(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "missing managed-by label",
		},
		{
			name:   "wrong managed-by label value",
			labels: map[string]string{managedByLabel: "someone-else"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-config",
					Namespace: "default",
					Labels:    tt.labels,
				},
				Data: map[string]string{"collector.yaml": "original config"},
			}
			k8s := getFakeK8sClient(t, cm)
			c := newTestClient(k8s)

			err := c.Delete("my-config", "default")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "cannot delete unmanaged ConfigMap default/my-config")

			remaining := &v1.ConfigMap{}
			require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "my-config", Namespace: "default"}, remaining))
			assert.Equal(t, cm.Data, remaining.Data)
		})
	}
}

func TestDelete_NotFound(t *testing.T) {
	k8s := getFakeK8sClient(t)
	c := newTestClient(k8s)

	require.NoError(t, c.Delete("nonexistent", "default"), "deleting non-existent ConfigMap should not error")
}

// --- ListInstances tests ---

func TestListInstances_ReturnsManagedConfigMaps(t *testing.T) {
	cm1 := managedConfigMap("cfg-a", "monitoring", "Deployment/col-a")
	cm2 := managedConfigMap("cfg-b", "monitoring", "")
	k8s := getFakeK8sClient(t, cm1, cm2)
	c := newTestClient(k8s)

	instances, err := c.ListInstances()
	require.NoError(t, err)
	assert.Len(t, instances, 2)
}

func TestListInstances_IgnoresUnmanagedConfigMaps(t *testing.T) {
	unmanaged := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "unmanaged", Namespace: "default"},
		Data:       map[string]string{"key": "val"},
	}
	managed := managedConfigMap("managed", "default", "")
	k8s := getFakeK8sClient(t, unmanaged, managed)
	c := newTestClient(k8s)

	instances, err := c.ListInstances()
	require.NoError(t, err)
	require.Len(t, instances, 1)
	assert.Equal(t, "managed", instances[0].GetName())
}

func TestListInstances_EffectiveConfigContainsData(t *testing.T) {
	cm := managedConfigMap("my-config", "default", "")
	cm.Data = map[string]string{
		"collector.yaml": "receivers:\n  otlp:\n",
		"pipelines.yaml": "service:\n  pipelines: {}\n",
	}
	k8s := getFakeK8sClient(t, cm)
	c := newTestClient(k8s)

	instances, err := c.ListInstances()
	require.NoError(t, err)
	require.Len(t, instances, 1)

	body := string(instances[0].GetEffectiveConfig())
	assert.Contains(t, body, "collector.yaml")
	assert.Contains(t, body, "pipelines.yaml")
	assert.Contains(t, body, "version: "+standaloneConfigVersion)
	assert.Contains(t, body, "name: my-config")
	assert.Contains(t, body, "namespace: default")
	assert.Contains(t, body, "config:")
	assert.NotContains(t, body, "apiVersion:")
	assert.NotContains(t, body, "kind:")
	assert.NotContains(t, body, "metadata:")

	var effective standaloneConfig
	require.NoError(t, yaml.Unmarshal([]byte(body), &effective))
	assert.Equal(t, standaloneConfigVersion, effective.Version)
	assert.Equal(t, "my-config", effective.Name)
	assert.Equal(t, "default", effective.Namespace)
	assert.Equal(t, cm.Data, effective.Config)
}

func TestListInstances_EmptyConfigMap(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "empty-config",
			Namespace: "default",
			Labels:    map[string]string{managedByLabel: managedByValue},
		},
	}
	k8s := getFakeK8sClient(t, cm)
	c := newTestClient(k8s)

	instances, err := c.ListInstances()
	require.NoError(t, err)
	require.Len(t, instances, 1)
	assert.Nil(t, instances[0].GetEffectiveConfig())
}

func TestListInstances_SelectorLabelsAlwaysEmpty(t *testing.T) {
	cm := managedConfigMap("my-config", "default", "Deployment/my-collector")
	k8s := getFakeK8sClient(t, cm)
	c := newTestClient(k8s)

	instances, err := c.ListInstances()
	require.NoError(t, err)
	require.Len(t, instances, 1)
	assert.Equal(t, map[string]string{}, instances[0].GetSelectorLabels())
}

func TestListInstances_StatusReplicasEmpty(t *testing.T) {
	cm := managedConfigMap("my-config", "default", "")
	k8s := getFakeK8sClient(t, cm)
	c := newTestClient(k8s)

	instances, err := c.ListInstances()
	require.NoError(t, err)
	require.Len(t, instances, 1)
	assert.Equal(t, "", instances[0].GetStatusReplicas())
}

// --- parseWorkloadAnnotation tests ---

func TestParseWorkloadAnnotation(t *testing.T) {
	tests := []struct {
		annotation string
		want       []string
	}{
		{"", nil},
		{"Deployment/col", []string{"Deployment/col"}},
		{"Deployment/col,DaemonSet/agent", []string{"Deployment/col", "DaemonSet/agent"}},
		{" Deployment/col , DaemonSet/agent ", []string{"Deployment/col", "DaemonSet/agent"}},
		{",,,", nil},
	}
	for _, tt := range tests {
		got := parseWorkloadAnnotation(tt.annotation)
		assert.Equal(t, tt.want, got, "annotation: %q", tt.annotation)
	}
}

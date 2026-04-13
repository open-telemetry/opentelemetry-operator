// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/config"
	bridgemanager "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/manager"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/operator"
)

const validCollectorConfig = `receivers:
  otlp:
    protocols:
      grpc:
exporters:
  debug:
service:
  pipelines:
    traces:
      receivers:
        - otlp
      exporters:
        - debug
`

func getFakeK8sClient(t *testing.T, objs ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	require.NoError(t, v1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, authorizationv1.AddToScheme(scheme))
	builder := fake.NewClientBuilder().WithScheme(scheme)
	for _, obj := range objs {
		builder = builder.WithObjects(obj)
	}
	return builder.Build()
}

func getPermissionReviewClient(t *testing.T, allowed func(authorizationv1.ResourceAttributes) bool) client.Client {
	scheme := runtime.NewScheme()
	require.NoError(t, v1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, authorizationv1.AddToScheme(scheme))
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				review, ok := obj.(*authorizationv1.SelfSubjectAccessReview)
				if !ok {
					return c.Create(ctx, obj, opts...)
				}
				review.Status.Allowed = allowed(*review.Spec.ResourceAttributes)
				if !review.Status.Allowed {
					review.Status.Reason = "denied by test"
				}
				return nil
			},
		}).
		Build()
}

func testConfigMap() *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "collector-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"collector.yaml": validCollectorConfig,
			"extra.yaml":     "extra: true",
		},
	}
}

func testAgentConfig() config.StandaloneAgentConfig {
	return config.StandaloneAgentConfig{
		Namespace: "default",
		Type:      "otel-collector",
		WorkloadRef: config.StandaloneWorkloadRef{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "standalone-collector",
		},
		Config: map[string]config.StandaloneConfigEntry{
			"collector": {
				Kind: "configmap",
				Name: "collector-config",
				Key:  "collector.yaml",
			},
		},
	}
}

func testDeployment() *appsv1.Deployment {
	replicas := ptr.To[int32](10)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "standalone-collector",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Template: v1.PodTemplateSpec{},
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 5,
			Replicas:      5,
		},
	}
}

func testStatefulSet() *appsv1.StatefulSet {
	replicas := ptr.To[int32](4)
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "standalone-collector",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: replicas,
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas: 3,
			Replicas:      3,
		},
	}
}

func testDaemonSet() *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "standalone-collector",
			Namespace: "default",
		},
		Status: appsv1.DaemonSetStatus{
			NumberReady:            7,
			DesiredNumberScheduled: 9,
		},
	}
}

func newTestClient(k8s client.Client) *Client {
	return NewClient(logr.Discard(), k8s, nil, nil)
}

func TestScopedApplierListInstancesReturnsWorkloadWithConfiguredConfigMap(t *testing.T) {
	cm := testConfigMap()
	c := newTestClient(getFakeK8sClient(t, cm, testDeployment()))
	agentCfg := testAgentConfig()
	agentCfg.Config["extra"] = config.StandaloneConfigEntry{
		Kind: "configmap",
		Name: "collector-config",
		Key:  "extra.yaml",
	}

	instances, err := c.ScopedApplier(agentCfg).ListInstances()
	require.NoError(t, err)
	require.Len(t, instances, 1)

	assert.Equal(t, "standalone-collector", instances[0].GetName())
	assert.Equal(t, "default", instances[0].GetNamespace())
	configMap := instances[0].GetConfigMap()
	require.Len(t, configMap, 2)
	assert.Equal(t, validCollectorConfig, string(configMap["collector"].Body))
	assert.Equal(t, "yaml", configMap["collector"].ContentType)
	assert.Equal(t, "extra: true", string(configMap["extra"].Body))
	assert.Equal(t, "yaml", configMap["extra"].ContentType)
}

func TestClientGetWorkloadStatusReplicas(t *testing.T) {
	tests := []struct {
		name         string
		workloadType string
		workload     client.Object
		want         string
		wantHealthy  bool
	}{
		{
			name:         "deployment uses spec replicas during scale up",
			workloadType: "deployment",
			workload:     testDeployment(),
			want:         "5/10",
			wantHealthy:  false,
		},
		{
			name:         "deployment defaults desired replicas to one",
			workloadType: "deployment",
			workload: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "standalone-collector",
					Namespace: "default",
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 1,
					Replicas:      1,
				},
			},
			want:        "1/1",
			wantHealthy: true,
		},
		{
			name:         "daemonset",
			workloadType: "daemonset",
			workload:     testDaemonSet(),
			want:         "7/9",
			wantHealthy:  false,
		},
		{
			name:         "statefulset uses spec replicas during scale up",
			workloadType: "statefulset",
			workload:     testStatefulSet(),
			want:         "3/4",
			wantHealthy:  false,
		},
		{
			name:         "statefulset defaults desired replicas to one",
			workloadType: "statefulset",
			workload: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "standalone-collector",
					Namespace: "default",
				},
				Status: appsv1.StatefulSetStatus{
					ReadyReplicas: 1,
					Replicas:      1,
				},
			},
			want:        "1/1",
			wantHealthy: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(getFakeK8sClient(t, tt.workload))

			status, err := c.getWorkloadStatusReplicas(context.Background(), "default", tt.workloadType, "standalone-collector")
			require.NoError(t, err)
			assert.Equal(t, tt.want, status.String())
			assert.Equal(t, tt.wantHealthy, status.Healthy())
		})
	}
}

func TestScopedApplierGetHealthReturnsRootWorkloadHealth(t *testing.T) {
	healthyDeploy := testDeployment()
	healthyDeploy.Status.ReadyReplicas = 10
	healthyDeploy.Status.Replicas = 10
	tests := []struct {
		name     string
		workload client.Object
		want     operator.Health
	}{
		{
			name:     "unhealthy rollout",
			workload: testDeployment(),
			want: operator.Health{
				Healthy:  false,
				Status:   "5/10",
				Children: map[string]operator.Health{},
			},
		},
		{
			name:     "healthy rollout",
			workload: healthyDeploy,
			want: operator.Health{
				Healthy:  true,
				Status:   "10/10",
				Children: map[string]operator.Health{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(getFakeK8sClient(t, tt.workload))

			got, err := c.ScopedApplier(testAgentConfig()).GetHealth()

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClientNotifyWorkloadHealthUpdate(t *testing.T) {
	c := newTestClient(getFakeK8sClient(t))
	called := false
	c.RegisterHealthUpdater(testAgentConfig(), func() error {
		called = true
		return nil
	})

	c.notifyWorkloadHealthUpdate("deployment", testDeployment())

	assert.True(t, called)
}

func TestClientCheckPermissionsAllowsConfiguredStandaloneResources(t *testing.T) {
	c := newTestClient(getPermissionReviewClient(t, func(_ authorizationv1.ResourceAttributes) bool {
		return true
	}))

	err := c.CheckPermissions(context.Background(), []config.StandaloneAgentConfig{testAgentConfig()}, true)

	require.NoError(t, err)
}

func TestClientCheckPermissionsRejectsMissingConfiguredResourcePermission(t *testing.T) {
	c := newTestClient(getPermissionReviewClient(t, func(attrs authorizationv1.ResourceAttributes) bool {
		return attrs.Verb != "update" || attrs.Resource != "deployments" || attrs.Name != "standalone-collector"
	}))

	err := c.CheckPermissions(context.Background(), []config.StandaloneAgentConfig{testAgentConfig()}, true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "standalone permission check failed")
	assert.Contains(t, err.Error(), "missing update permission for apps/deployments default/standalone-collector")
}

func TestPermissionsCheckerRejectsDeniedPermission(t *testing.T) {
	k8sClient := getPermissionReviewClient(t, func(_ authorizationv1.ResourceAttributes) bool {
		return false
	})

	err := bridgemanager.CheckPermissions(context.Background(), k8sClient, []bridgemanager.Permission{{
		Verb:      "update",
		APIGroup:  "apps",
		Resource:  "deployments",
		Namespace: "default",
		Name:      "standalone-collector",
	}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing update permission for apps/deployments default/standalone-collector")
}

func TestScopedApplierApplyUpdatesConfiguredConfigMapKeyAndRestartsWorkload(t *testing.T) {
	cm := testConfigMap()
	deploy := testDeployment()
	k8s := getFakeK8sClient(t, cm, deploy)
	c := newTestClient(k8s)

	updatedConfig := `receivers:
  otlp:
    protocols:
      http:
exporters:
  debug:
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [debug]
`
	err := c.ScopedApplier(testAgentConfig()).Apply("collector", &protobufs.AgentConfigFile{Body: []byte(updatedConfig)})
	require.NoError(t, err)

	updated := &v1.ConfigMap{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "collector-config", Namespace: "default"}, updated))
	assert.Equal(t, updatedConfig, updated.Data["collector.yaml"])
	assert.Equal(t, "extra: true", updated.Data["extra.yaml"])

	updatedDeploy := &appsv1.Deployment{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "standalone-collector", Namespace: "default"}, updatedDeploy))
	assert.NotEmpty(t, updatedDeploy.Spec.Template.Annotations[restartAnnotation])
}

func TestScopedApplierApplyRejectsUnknownRemoteName(t *testing.T) {
	c := newTestClient(getFakeK8sClient(t, testConfigMap()))

	err := c.ScopedApplier(testAgentConfig()).Apply("unknown", &protobufs.AgentConfigFile{Body: []byte(validCollectorConfig)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not manage config")
}

func TestScopedApplierApplyDoesNotCreateConfigMap(t *testing.T) {
	c := newTestClient(getFakeK8sClient(t))

	err := c.ScopedApplier(testAgentConfig()).Apply("collector", &protobufs.AgentConfigFile{Body: []byte(validCollectorConfig)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support creating ConfigMap")
}

func TestScopedApplierDeleteUnsupported(t *testing.T) {
	c := newTestClient(getFakeK8sClient(t, testConfigMap()))

	err := c.ScopedApplier(testAgentConfig()).Delete("collector")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support deleting")
}

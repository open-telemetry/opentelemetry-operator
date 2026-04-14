// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policy1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/cmd/gather/config"
)

// MockClient is a mock implementation of client.Client.
type MockClient struct {
	mock.Mock
}

func (m *MockClient) Apply(ctx context.Context, obj runtime.ApplyConfiguration, _ ...client.ApplyOption) error {
	args := m.Called(ctx, obj)
	return args.Error(0)
}

func (m *MockClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
	args := m.Called(ctx, key, obj)
	return args.Error(0)
}

func (m *MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	args := m.Called(ctx, list, opts)
	return args.Error(0)
}

func (m *MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

func (m *MockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Status() client.StatusWriter {
	args := m.Called()
	return args.Get(0).(client.StatusWriter)
}

func (m *MockClient) Scheme() *runtime.Scheme {
	args := m.Called()
	return args.Get(0).(*runtime.Scheme)
}

func (m *MockClient) RESTMapper() meta.RESTMapper {
	args := m.Called()
	return args.Get(0).(meta.RESTMapper)
}

func (*MockClient) GroupVersionKindFor(runtime.Object) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}

func (*MockClient) IsObjectNamespaced(runtime.Object) (bool, error) {
	return true, nil
}

func (*MockClient) SubResource(string) client.SubResourceClient {
	return nil
}

func TestGetOperatorNamespace(t *testing.T) {
	mockClient := new(MockClient)
	cfg := &config.Config{
		KubernetesClient: mockClient,
	}
	cluster := NewCluster(cfg)

	// Test when OperatorNamespace is already set
	cfg.OperatorNamespace = "test-namespace"
	ns, err := cluster.getOperatorNamespace()
	assert.NoError(t, err)
	assert.Equal(t, "test-namespace", ns)

	// Test when OperatorNamespace is not set
	cfg.OperatorNamespace = ""
	mockClient.On("List", mock.Anything, &appsv1.DeploymentList{}, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(1).(*appsv1.DeploymentList)
		arg.Items = []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "operator-namespace",
				},
			},
		}
	})

	ns, err = cluster.getOperatorNamespace()
	assert.NoError(t, err)
	assert.Equal(t, "operator-namespace", ns)
	mockClient.AssertExpectations(t)
}

func TestGetOperatorDeployment(t *testing.T) {
	mockClient := new(MockClient)
	cfg := &config.Config{
		KubernetesClient: mockClient,
	}
	cluster := NewCluster(cfg)
	// Test successful case
	mockClient.On("List", mock.Anything, &appsv1.DeploymentList{}, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(1).(*appsv1.DeploymentList)
		arg.Items = []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "opentelemetry-operator",
				},
			},
		}
	})

	deployment, err := cluster.getOperatorDeployment()
	assert.NoError(t, err)
	assert.Equal(t, "opentelemetry-operator", deployment.Name)

	mockClient.AssertExpectations(t)
}

func buildTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(v1beta1.AddToScheme(s))
	utilruntime.Must(otelv1alpha1.AddToScheme(s))
	utilruntime.Must(appsv1.AddToScheme(s))
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(networkingv1.AddToScheme(s))
	utilruntime.Must(autoscalingv2.AddToScheme(s))
	utilruntime.Must(rbacv1.AddToScheme(s))
	utilruntime.Must(policy1.AddToScheme(s))
	return s
}

func TestOMCDirectoryLayout(t *testing.T) {
	scheme := buildTestScheme()

	collectorUID := types.UID("collector-uid-abc")
	collector := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-collector",
			Namespace: "test-ns",
			UID:       collectorUID,
		},
	}
	// Pod is owned by a ReplicaSet (not directly by the collector), so it must be
	// collected via the app.kubernetes.io/instance label, not by owner reference.
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pod",
			Namespace: "test-ns",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
				"app.kubernetes.io/instance":   "test-ns.my-collector",
			},
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-service",
			Namespace: "test-ns",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			},
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "OpenTelemetryCollector", UID: collectorUID},
			},
		},
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-configmap",
			Namespace: "test-ns",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			},
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "OpenTelemetryCollector", UID: collectorUID},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(collector, svc, cm, pod).
		Build()

	collectionDir := t.TempDir()
	cfg := &config.Config{
		CollectionDir:    collectionDir,
		KubernetesClient: fakeClient,
		Scheme:           scheme,
	}

	c := NewCluster(cfg)
	err := c.GetOpenTelemetryCollectors()
	assert.NoError(t, err)

	// Collector itself at omc-compatible path.
	assert.FileExists(t, filepath.Join(collectionDir, "namespaces", "test-ns", "opentelemetry.io", "opentelemetrycollectors", "my-collector.yaml"))
	// Owned Service: core group (empty API group → "core").
	assert.FileExists(t, filepath.Join(collectionDir, "namespaces", "test-ns", "core", "services", "my-service.yaml"))
	// Owned ConfigMap: core group.
	assert.FileExists(t, filepath.Join(collectionDir, "namespaces", "test-ns", "core", "configmaps", "my-configmap.yaml"))

	// Pod collected via instance label (not direct owner reference).
	assert.FileExists(t, filepath.Join(collectionDir, "namespaces", "test-ns", "core", "pods", "my-pod.yaml"))

	// Old collector-name-as-directory layout must not exist.
	assert.NoDirExists(t, filepath.Join(collectionDir, "namespaces", "test-ns", "my-collector"))
}

func TestGetOperatorDeploymentNotFound(t *testing.T) {
	mockClient := new(MockClient)
	cfg := &config.Config{
		KubernetesClient: mockClient,
	}
	cluster := NewCluster(cfg)

	// Test when no operator is found
	mockClient.On("List", mock.Anything, &appsv1.DeploymentList{}, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(1).(*appsv1.DeploymentList)
		arg.Items = []appsv1.Deployment{}
	})

	_, err := cluster.getOperatorDeployment()
	assert.Error(t, err)
	assert.Equal(t, "operator not found", err.Error())
}

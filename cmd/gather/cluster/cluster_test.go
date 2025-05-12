// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/cmd/gather/config"
)

// MockClient is a mock implementation of client.Client.
type MockClient struct {
	mock.Mock
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

func (m *MockClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}

func (m *MockClient) IsObjectNamespaced(_ runtime.Object) (bool, error) {
	return true, nil
}

func (m *MockClient) SubResource(string) client.SubResourceClient {
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

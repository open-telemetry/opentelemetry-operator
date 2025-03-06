// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

type MockObject struct {
	mock.Mock
}

// Implement all methods required by client.Object and runtime.Object

// GetObjectKind mocks the GetObjectKind method.
func (m *MockObject) GetObjectKind() schema.ObjectKind {
	args := m.Called()
	return args.Get(0).(schema.ObjectKind)
}

// GetName mocks the GetName method.
func (m *MockObject) GetName() string {
	args := m.Called()
	return args.String(0)
}

// SetName mocks the SetName method.
func (m *MockObject) SetName(name string) {
	m.Called(name)
}

// GetNamespace mocks the GetNamespace method.
func (m *MockObject) GetNamespace() string {
	args := m.Called()
	return args.String(0)
}

// SetNamespace mocks the SetNamespace method.
func (m *MockObject) SetNamespace(namespace string) {
	m.Called(namespace)
}

// GetAnnotations mocks the GetAnnotations method.
func (m *MockObject) GetAnnotations() map[string]string {
	args := m.Called()
	return args.Get(0).(map[string]string)
}

// SetAnnotations mocks the SetAnnotations method.
func (m *MockObject) SetAnnotations(annotations map[string]string) {
	m.Called(annotations)
}

// GetCreationTimestamp mocks the GetCreationTimestamp method.
func (m *MockObject) GetCreationTimestamp() v1.Time {
	args := m.Called()
	return args.Get(0).(v1.Time)
}

// SetCreationTimestamp mocks the SetCreationTimestamp method.
func (m *MockObject) SetCreationTimestamp(timestamp v1.Time) {
	m.Called(timestamp)
}

// GetDeletionGracePeriodSeconds mocks the GetDeletionGracePeriodSeconds method.
func (m *MockObject) GetDeletionGracePeriodSeconds() *int64 {
	args := m.Called()
	return args.Get(0).(*int64)
}

// GetDeletionTimestamp mocks the GetDeletionTimestamp method.
func (m *MockObject) GetDeletionTimestamp() *v1.Time {
	args := m.Called()
	return args.Get(0).(*v1.Time)
}

// GetLabels mocks the GetLabels method.
func (m *MockObject) GetLabels() map[string]string {
	args := m.Called()
	return args.Get(0).(map[string]string)
}

// SetLabels mocks the SetLabels method.
func (m *MockObject) SetLabels(labels map[string]string) {
	m.Called(labels)
}

// GetFinalizers mocks the GetFinalizers method.
func (m *MockObject) GetFinalizers() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

// SetFinalizers mocks the SetFinalizers method.
func (m *MockObject) SetFinalizers(finalizers []string) {
	m.Called(finalizers)
}

// GetGenerateName mocks the GetGenerateName method.
func (m *MockObject) GetGenerateName() string {
	args := m.Called()
	return args.String(0)
}

// SetGenerateName mocks the SetGenerateName method.
func (m *MockObject) SetGenerateName(name string) {
	m.Called(name)
}

// DeepCopyObject mocks the DeepCopyObject method.
func (m *MockObject) DeepCopyObject() runtime.Object {
	args := m.Called()
	return args.Get(0).(runtime.Object)
}

func (m *MockObject) GetManagedFields() []v1.ManagedFieldsEntry {
	args := m.Called()
	return args.Get(0).([]v1.ManagedFieldsEntry)
}

func (m *MockObject) GetOwnerReferences() []v1.OwnerReference {
	args := m.Called()
	return args.Get(0).([]v1.OwnerReference)
}

func (m *MockObject) GetGeneration() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

func (m *MockObject) GetResourceVersion() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockObject) GetSelfLink() string {
	args := m.Called()
	return args.String(0)
}

type MockPodInterface struct {
	mock.Mock
}

func (m *MockPodInterface) GetLogs(podName string, options *corev1.PodLogOptions) *rest.Request {
	args := m.Called(podName, options)
	return args.Get(0).(*rest.Request)
}

type MockRequest struct {
	mock.Mock
}

func (m *MockRequest) Stream(ctx context.Context) (io.ReadCloser, error) {
	args := m.Called(ctx)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func TestCreateOTELFolder(t *testing.T) {
	collectionDir := "/tmp/test-dir"
	otelCol := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "test-otel",
		},
	}

	outputDir, err := createOTELFolder(collectionDir, otelCol)

	expectedDir := filepath.Join(collectionDir, "namespaces", otelCol.Namespace, otelCol.Name)
	assert.NoError(t, err)
	assert.Equal(t, expectedDir, outputDir)

	// Clean up after the test
	os.RemoveAll(collectionDir)
}

func TestCreateFile(t *testing.T) {
	outputDir := "/tmp/test-dir"
	err := os.MkdirAll(outputDir, os.ModePerm)
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	mockObj := &MockObject{}
	mockObj.On("GetObjectKind").Return(schema.EmptyObjectKind)
	mockObj.On("GetName").Return("test-deployment")
	mockObj.On("DeepCopyObject").Return(mockObj)

	file, err := createFile(outputDir, mockObj)
	assert.NoError(t, err)
	defer file.Close()

	expectedPath := filepath.Join(outputDir, "mockobject-test-deployment.yaml")
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err)
}

func (m *MockObject) SetUID(uid types.UID) {
	m.Called(uid)
}

func (m *MockObject) GetUID() types.UID {
	args := m.Called()
	return args.Get(0).(types.UID)
}

func (m *MockObject) SetDeletionGracePeriodSeconds(seconds *int64) {
	m.Called(seconds)
}

func (m *MockObject) SetDeletionTimestamp(timestamp *v1.Time) {
	m.Called(timestamp)
}

func (m *MockObject) SetGeneration(generation int64) {
	m.Called(generation)
}

func (m *MockObject) SetManagedFields(fields []v1.ManagedFieldsEntry) {
	m.Called(fields)
}

func (m *MockObject) SetOwnerReferences(references []v1.OwnerReference) {
	m.Called(references)
}

func (m *MockObject) SetResourceVersion(version string) {
	m.Called(version)
}

func (m *MockObject) SetSelfLink(selfLink string) {
	m.Called(selfLink)
}

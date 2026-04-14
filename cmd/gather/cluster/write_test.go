// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestResourceDirNamespaced(t *testing.T) {
	collectionDir := t.TempDir()
	got := resourceDir(collectionDir, "test-ns", "opentelemetry.io", "opentelemetrycollectors")
	want := filepath.Join(collectionDir, "namespaces", "test-ns", "opentelemetry.io", "opentelemetrycollectors")
	assert.Equal(t, want, got)
}

func TestResourceDirCoreGroup(t *testing.T) {
	collectionDir := t.TempDir()
	// Empty API group (core resources like Service, ConfigMap) must map to "core" in the path.
	got := resourceDir(collectionDir, "test-ns", "", "services")
	want := filepath.Join(collectionDir, "namespaces", "test-ns", "core", "services")
	assert.Equal(t, want, got)
}

func TestResourceDirClusterScoped(t *testing.T) {
	collectionDir := t.TempDir()
	got := resourceDir(collectionDir, "", "apiextensions.k8s.io", "customresourcedefinitions")
	want := filepath.Join(collectionDir, "cluster-scoped-resources", "apiextensions.k8s.io", "customresourcedefinitions")
	assert.Equal(t, want, got)
}

func TestWriteToFileNaming(t *testing.T) {
	collectionDir := t.TempDir()
	scheme := buildTestScheme()

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-deployment",
			Namespace: "test-ns",
		},
	}

	writeToFile(collectionDir, deployment, scheme)

	// New naming: <name>.yaml, not <kind>-<name>.yaml
	expectedPath := filepath.Join(collectionDir, "namespaces", "test-ns", "apps", "deployments", "my-deployment.yaml")
	assert.FileExists(t, expectedPath)

	wrongPath := filepath.Join(collectionDir, "namespaces", "test-ns", "apps", "deployments", "deployment-my-deployment.yaml")
	assert.NoFileExists(t, wrongPath)
}

func TestWriteToFileGVK(t *testing.T) {
	collectionDir := t.TempDir()
	scheme := buildTestScheme()

	// Simulate controller-runtime List behavior: TypeMeta is not set on items.
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-deployment",
			Namespace: "test-ns",
		},
	}

	writeToFile(collectionDir, deployment, scheme)

	yamlPath := filepath.Join(collectionDir, "namespaces", "test-ns", "apps", "deployments", "my-deployment.yaml")
	content, err := os.ReadFile(yamlPath)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "apiVersion: apps/v1")
	assert.Contains(t, contentStr, "kind: Deployment")
}

func TestLogOutputPath(t *testing.T) {
	got := logOutputPath("/collection", "my-ns", "my-pod", "manager")
	want := "/collection/namespaces/my-ns/pods/my-pod/manager/logs/current.log"
	assert.Equal(t, want, got)
}

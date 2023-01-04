// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operator

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

var (
	clientLogger = logf.Log.WithName("client-tests")
)

func getFakeClient(t *testing.T) client.WithWatch {
	schemeBuilder := runtime.NewSchemeBuilder(func(s *runtime.Scheme) error {
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.OpenTelemetryCollector{}, &v1alpha1.OpenTelemetryCollectorList{})
		metav1.AddToGroupVersion(s, v1alpha1.GroupVersion)
		return nil
	})
	scheme := runtime.NewScheme()
	err := schemeBuilder.AddToScheme(scheme)
	require.NoError(t, err, "Should be able to add custom types")
	c := fake.NewClientBuilder().WithScheme(scheme)
	return c.Build()
}

func TestClient_Apply(t *testing.T) {
	type args struct {
		name      string
		namespace string
		file      string
		config    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "base case",
			args: args{
				name:      "test",
				namespace: "opentelemetry",
				file:      "testdata/collector.yaml",
			},
			wantErr: false,
		},
		{
			name: "invalid config",
			args: args{
				name:      "test",
				namespace: "opentelemetry",
				file:      "testdata/invalid-collector.yaml",
			},
			wantErr: true,
		},
		{
			name: "empty config",
			args: args{
				name:      "test",
				namespace: "opentelemetry",
				config:    "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := getFakeClient(t)
			c := NewClient(clientLogger, fakeClient, nil)
			var colConfig []byte
			var err error
			if len(tt.args.file) > 0 {
				colConfig, err = loadConfig(tt.args.file)
				require.NoError(t, err, "Should be no error on loading test configuration")
			} else {
				colConfig = []byte(tt.args.config)
			}
			configmap := &protobufs.AgentConfigFile{
				Body:        colConfig,
				ContentType: "yaml",
			}
			if err := c.Apply(tt.args.name, tt.args.namespace, configmap); (err != nil) != tt.wantErr {
				t.Errorf("Apply() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_collectorUpdate(t *testing.T) {
	name := "test"
	namespace := "testing"
	fakeClient := getFakeClient(t)
	c := NewClient(clientLogger, fakeClient, nil)
	colConfig, err := loadConfig("testdata/collector.yaml")
	require.NoError(t, err, "Should be no error on loading test configuration")
	configmap := &protobufs.AgentConfigFile{
		Body:        colConfig,
		ContentType: "yaml",
	}
	// Apply a valid initial configuration
	err = c.Apply(name, namespace, configmap)
	require.NoError(t, err, "Should apply base config")

	// Get the newly created collector
	instance, err := c.GetInstance(name, namespace)
	require.NoError(t, err, "Should be able to get the newly created instance")
	assert.Contains(t, instance.Spec.Config, "processors: []")

	// Try updating with an invalid one
	configmap.Body = []byte("empty, invalid!")
	err = c.Apply(name, namespace, configmap)
	assert.Error(t, err, "Should be unable to update")

	// Update successfully with a valid configuration
	newColConfig, err := loadConfig("testdata/updated-collector.yaml")
	require.NoError(t, err, "Should be no error on loading test configuration")
	newConfigMap := &protobufs.AgentConfigFile{
		Body:        newColConfig,
		ContentType: "yaml",
	}
	err = c.Apply(name, namespace, newConfigMap)
	require.NoError(t, err, "Should be able to update collector")

	// Get the updated collector
	updatedInstance, err := c.GetInstance(name, namespace)
	require.NoError(t, err, "Should be able to get the updated instance")
	assert.Contains(t, updatedInstance.Spec.Config, "processors: [memory_limiter, batch]")

	allInstances, err := c.ListInstances()
	require.NoError(t, err, "Should be able to list all collectors")
	assert.Len(t, allInstances, 1)
	assert.Equal(t, allInstances[0], *updatedInstance)
}

func Test_collectorDelete(t *testing.T) {
	name := "test"
	namespace := "testing"
	fakeClient := getFakeClient(t)
	c := NewClient(clientLogger, fakeClient, nil)
	colConfig, err := loadConfig("testdata/collector.yaml")
	require.NoError(t, err, "Should be no error on loading test configuration")
	configmap := &protobufs.AgentConfigFile{
		Body:        colConfig,
		ContentType: "yaml",
	}
	// Apply a valid initial configuration
	err = c.Apply(name, namespace, configmap)
	require.NoError(t, err, "Should apply base config")

	// Get the newly created collector
	instance, err := c.GetInstance(name, namespace)
	require.NoError(t, err, "Should be able to get the newly created instance")
	assert.Contains(t, instance.Spec.Config, "processors: []")

	// Delete it
	err = c.Delete(name, namespace)
	require.NoError(t, err, "Should be able to delete a collector")

	// Check there's nothing left
	allInstances, err := c.ListInstances()
	require.NoError(t, err, "Should be able to list all collectors")
	assert.Len(t, allInstances, 0)
}

func loadConfig(file string) ([]byte, error) {
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return []byte{}, err
	}
	return yamlFile, nil
}

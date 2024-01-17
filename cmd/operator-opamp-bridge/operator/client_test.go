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
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

var (
	clientLogger = logr.Discard()
)

const (
	bridgeName = "bridge-test"
)

func getFakeClient(t *testing.T, lists ...client.ObjectList) client.WithWatch {
	schemeBuilder := runtime.NewSchemeBuilder(func(s *runtime.Scheme) error {
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.OpenTelemetryCollector{}, &v1alpha1.OpenTelemetryCollectorList{})
		s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Pod{}, &v1.PodList{})
		metav1.AddToGroupVersion(s, v1alpha1.GroupVersion)
		return nil
	})
	scheme := runtime.NewScheme()
	err := schemeBuilder.AddToScheme(scheme)
	require.NoError(t, err, "Should be able to add custom types")
	c := fake.NewClientBuilder().WithLists(lists...).WithScheme(scheme)
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
		name        string
		args        args
		wantErr     bool
		errContains string
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
			wantErr:     true,
			errContains: "error converting YAML to JSON",
		},
		{
			name: "empty config",
			args: args{
				name:      "test",
				namespace: "opentelemetry",
				config:    "",
			},
			wantErr:     true,
			errContains: "Must supply valid configuration",
		},
		{
			name: "create reporting-only",
			args: args{
				name:      "test",
				namespace: "opentelemetry",
				file:      "testdata/reporting-collector.yaml",
			},
			wantErr:     true,
			errContains: "opentelemetry.io/opamp-reporting",
		},
		{
			name: "create managed false",
			args: args{
				name:      "test",
				namespace: "opentelemetry",
				file:      "testdata/unmanaged-collector.yaml",
			},
			wantErr:     true,
			errContains: "opentelemetry.io/opamp-managed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := getFakeClient(t)
			c := NewClient(bridgeName, clientLogger, fakeClient, nil)
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
			applyErr := c.Apply(tt.args.name, tt.args.namespace, configmap)
			if tt.wantErr {
				assert.Error(t, applyErr)
				assert.ErrorContains(t, applyErr, tt.errContains)
			}
		})
	}
}

func Test_collectorUpdate(t *testing.T) {
	name := "test"
	namespace := "testing"
	fakeClient := getFakeClient(t)
	c := NewClient(bridgeName, clientLogger, fakeClient, nil)

	// Load reporting-only collector
	reportingColConfig, err := loadConfig("testdata/reporting-collector.yaml")
	require.NoError(t, err, "Should be no error on loading test configuration")
	var reportingCol v1alpha1.OpenTelemetryCollector
	err = yaml.Unmarshal(reportingColConfig, &reportingCol)
	require.NoError(t, err, "Should be no error on unmarshal")
	reportingCol.TypeMeta.Kind = CollectorResource
	reportingCol.TypeMeta.APIVersion = v1alpha1.GroupVersion.String()
	reportingCol.ObjectMeta.Name = "simplest"
	reportingCol.ObjectMeta.Namespace = namespace
	err = fakeClient.Create(context.Background(), &reportingCol)
	require.NoError(t, err, "Should be able to make reporting col")
	allInstances, err := c.ListInstances()
	require.NoError(t, err, "Should be able to list all collectors")
	require.Len(t, allInstances, 1)

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

	allInstances, err = c.ListInstances()
	require.NoError(t, err, "Should be able to list all collectors")
	assert.Len(t, allInstances, 2)
	assert.Contains(t, allInstances, reportingCol)
	assert.Contains(t, allInstances, *updatedInstance)
}

func Test_collectorDelete(t *testing.T) {
	name := "test"
	namespace := "testing"
	fakeClient := getFakeClient(t)
	c := NewClient(bridgeName, clientLogger, fakeClient, nil)
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

func TestClient_GetCollectorPods(t *testing.T) {
	mockPodList := &v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mock-pod",
					Namespace: "something",
					Labels: map[string]string{
						"match1": "yes",
						"match2": "1",
					},
				},
				Spec: v1.PodSpec{},
			},
		}}
	emptyList := &v1.PodList{
		Items: []v1.Pod{}}
	type args struct {
		selector  map[string]string
		namespace string
	}
	tests := []struct {
		name    string
		args    args
		want    *v1.PodList
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "base case",
			args: args{
				selector: map[string]string{
					"match1": "yes",
					"match2": "1",
				},
			},
			want:    mockPodList,
			wantErr: assert.NoError,
		},
		{
			name: "no match",
			args: args{
				selector: map[string]string{
					"match1": "yes",
					"match2": "2",
				},
			},
			want:    emptyList,
			wantErr: assert.NoError,
		},
		{
			name: "good selector wrong namespace",
			args: args{
				selector: map[string]string{
					"match1": "yes",
					"match2": "1",
				},
				namespace: "nothing",
			},
			want:    emptyList,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := getFakeClient(t, mockPodList)
			c := NewClient(bridgeName, clientLogger, fakeClient, nil)
			got, err := c.GetCollectorPods(tt.args.selector, tt.args.namespace)
			if !tt.wantErr(t, err, fmt.Sprintf("GetCollectorPods(%v)", tt.args.selector)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetCollectorPods(%v)", tt.args.selector)
		})
	}
}

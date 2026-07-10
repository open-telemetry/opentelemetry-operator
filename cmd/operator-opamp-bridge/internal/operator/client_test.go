// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

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
	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/rollout"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var clientLogger = logr.Discard()

const (
	bridgeName = "bridge-test"
)

func getFakeClient(t *testing.T, lists ...client.ObjectList) client.WithWatch {
	schemeBuilder := runtime.NewSchemeBuilder(func(s *runtime.Scheme) error {
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.OpenTelemetryCollector{}, &v1alpha1.OpenTelemetryCollectorList{})
		s.AddKnownTypes(v1beta1.GroupVersion, &v1beta1.OpenTelemetryCollector{}, &v1beta1.OpenTelemetryCollectorList{})
		s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Pod{}, &v1.PodList{})
		metav1.AddToGroupVersion(s, v1alpha1.GroupVersion)
		return appsv1.AddToScheme(s)
	})
	scheme := runtime.NewScheme()
	err := schemeBuilder.AddToScheme(scheme)
	require.NoError(t, err, "Should be able to add custom types")
	c := fake.NewClientBuilder().WithLists(lists...).WithScheme(scheme)
	return c.Build()
}

func TestClient_Apply(t *testing.T) {
	componentsAllowed := map[string]map[string]bool{
		"receivers": {
			"otlp": true,
		},
		"processors": {
			"memory_limiter": true,
			"batch":          true,
		},
		"exporters": {
			"debug": true,
		},
	}

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
			name: "no processors case",
			args: args{
				name:      "test",
				namespace: "opentelemetry",
				file:      "testdata/no-processors-collector.yaml",
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
			errContains: "invalid config to apply: config is empty",
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
		{
			name: "cannot apply v1alpha1 Collector config",
			args: args{
				name:      "test",
				namespace: "opentelemetry",
				file:      "testdata/collector-v1alpha1.yaml",
			},
			wantErr:     true,
			errContains: "failed to unmarshal config into v1beta1 API Version",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := getFakeClient(t)
			c := NewClient(bridgeName, clientLogger, fakeClient, componentsAllowed)
			var colConfig []byte
			var err error
			if tt.args.file != "" {
				colConfig, err = loadConfig(tt.args.file)
				require.NoError(t, err, "Should be no error on loading test configuration")
			} else {
				colConfig = []byte(tt.args.config)
			}
			configmap := &protobufs.AgentConfigFile{
				Body:        colConfig,
				ContentType: "yaml",
			}
			applyErr := c.Apply(NewKubeResourceKey(tt.args.namespace, tt.args.name).String(), configmap)
			if tt.wantErr {
				assert.Error(t, applyErr)
				assert.ErrorContains(t, applyErr, tt.errContains)
			}
		})
	}
}

func TestClient_ApplyUpdate(t *testing.T) {
	name := "test"
	namespace := "testing"
	fakeClient := getFakeClient(t)
	c := NewClient(bridgeName, clientLogger, fakeClient, nil)

	// Load reporting-only collector
	reportingColConfig, err := loadConfig("testdata/reporting-collector.yaml")
	require.NoError(t, err, "Should be no error on loading test configuration")

	var reportingCol v1beta1.OpenTelemetryCollector
	err = yaml.Unmarshal(reportingColConfig, &reportingCol)
	require.NoError(t, err, "Should be no error on unmarshal")

	setTypedMeta(&reportingCol)
	reportingCol.Name = "simplest"
	reportingCol.Namespace = namespace

	err = fakeClient.Create(context.Background(), &reportingCol)
	require.NoError(t, err, "Should be able to make reporting col")
	setTypedMeta(&reportingCol) // calling client.Create() can unset this

	allInstances, err := c.ListInstances()
	require.NoError(t, err, "Should be able to list all collectors")
	require.Len(t, allInstances, 1)

	// Create managed collector
	colConfig, err := loadConfig("testdata/collector.yaml")
	require.NoError(t, err, "Should be no error on loading test configuration")
	configmap := &protobufs.AgentConfigFile{
		Body:        colConfig,
		ContentType: "yaml",
	}
	// Apply a valid initial configuration
	err = c.Apply(NewKubeResourceKey(namespace, name).String(), configmap)
	require.NoError(t, err, "Should apply base config")

	// Confirm there are now two collector instances, reporting and managed
	allInstances, err = c.ListInstances()
	require.NoError(t, err, "Should be able to list all collectors")
	require.Len(t, allInstances, 2, "Should be two collector instances")

	// Get the newly created collector instance
	instance, err := c.GetInstance(name, namespace)
	require.NoError(t, err, "Should be able to get the newly created instance")

	require.NotNil(t, instance, "Should be able to get the newly created instance")
	require.Len(t, instance.Spec.Config.Service.Pipelines, 1, "Should have a single pipeline")
	require.Contains(t, instance.Spec.Config.Service.Pipelines, "traces", "Should have a traces pipeline")
	originalTracesPipeline := instance.Spec.Config.Service.Pipelines["traces"]
	require.NotNil(t, originalTracesPipeline, "Should have a traces pipeline")
	require.Empty(t, originalTracesPipeline.Processors, "Should have the no processors configured for the traces pipeline")

	// Try updating with an invalid configuration
	configmap.Body = []byte("empty, invalid!")
	err = c.Apply(NewKubeResourceKey(namespace, name).String(), configmap)
	assert.Error(t, err, "Should be unable to update with invalid config")

	// Update successfully with a valid configuration
	newColConfig, err := loadConfig("testdata/updated-collector.yaml")
	require.NoError(t, err, "Should be no error on loading test configuration")
	newConfigMap := &protobufs.AgentConfigFile{
		Body:        newColConfig,
		ContentType: "yaml",
	}
	err = c.Apply(NewKubeResourceKey(namespace, name).String(), newConfigMap)
	require.NoError(t, err, "Should be able to update collector")

	// Get the updated collector
	updatedInstance, err := c.GetInstance(name, namespace)
	require.NoError(t, err, "Should be able to get the updated instance without error")
	require.NotNil(t, updatedInstance, "Should be able to get the newly created instance")
	require.Len(t, updatedInstance.Spec.Config.Service.Pipelines, 1, "Should have a single pipeline")
	require.Contains(t, updatedInstance.Spec.Config.Service.Pipelines, "traces", "Should have a traces pipeline")
	newTracesPipeline := updatedInstance.Spec.Config.Service.Pipelines["traces"]
	require.NotNil(t, newTracesPipeline, "Should have a traces pipeline")
	require.Equal(t, []string{"memory_limiter", "batch"}, newTracesPipeline.Processors, "Should have the memory_limiter and batch processors")

	allInstances, err = c.ListInstances()
	require.NoError(t, err, "Should be able to list all collectors")
	assert.Len(t, allInstances, 2)
	instanceNames := make([]string, len(allInstances))
	for i, inst := range allInstances {
		instanceNames[i] = inst.GetNamespace() + "/" + inst.GetName()
	}
	assert.Contains(t, instanceNames, reportingCol.GetNamespace()+"/"+reportingCol.GetName())
	assert.Contains(t, instanceNames, updatedInstance.GetNamespace()+"/"+updatedInstance.GetName())
}

func TestClient_Delete(t *testing.T) {
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
	err = c.Apply(NewKubeResourceKey(namespace, name).String(), configmap)
	require.NoError(t, err, "Should apply base config")

	// Get the newly created collector
	instance, err := c.GetInstance(name, namespace)
	require.NoError(t, err, "Should be able to get the newly created instance without error")
	require.NotNil(t, instance, "Should be able to get the newly created instance")
	require.NotNil(t, instance.Spec.Config.Processors, "Should have processor")
	require.Contains(t, instance.Spec.Config.Processors.Object, "batch", "Should have the batch processor")
	require.Len(t, instance.Spec.Config.Service.Pipelines, 1, "Should have a pipeline")

	// Delete it
	err = c.Delete(NewKubeResourceKey(namespace, name).String())
	require.NoError(t, err, "Should be able to delete a collector")

	// Check there's nothing left
	allInstances, err := c.ListInstances()
	require.NoError(t, err, "Should be able to list all collectors")
	require.Empty(t, allInstances, "Should be empty after deletion")
}

func loadConfig(file string) ([]byte, error) {
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return []byte{}, err
	}
	return yamlFile, nil
}

func TestClient_getCollectorPods(t *testing.T) {
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
		},
	}
	emptyList := &v1.PodList{
		Items: []v1.Pod{},
	}
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
			got, err := c.getCollectorPods(tt.args.selector, tt.args.namespace)
			if !tt.wantErr(t, err, fmt.Sprintf("getCollectorPods(%v)", tt.args.selector)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getCollectorPods(%v)", tt.args.selector)
		})
	}
}

// managedCollector creates a v1beta1 OpenTelemetryCollector with the managed label set,
// which makes it visible to listOpenTelemetryCollectors.
func managedCollector(name, namespace string, mode v1beta1.Mode) *v1beta1.OpenTelemetryCollector {
	return &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{ManagedLabelKey: "true"},
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode: mode,
		},
	}
}

func TestClient_Restart_Deployment(t *testing.T) {
	col := managedCollector("test-col", "default", v1beta1.ModeDeployment)
	workloadName := naming.Collector(col.Name)
	deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: workloadName, Namespace: "default"}}

	fakeClient := getFakeClient(t)
	require.NoError(t, fakeClient.Create(context.Background(), col))
	require.NoError(t, fakeClient.Create(context.Background(), deploy))

	c := NewClient(bridgeName, clientLogger, fakeClient, nil)
	before := time.Now().Truncate(time.Second)
	require.NoError(t, c.Restart(context.Background()))

	result := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: workloadName, Namespace: "default"}, result))
	val := result.Spec.Template.Annotations[rollout.RestartAnnotation]
	assert.NotEmpty(t, val)
	parsed, err := time.Parse(time.RFC3339, val)
	require.NoError(t, err)
	assert.False(t, parsed.Before(before))
}

func TestClient_Restart_DaemonSet(t *testing.T) {
	col := managedCollector("test-col", "default", v1beta1.ModeDaemonSet)
	workloadName := naming.Collector(col.Name)
	ds := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: workloadName, Namespace: "default"}}

	fakeClient := getFakeClient(t)
	require.NoError(t, fakeClient.Create(context.Background(), col))
	require.NoError(t, fakeClient.Create(context.Background(), ds))

	c := NewClient(bridgeName, clientLogger, fakeClient, nil)
	require.NoError(t, c.Restart(context.Background()))

	result := &appsv1.DaemonSet{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: workloadName, Namespace: "default"}, result))
	assert.NotEmpty(t, result.Spec.Template.Annotations[rollout.RestartAnnotation])
}

func TestClient_Restart_StatefulSet(t *testing.T) {
	col := managedCollector("test-col", "default", v1beta1.ModeStatefulSet)
	workloadName := naming.Collector(col.Name)
	sts := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: workloadName, Namespace: "default"}}

	fakeClient := getFakeClient(t)
	require.NoError(t, fakeClient.Create(context.Background(), col))
	require.NoError(t, fakeClient.Create(context.Background(), sts))

	c := NewClient(bridgeName, clientLogger, fakeClient, nil)
	require.NoError(t, c.Restart(context.Background()))

	result := &appsv1.StatefulSet{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: workloadName, Namespace: "default"}, result))
	assert.NotEmpty(t, result.Spec.Template.Annotations[rollout.RestartAnnotation])
}

func TestClient_Restart_SidecarSkipped(t *testing.T) {
	col := managedCollector("test-col", "default", v1beta1.ModeSidecar)

	fakeClient := getFakeClient(t)
	require.NoError(t, fakeClient.Create(context.Background(), col))

	c := NewClient(bridgeName, clientLogger, fakeClient, nil)
	require.NoError(t, c.Restart(context.Background()))
}

func TestClient_Restart_NoCollectors(t *testing.T) {
	fakeClient := getFakeClient(t)
	c := NewClient(bridgeName, clientLogger, fakeClient, nil)
	require.NoError(t, c.Restart(context.Background()))
}

func TestClient_Restart_PartialFailure(t *testing.T) {
	// Two collectors: one with its workload present, one without.
	col1 := managedCollector("col-ok", "default", v1beta1.ModeDeployment)
	col2 := managedCollector("col-missing", "default", v1beta1.ModeDeployment)
	workload1 := naming.Collector(col1.Name)
	deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: workload1, Namespace: "default"}}

	fakeClient := getFakeClient(t)
	require.NoError(t, fakeClient.Create(context.Background(), col1))
	require.NoError(t, fakeClient.Create(context.Background(), col2))
	require.NoError(t, fakeClient.Create(context.Background(), deploy))

	c := NewClient(bridgeName, clientLogger, fakeClient, nil)
	err := c.Restart(context.Background())
	require.Error(t, err, "partial failure must be reported")

	// The workload that exists must still have been patched.
	result := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: workload1, Namespace: "default"}, result))
	assert.NotEmpty(t, result.Spec.Template.Annotations[rollout.RestartAnnotation])
}

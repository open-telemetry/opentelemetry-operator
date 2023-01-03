package operator

import (
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
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
	assert.NoError(t, err, "Should be able to add custom types")
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
			c := NewClient(clientLogger, fakeClient)
			var colConfig []byte
			var err error
			if len(tt.args.file) > 0 {
				colConfig, err = loadConfig(tt.args.file)
				assert.NoError(t, err, "Should be no error on loading test configuration")
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
	c := NewClient(clientLogger, fakeClient)
	colConfig, err := loadConfig("testdata/collector.yaml")
	assert.NoError(t, err, "Should be no error on loading test configuration")
	configmap := &protobufs.AgentConfigFile{
		Body:        colConfig,
		ContentType: "yaml",
	}
	// Apply a valid initial configuration
	err = c.Apply(name, namespace, configmap)
	assert.NoError(t, err, "Should apply base config")

	// Get the newly created collector
	instance, err := c.GetInstance(name, namespace)
	assert.NoError(t, err, "Should be able to get the newly created instance")
	assert.Contains(t, instance.Spec.Config, "processors: []")

	// Try updating with an invalid one
	configmap.Body = []byte("empty, invalid!")
	err = c.Apply(name, namespace, configmap)
	assert.Error(t, err, "Should be unable to update")

	// Update successfully with a valid configuration
	newColConfig, err := loadConfig("testdata/updated-collector.yaml")
	assert.NoError(t, err, "Should be no error on loading test configuration")
	newConfigMap := &protobufs.AgentConfigFile{
		Body:        newColConfig,
		ContentType: "yaml",
	}
	err = c.Apply(name, namespace, newConfigMap)
	assert.NoError(t, err, "Should be able to update collector")

	// Get the updated collector
	updatedInstance, err := c.GetInstance(name, namespace)
	assert.NoError(t, err, "Should be able to get the updated instance")
	assert.Contains(t, updatedInstance.Spec.Config, "processors: [memory_limiter, batch]")

	allInstances, err := c.ListInstances()
	assert.NoError(t, err, "Should be able to list all collectors")
	assert.Len(t, allInstances, 1)
	assert.Equal(t, allInstances[0], *updatedInstance)
}

func Test_collectorDelete(t *testing.T) {
	name := "test"
	namespace := "testing"
	fakeClient := getFakeClient(t)
	c := NewClient(clientLogger, fakeClient)
	colConfig, err := loadConfig("testdata/collector.yaml")
	assert.NoError(t, err, "Should be no error on loading test configuration")
	configmap := &protobufs.AgentConfigFile{
		Body:        colConfig,
		ContentType: "yaml",
	}
	// Apply a valid initial configuration
	err = c.Apply(name, namespace, configmap)
	assert.NoError(t, err, "Should apply base config")

	// Get the newly created collector
	instance, err := c.GetInstance(name, namespace)
	assert.NoError(t, err, "Should be able to get the newly created instance")
	assert.Contains(t, instance.Spec.Config, "processors: []")

	// Delete it
	err = c.Delete(name, namespace)
	assert.NoError(t, err, "Should be able to delete a collector")

	// Check there's nothing left
	allInstances, err := c.ListInstances()
	assert.NoError(t, err, "Should be able to list all collectors")
	assert.Len(t, allInstances, 0)
}

func loadConfig(file string) ([]byte, error) {
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return []byte{}, err
	}
	return yamlFile, nil
}

package operator

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	CollectorResource       = "OpenTelemetryCollector"
	ResourceIdentifierKey   = "created-by"
	ResourceIdentifierValue = "remote-configuration"
)

var (
	schemeBuilder = runtime.NewSchemeBuilder(registerKnownTypes)
)

func registerKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.OpenTelemetryCollector{}, &v1alpha1.OpenTelemetryCollectorList{})
	metav1.AddToGroupVersion(s, v1alpha1.GroupVersion)
	return nil
}

type ConfigApplier interface {
	Apply(name string, namespace string, configmap *protobufs.AgentConfigFile) error
	GetInstance(name string, namespace string) (*v1alpha1.OpenTelemetryCollector, error)
	ListInstances() ([]v1alpha1.OpenTelemetryCollector, error)
}

type Client struct {
	log       logr.Logger
	k8sClient client.Client
	close     chan bool
}

var _ ConfigApplier = &Client{}

func NewClient(log logr.Logger, kubeConfig *rest.Config) (*Client, error) {
	err := schemeBuilder.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}
	c, err := client.New(kubeConfig, client.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return nil, err
	}
	return &Client{
		log:       log,
		k8sClient: c,
		close:     make(chan bool, 1),
	}, nil
}

func (c Client) create(ctx context.Context, name string, namespace string, collector v1alpha1.OpenTelemetryCollector) error {
	// Set the defaults
	collector.Default()
	collector.TypeMeta.Kind = CollectorResource
	collector.TypeMeta.APIVersion = v1alpha1.GroupVersion.String()
	collector.ObjectMeta.Name = name
	collector.ObjectMeta.Namespace = namespace

	if collector.ObjectMeta.Labels == nil {
		collector.ObjectMeta.Labels = map[string]string{}
	}
	collector.ObjectMeta.Labels[ResourceIdentifierKey] = ResourceIdentifierValue
	err := collector.ValidateCreate()
	if err != nil {
		return err
	}
	c.log.Info("Was given a valid configuration", "collector", collector)
	err = c.k8sClient.Create(ctx, &collector)
	if err != nil {
		c.log.Error(err, "unable to create collector")
		return err
	}
	return nil
}

func (c Client) Apply(name string, namespace string, configmap *protobufs.AgentConfigFile) error {
	c.log.Info("Received new config", name, len(configmap.String()))
	var collectorSpec v1alpha1.OpenTelemetryCollectorSpec
	err := yaml.Unmarshal(configmap.Body, &collectorSpec)
	if err != nil {
		return err
	}
	collector := v1alpha1.OpenTelemetryCollector{Spec: collectorSpec}
	ctx := context.Background()
	//created := false
	instance, err := c.GetInstance(name, namespace)
	if err != nil {
		return err
	}
	if instance != nil {
		collector.ObjectMeta = instance.ObjectMeta
		collector.TypeMeta = instance.TypeMeta
		err := collector.ValidateUpdate(instance)
		if err != nil {
			return err
		}
		err = c.k8sClient.Update(ctx, &collector)
		if err != nil {
			return err
		}
		c.log.Info("Updated collector")
		return nil
	}

	err = c.create(ctx, name, namespace, collector)
	if err != nil {
		return err
	}
	c.log.Info("Created collector")
	return nil
}

func (c Client) ListInstances() ([]v1alpha1.OpenTelemetryCollector, error) {
	ctx := context.Background()
	result := v1alpha1.OpenTelemetryCollectorList{}
	err := c.k8sClient.List(ctx, &result, client.MatchingLabels{
		ResourceIdentifierKey: ResourceIdentifierValue,
	})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (c Client) GetInstance(name string, namespace string) (*v1alpha1.OpenTelemetryCollector, error) {
	ctx := context.Background()
	result := v1alpha1.OpenTelemetryCollector{}
	err := c.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &result)
	if errors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

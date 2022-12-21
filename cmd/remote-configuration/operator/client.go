package operator

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	CollectorResource       = "OpenTelemetryCollector"
	ResourceIdentifierKey   = "created-by"
	ResourceIdentifierValue = "remote-configuration"
)

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

func NewClient(log logr.Logger, c client.Client) *Client {
	return &Client{
		log:       log,
		k8sClient: c,
		close:     make(chan bool, 1),
	}
}

func (c Client) create(ctx context.Context, name string, namespace string, collector *v1alpha1.OpenTelemetryCollector) error {
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
	c.log.Info("Creating collector")
	return c.k8sClient.Create(ctx, collector)
}

func (c Client) update(ctx context.Context, old *v1alpha1.OpenTelemetryCollector, new *v1alpha1.OpenTelemetryCollector) error {
	new.ObjectMeta = old.ObjectMeta
	new.TypeMeta = old.TypeMeta
	err := new.ValidateUpdate(old)
	if err != nil {
		return err
	}
	c.log.Info("Updating collector")
	return c.k8sClient.Update(ctx, new)
}

func (c Client) Apply(name string, namespace string, configmap *protobufs.AgentConfigFile) error {
	c.log.Info("Received new config", "name", name, "namespace", namespace)
	var collectorSpec v1alpha1.OpenTelemetryCollectorSpec
	err := yaml.Unmarshal(configmap.Body, &collectorSpec)
	if err != nil {
		return err
	}
	if len(collectorSpec.Config) == 0 {
		return errors.NewBadRequest("Must supply valid configuration")
	}
	collector := &v1alpha1.OpenTelemetryCollector{Spec: collectorSpec}
	ctx := context.Background()
	instance, err := c.GetInstance(name, namespace)
	if err != nil {
		return err
	}
	if instance != nil {
		return c.update(ctx, instance, collector)
	}
	return c.create(ctx, name, namespace, collector)
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
